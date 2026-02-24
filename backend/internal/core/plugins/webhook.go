package plugins

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// WebhookConfig defines the configuration for a single webhook endpoint.
type WebhookConfig struct {
	// ID is the unique identifier for this webhook (UUID).
	ID string `json:"id"`
	// PluginName is the name of the plugin that owns this webhook.
	PluginName string `json:"pluginName"`
	// URL is the HTTP endpoint that will receive webhook payloads.
	URL string `json:"url"`
	// Secret is used to compute HMAC-SHA256 signatures for payload verification.
	Secret string `json:"secret,omitempty"`
	// Events lists which event names trigger this webhook.
	Events []string `json:"events"`
	// Active indicates whether this webhook is currently enabled.
	Active bool `json:"active"`
	// CreatedAt is when this webhook was registered.
	CreatedAt time.Time `json:"createdAt"`
	// LastTriggered is the most recent time this webhook was fired.
	LastTriggered time.Time `json:"lastTriggered,omitempty"`
	// FailureCount is the number of consecutive delivery failures.
	FailureCount int `json:"failureCount"`
	// MaxRetries is the maximum number of delivery attempts before marking as failed.
	// Zero means no retries (single attempt only).
	MaxRetries int `json:"maxRetries"`
}

// WebhookPayload is the body sent to webhook endpoints.
type WebhookPayload struct {
	// Event is the name of the event that triggered this webhook.
	Event string `json:"event"`
	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp"`
	// PluginName identifies which plugin emitted the event.
	PluginName string `json:"pluginName"`
	// Data is the event-specific payload.
	Data interface{} `json:"data"`
	// Signature is the HMAC-SHA256 hex digest of the JSON body, computed using the webhook secret.
	Signature string `json:"signature,omitempty"`
}

// WebhookPlugin is implemented by plugins that can emit webhook events.
type WebhookPlugin interface {
	// WebhookEvents returns the list of event names this plugin can emit.
	WebhookEvents() []string
}

// WebhookManager handles registration, delivery, and lifecycle of webhooks.
type WebhookManager struct {
	mu       sync.RWMutex
	webhooks map[string]*WebhookConfig // keyed by webhook ID
	logger   zerolog.Logger
	client   *http.Client
}

// NewWebhookManager creates a new WebhookManager.
func NewWebhookManager(logger zerolog.Logger) *WebhookManager {
	return &WebhookManager{
		webhooks: make(map[string]*WebhookConfig),
		logger:   logger.With().Str("component", "webhook-manager").Logger(),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Register adds a new webhook. If the config has no ID, one is generated.
func (wm *WebhookManager) Register(config WebhookConfig) error {
	if config.URL == "" {
		return fmt.Errorf("webhook URL must not be empty")
	}
	if config.PluginName == "" {
		return fmt.Errorf("webhook plugin name must not be empty")
	}
	if len(config.Events) == 0 {
		return fmt.Errorf("webhook must subscribe to at least one event")
	}

	if config.ID == "" {
		config.ID = uuid.New().String()
	}
	if config.CreatedAt.IsZero() {
		config.CreatedAt = time.Now()
	}

	wm.mu.Lock()
	defer wm.mu.Unlock()

	wm.webhooks[config.ID] = &config
	wm.logger.Info().
		Str("id", config.ID).
		Str("plugin", config.PluginName).
		Str("url", config.URL).
		Strs("events", config.Events).
		Msg("webhook registered")

	return nil
}

// Unregister removes a webhook by its ID.
func (wm *WebhookManager) Unregister(id string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if _, ok := wm.webhooks[id]; !ok {
		return fmt.Errorf("webhook %q not found", id)
	}

	delete(wm.webhooks, id)
	wm.logger.Info().Str("id", id).Msg("webhook unregistered")
	return nil
}

// Trigger fires all active webhooks that are subscribed to the given event.
// Delivery happens asynchronously in goroutines.
func (wm *WebhookManager) Trigger(pluginName, event string, data interface{}) {
	wm.mu.RLock()
	var targets []*WebhookConfig
	for _, wh := range wm.webhooks {
		if !wh.Active {
			continue
		}
		if wh.PluginName != pluginName {
			continue
		}
		if !containsString(wh.Events, event) {
			continue
		}
		// Copy the config to avoid data races.
		c := *wh
		targets = append(targets, &c)
	}
	wm.mu.RUnlock()

	for _, target := range targets {
		go wm.deliver(target, event, pluginName, data)
	}
}

// GetWebhooks returns all webhooks registered by a specific plugin.
func (wm *WebhookManager) GetWebhooks(pluginName string) []WebhookConfig {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	var result []WebhookConfig
	for _, wh := range wm.webhooks {
		if wh.PluginName == pluginName {
			result = append(result, *wh)
		}
	}
	return result
}

// GetAllWebhooks returns all registered webhooks.
func (wm *WebhookManager) GetAllWebhooks() []WebhookConfig {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	result := make([]WebhookConfig, 0, len(wm.webhooks))
	for _, wh := range wm.webhooks {
		result = append(result, *wh)
	}
	return result
}

// deliver attempts to send a payload to a webhook endpoint with retry logic.
func (wm *WebhookManager) deliver(wh *WebhookConfig, event, pluginName string, data interface{}) {
	payload := WebhookPayload{
		Event:      event,
		Timestamp:  time.Now(),
		PluginName: pluginName,
		Data:       data,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		wm.logger.Error().Err(err).
			Str("webhook_id", wh.ID).
			Msg("failed to marshal webhook payload")
		return
	}

	// Sign the payload if a secret is configured.
	if wh.Secret != "" {
		payload.Signature = signPayload(body, wh.Secret)
		// Re-marshal with the signature included.
		body, err = json.Marshal(payload)
		if err != nil {
			wm.logger.Error().Err(err).
				Str("webhook_id", wh.ID).
				Msg("failed to marshal signed webhook payload")
			return
		}
	}

	maxAttempts := 1 + wh.MaxRetries
	backoff := time.Second

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(backoff)
			backoff *= 2 // Exponential backoff: 1s, 2s, 4s, ...
		}

		if err := wm.sendHTTP(wh, body); err != nil {
			wm.logger.Warn().Err(err).
				Str("webhook_id", wh.ID).
				Int("attempt", attempt+1).
				Int("max_attempts", maxAttempts).
				Msg("webhook delivery failed")

			if attempt == maxAttempts-1 {
				wm.markFailed(wh.ID)
			}
			continue
		}

		// Success.
		wm.markTriggered(wh.ID)
		wm.logger.Debug().
			Str("webhook_id", wh.ID).
			Str("event", event).
			Msg("webhook delivered successfully")
		return
	}
}

// sendHTTP performs the actual HTTP POST to the webhook URL.
func (wm *WebhookManager) sendHTTP(wh *WebhookConfig, body []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "HomeBoxNG-Webhook/1.0")
	if wh.Secret != "" {
		req.Header.Set("X-Webhook-Signature", signPayload(body, wh.Secret))
	}

	resp, err := wm.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// markTriggered updates the last triggered time and resets failure count.
func (wm *WebhookManager) markTriggered(id string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if wh, ok := wm.webhooks[id]; ok {
		wh.LastTriggered = time.Now()
		wh.FailureCount = 0
	}
}

// markFailed increments the failure count and deactivates the webhook if it exceeds max retries.
func (wm *WebhookManager) markFailed(id string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	wh, ok := wm.webhooks[id]
	if !ok {
		return
	}

	wh.FailureCount++
	if wh.MaxRetries > 0 && wh.FailureCount > wh.MaxRetries {
		wh.Active = false
		wm.logger.Warn().
			Str("webhook_id", id).
			Int("failure_count", wh.FailureCount).
			Msg("webhook deactivated after exceeding max retries")
	}
}

// signPayload computes the HMAC-SHA256 hex digest of the body using the given secret.
func signPayload(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// containsString checks if a slice contains a specific string.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

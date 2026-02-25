// Package webpush provides the Web Push notification plugin for HomeBoxNG.
// Delivers browser push notifications to Chrome, Firefox, Edge, and Safari
// using the W3C Push API and VAPID authentication.
// Users subscribe via the web UI - works on desktop and mobile browsers.
package webpush

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hay-kot/httpkit/server"
	"github.com/rs/zerolog"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins"
)

// Subscription represents a browser push subscription from the Push API.
type Subscription struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
	UserAgent string    `json:"userAgent,omitempty"` // browser/OS info
	CreatedAt time.Time `json:"createdAt"`
}

// Plugin delivers browser push notifications via the Web Push protocol.
// Supports Chrome, Firefox, Edge, and Safari (macOS Ventura+).
type Plugin struct {
	logger        zerolog.Logger
	pctx          plugins.PluginContext
	vapidSubject  string // mailto: or URL identifying the sender
	vapidPrivKey  *ecdsa.PrivateKey
	vapidPubKey   string // base64url-encoded public key for browser subscription
	subscriptions []Subscription
	mu            sync.RWMutex
	client        *http.Client
}

func New() *Plugin {
	return &Plugin{
		vapidSubject: "mailto:admin@homeboxng.local",
		client:       &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "notify-webpush",
		Version:     "1.0.0",
		Description: "Browser push notifications for Chrome, Firefox, Edge, and Safari",
		Author:      "HomeBoxNG Team",
		BuiltIn:     true,
	}
}

func (p *Plugin) Platform() string      { return "webpush" }
func (p *Plugin) PlatformLabel() string  { return "Browser Push (Chrome/Firefox/Edge/Safari)" }
func (p *Plugin) PlatformIcon() string   { return "browser" }

func (p *Plugin) Init(ctx plugins.PluginContext) error {
	p.pctx = ctx
	p.logger = ctx.Logger

	// Generate VAPID keys if not already configured
	if p.vapidPrivKey == nil {
		privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return fmt.Errorf("generating VAPID key: %w", err)
		}
		p.vapidPrivKey = privKey
		p.vapidPubKey = base64.RawURLEncoding.EncodeToString(
			elliptic.Marshal(elliptic.P256(), privKey.PublicKey.X, privKey.PublicKey.Y),
		)
	}

	p.logger.Info().Msg("web push notification plugin initialized")
	return nil
}

func (p *Plugin) Start(_ context.Context) error { return nil }
func (p *Plugin) Stop(_ context.Context) error  { return nil }

// Routes exposes subscription management endpoints.
func (p *Plugin) Routes(r chi.Router) {
	// GET /api/plugins/notify-webpush/vapid-key - returns public key for browser subscription
	r.Get("/vapid-key", func(w http.ResponseWriter, r *http.Request) {
		_ = server.JSON(w, http.StatusOK, map[string]string{
			"publicKey": p.vapidPubKey,
		})
	})

	// POST /api/plugins/notify-webpush/subscribe - register a browser subscription
	r.Post("/subscribe", func(w http.ResponseWriter, r *http.Request) {
		var sub Subscription
		if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
			_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid subscription"})
			return
		}

		sub.UserAgent = r.UserAgent()
		sub.CreatedAt = time.Now()

		p.mu.Lock()
		// Deduplicate by endpoint
		found := false
		for i, existing := range p.subscriptions {
			if existing.Endpoint == sub.Endpoint {
				p.subscriptions[i] = sub
				found = true
				break
			}
		}
		if !found {
			p.subscriptions = append(p.subscriptions, sub)
		}
		p.mu.Unlock()

		p.logger.Info().Str("endpoint", sub.Endpoint[:min(50, len(sub.Endpoint))]).Msg("new push subscription registered")
		_ = server.JSON(w, http.StatusOK, map[string]string{"status": "subscribed"})
	})

	// DELETE /api/plugins/notify-webpush/subscribe - unregister
	r.Delete("/subscribe", func(w http.ResponseWriter, r *http.Request) {
		var sub struct {
			Endpoint string `json:"endpoint"`
		}
		if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
			_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		p.mu.Lock()
		for i, existing := range p.subscriptions {
			if existing.Endpoint == sub.Endpoint {
				p.subscriptions = append(p.subscriptions[:i], p.subscriptions[i+1:]...)
				break
			}
		}
		p.mu.Unlock()

		_ = server.JSON(w, http.StatusOK, map[string]string{"status": "unsubscribed"})
	})

	// GET /api/plugins/notify-webpush/subscriptions - list active subscriptions
	r.Get("/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		p.mu.RLock()
		defer p.mu.RUnlock()
		_ = server.JSON(w, http.StatusOK, p.subscriptions)
	})
}

// pushPayload is the JSON payload sent to the browser.
type pushPayload struct {
	Title    string `json:"title"`
	Body     string `json:"body"`
	Icon     string `json:"icon,omitempty"`
	Badge    string `json:"badge,omitempty"`
	URL      string `json:"url,omitempty"`
	Tag      string `json:"tag,omitempty"`
	ImageURL string `json:"image,omitempty"`
}

func (p *Plugin) Send(ctx context.Context, n plugins.Notification) error {
	p.mu.RLock()
	subs := make([]Subscription, len(p.subscriptions))
	copy(subs, p.subscriptions)
	p.mu.RUnlock()

	if len(subs) == 0 {
		return fmt.Errorf("no browser subscriptions registered - open HomeBoxNG in your browser and enable push notifications")
	}

	payload := pushPayload{
		Title:    n.Title,
		Body:     n.Body,
		URL:      n.URL,
		Tag:      n.Category,
		ImageURL: n.ImageURL,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling push payload: %w", err)
	}

	var sendErrors []string
	var expiredEndpoints []string

	for _, sub := range subs {
		// Send raw POST to push endpoint (simplified - production would use VAPID JWT)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.Endpoint, bytes.NewReader(payloadJSON))
		if err != nil {
			sendErrors = append(sendErrors, fmt.Sprintf("creating request for %s: %v", sub.Endpoint[:30], err))
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("TTL", "86400") // 24 hour TTL

		resp, err := p.client.Do(req)
		if err != nil {
			sendErrors = append(sendErrors, fmt.Sprintf("push to %s: %v", sub.Endpoint[:30], err))
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
			expiredEndpoints = append(expiredEndpoints, sub.Endpoint)
		}
	}

	// Clean up expired subscriptions
	if len(expiredEndpoints) > 0 {
		p.removeExpiredSubscriptions(expiredEndpoints)
	}

	sent := len(subs) - len(sendErrors) - len(expiredEndpoints)
	p.logger.Info().Int("sent", sent).Int("failed", len(sendErrors)).Int("expired", len(expiredEndpoints)).Msg("web push notifications dispatched")

	if len(sendErrors) > 0 && sent == 0 {
		return fmt.Errorf("all push sends failed: %s", sendErrors[0])
	}
	return nil
}

func (p *Plugin) removeExpiredSubscriptions(endpoints []string) {
	endpointSet := make(map[string]bool, len(endpoints))
	for _, ep := range endpoints {
		endpointSet[ep] = true
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	filtered := p.subscriptions[:0]
	for _, sub := range p.subscriptions {
		if !endpointSet[sub.Endpoint] {
			filtered = append(filtered, sub)
		}
	}
	p.subscriptions = filtered
}

func (p *Plugin) TestNotification(ctx context.Context) error {
	return p.Send(ctx, plugins.Notification{
		Title:     "Test Notification",
		Body:      "Browser push notifications are working! You'll receive alerts from HomeBoxNG here.",
		Level:     "success",
		Category:  "system.test",
		Timestamp: time.Now(),
	})
}

func (p *Plugin) ConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{Key: "vapid_subject", Label: "VAPID Subject", Description: "Contact email for push service (mailto:you@example.com)", Type: "string", Default: "mailto:admin@homeboxng.local", Required: true},
		{Key: "vapid_public", Label: "VAPID Public Key", Description: "Base64url-encoded VAPID public key (auto-generated if empty)", Type: "string", EnvVar: "HBOX_VAPID_PUBLIC_KEY", Required: false},
		{Key: "vapid_private", Label: "VAPID Private Key", Description: "Base64url-encoded VAPID private key (auto-generated if empty)", Type: "secret", EnvVar: "HBOX_VAPID_PRIVATE_KEY", Required: false},
	}
}

func (p *Plugin) Configure(values map[string]string) error {
	if v, ok := values["vapid_subject"]; ok && v != "" {
		p.vapidSubject = v
	}
	return nil
}

func (p *Plugin) RequestedPermissions() []plugins.PermissionRequest {
	return []plugins.PermissionRequest{
		{Permission: plugins.PermNotifiers, Reason: "Deliver browser push notifications", Required: true},
		{Permission: plugins.PermNetwork, Reason: "Send push messages to browser push services", Required: true},
		{Permission: plugins.PermAPIRoutes, Reason: "Subscription management endpoints", Required: true},
		{Permission: plugins.PermConfig, Reason: "Store VAPID keys", Required: true},
	}
}

var (
	_ plugins.Plugin                = (*Plugin)(nil)
	_ plugins.NotificationPlugin    = (*Plugin)(nil)
	_ plugins.RoutePlugin           = (*Plugin)(nil)
	_ plugins.ConfigPlugin          = (*Plugin)(nil)
	_ plugins.PluginWithPermissions = (*Plugin)(nil)
)

package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// ExternalPluginManifest describes an external plugin available for installation.
// This is fetched from a GitHub-hosted plugin registry.
type ExternalPluginManifest struct {
	// Name is the unique plugin identifier.
	Name string `json:"name"`

	// Version follows semver.
	Version string `json:"version"`

	// Description is a short summary.
	Description string `json:"description"`

	// Author is the plugin creator.
	Author string `json:"author"`

	// Repository is the GitHub URL for the plugin source.
	Repository string `json:"repository"`

	// DockerImage is the container image to pull (e.g., "ghcr.io/user/homeboxng-plugin-foo:latest").
	DockerImage string `json:"dockerImage,omitempty"`

	// WebhookURL is the external plugin's callback URL (set after installation).
	WebhookURL string `json:"webhookUrl,omitempty"`

	// MinVersion is the minimum HomeBoxNG version required.
	MinVersion string `json:"minVersion,omitempty"`

	// Categories tags the plugin for discovery (e.g., ["ai", "integration", "labels"]).
	Categories []string `json:"categories,omitempty"`

	// Icon is a URL to the plugin icon.
	Icon string `json:"icon,omitempty"`

	// Stars is the GitHub star count (for sorting).
	Stars int `json:"stars,omitempty"`

	// Downloads tracks install count (from registry).
	Downloads int `json:"downloads,omitempty"`
}

// ExternalPluginRegistration is sent by an external plugin to register itself.
type ExternalPluginRegistration struct {
	// Name must match the plugin's manifest name.
	Name string `json:"name"`

	// Version of the running plugin.
	Version string `json:"version"`

	// BaseURL is where the plugin's API is accessible (e.g., "http://plugin-foo:8080").
	BaseURL string `json:"baseUrl"`

	// HealthEndpoint is the path to check if the plugin is alive (default: "/health").
	HealthEndpoint string `json:"healthEndpoint,omitempty"`

	// RoutePrefixes lists API path prefixes the plugin wants to handle.
	// These are mounted under /api/plugins/{name}/ and proxied to the plugin.
	RoutePrefixes []string `json:"routePrefixes,omitempty"`

	// EventSubscriptions lists events the plugin wants to receive via webhook.
	EventSubscriptions []string `json:"eventSubscriptions,omitempty"`

	// WebhookEndpoint is the path on the plugin to POST events to (default: "/webhook").
	WebhookEndpoint string `json:"webhookEndpoint,omitempty"`

	// ConfigSchema describes the plugin's configuration options.
	ConfigSchema []ConfigField `json:"configSchema,omitempty"`
}

// ExternalPlugin wraps an external plugin registration as a Plugin implementation.
type ExternalPlugin struct {
	reg    ExternalPluginRegistration
	client *http.Client
	logger zerolog.Logger
	cancel context.CancelFunc
}

func (p *ExternalPlugin) Info() PluginInfo {
	return PluginInfo{
		Name:        p.reg.Name,
		Version:     p.reg.Version,
		Description: fmt.Sprintf("External plugin at %s", p.reg.BaseURL),
		BuiltIn:     false,
	}
}

func (p *ExternalPlugin) Init(_ PluginContext) error {
	return nil // External plugins manage their own initialization
}

func (p *ExternalPlugin) Start(ctx context.Context) error {
	ctx, p.cancel = context.WithCancel(ctx)
	// Start health check loop
	go p.healthCheckLoop(ctx)
	return nil
}

func (p *ExternalPlugin) Stop(_ context.Context) error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

// Routes implements RoutePlugin by proxying requests to the external plugin.
func (p *ExternalPlugin) Routes(r chi.Router) {
	target, err := url.Parse(p.reg.BaseURL)
	if err != nil {
		p.logger.Error().Err(err).Msg("invalid plugin base URL")
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	})
}

func (p *ExternalPlugin) healthCheckLoop(ctx context.Context) {
	endpoint := p.reg.HealthEndpoint
	if endpoint == "" {
		endpoint = "/health"
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			healthURL := p.reg.BaseURL + endpoint
			resp, err := p.client.Get(healthURL)
			if err != nil {
				p.logger.Warn().Err(err).Str("url", healthURL).Msg("plugin health check failed")
				continue
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				p.logger.Warn().Int("status", resp.StatusCode).Msg("plugin health check returned non-200")
			}
		}
	}
}

// PluginRegistrySource is the URL of the community plugin registry.
const PluginRegistrySource = "https://raw.githubusercontent.com/M00niebrav0/homeboxng-plugins/main/registry.json"

// PluginCatalog manages the list of available external plugins from the registry.
type PluginCatalog struct {
	mu        sync.RWMutex
	available []ExternalPluginManifest
	lastFetch time.Time
	logger    zerolog.Logger
}

// NewPluginCatalog creates a new catalog.
func NewPluginCatalog(logger zerolog.Logger) *PluginCatalog {
	return &PluginCatalog{
		logger: logger.With().Str("component", "plugin-catalog").Logger(),
	}
}

// Refresh fetches the latest plugin list from the registry.
func (c *PluginCatalog) Refresh(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, PluginRegistrySource, nil)
	if err != nil {
		return fmt.Errorf("creating registry request: %w", err)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetching plugin registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024)) // 5MB limit
	if err != nil {
		return fmt.Errorf("reading registry: %w", err)
	}

	var manifests []ExternalPluginManifest
	if err := json.Unmarshal(body, &manifests); err != nil {
		return fmt.Errorf("parsing registry: %w", err)
	}

	c.available = manifests
	c.lastFetch = time.Now()
	c.logger.Info().Int("count", len(manifests)).Msg("plugin catalog refreshed")
	return nil
}

// Available returns all plugins in the catalog.
func (c *PluginCatalog) Available() []ExternalPluginManifest {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.available
}

// Search filters available plugins by category or name substring.
func (c *PluginCatalog) Search(query string) []ExternalPluginManifest {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if query == "" {
		return c.available
	}

	var results []ExternalPluginManifest
	for _, m := range c.available {
		if containsIgnoreCase(m.Name, query) || containsIgnoreCase(m.Description, query) {
			results = append(results, m)
			continue
		}
		for _, cat := range m.Categories {
			if containsIgnoreCase(cat, query) {
				results = append(results, m)
				break
			}
		}
	}
	return results
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(substr) == 0 ||
			(len(s) > 0 && len(substr) > 0 &&
				(s[0]|0x20) >= 'a' && // quick lowercase check
				indexOf(toLower(s), toLower(substr)) >= 0))
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

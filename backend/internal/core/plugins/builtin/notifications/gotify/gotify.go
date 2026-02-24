// Package gotify provides the Gotify notification plugin for HomeBoxNG.
// Gotify is a self-hosted push notification server that delivers to Android,
// Windows, Linux, and any browser. Users who prefer not to use Google/Apple
// push services can use Gotify as an open-source alternative.
//
// Supported platforms via Gotify:
// - Android (Gotify app or UnifiedPush)
// - Windows (Gotify desktop client or browser)
// - Linux (Gotify desktop client or browser)
// - iOS (via UnifiedPush + ntfy bridge)
package gotify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins"
)

// Plugin delivers notifications via a Gotify server.
type Plugin struct {
	logger   zerolog.Logger
	pctx     plugins.PluginContext
	serverURL string
	appToken  string
	priority  int
	client    *http.Client
}

func New() *Plugin {
	return &Plugin{
		priority: 5,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "notify-gotify",
		Version:     "1.0.0",
		Description: "Gotify push notifications - self-hosted, supports Android, Windows, Linux, iOS (via UnifiedPush)",
		Author:      "HomeBoxNG Team",
		BuiltIn:     true,
	}
}

func (p *Plugin) Platform() string      { return "gotify" }
func (p *Plugin) PlatformLabel() string  { return "Gotify (Android/Windows/Linux/iOS)" }
func (p *Plugin) PlatformIcon() string   { return "gotify" }

func (p *Plugin) Init(ctx plugins.PluginContext) error {
	p.pctx = ctx
	p.logger = ctx.Logger
	p.logger.Info().Msg("gotify notification plugin initialized")
	return nil
}

func (p *Plugin) Start(_ context.Context) error { return nil }
func (p *Plugin) Stop(_ context.Context) error  { return nil }

// gotifyMessage is the Gotify REST API message format.
type gotifyMessage struct {
	Title    string            `json:"title"`
	Message  string            `json:"message"`
	Priority int               `json:"priority"`
	Extras   map[string]any    `json:"extras,omitempty"`
}

func (p *Plugin) Send(ctx context.Context, n plugins.Notification) error {
	if p.serverURL == "" || p.appToken == "" {
		return fmt.Errorf("gotify not configured: set server URL and app token")
	}

	priority := p.levelToPriority(n.Level)

	msg := gotifyMessage{
		Title:    n.Title,
		Message:  n.Body,
		Priority: priority,
	}

	// Add click URL as extra if provided
	if n.URL != "" {
		msg.Extras = map[string]any{
			"client::notification": map[string]any{
				"click": map[string]string{"url": n.URL},
			},
		}
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling gotify message: %w", err)
	}

	url := fmt.Sprintf("%s/message?token=%s", p.serverURL, p.appToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating gotify request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending gotify message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gotify returned status %d", resp.StatusCode)
	}

	p.logger.Info().Str("title", n.Title).Int("priority", priority).Msg("gotify notification sent")
	return nil
}

func (p *Plugin) levelToPriority(level string) int {
	switch level {
	case "error":
		return 8
	case "warning":
		return 6
	case "success":
		return 4
	case "info":
		return 3
	default:
		return p.priority
	}
}

func (p *Plugin) TestNotification(ctx context.Context) error {
	return p.Send(ctx, plugins.Notification{
		Title:     "Test Notification",
		Body:      "Gotify push notifications are working! You'll receive HomeBoxNG alerts on all your connected devices.",
		Level:     "success",
		Category:  "system.test",
		Timestamp: time.Now(),
	})
}

func (p *Plugin) ConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{Key: "server_url", Label: "Gotify Server URL", Description: "URL of your Gotify server (e.g., https://gotify.example.com)", Type: "string", Required: true},
		{Key: "app_token", Label: "Application Token", Description: "Gotify application token (create in Gotify web UI > Apps)", Type: "secret", Required: true},
		{Key: "default_priority", Label: "Default Priority", Description: "Default message priority (1-10, higher = more urgent)", Type: "number", Default: "5", Required: false},
	}
}

func (p *Plugin) Configure(values map[string]string) error {
	if v, ok := values["server_url"]; ok {
		p.serverURL = v
	}
	if v, ok := values["app_token"]; ok {
		p.appToken = v
	}
	if v, ok := values["default_priority"]; ok && v != "" {
		if pri, err := strconv.Atoi(v); err == nil && pri >= 1 && pri <= 10 {
			p.priority = pri
		}
	}
	return nil
}

func (p *Plugin) RequestedPermissions() []plugins.PermissionRequest {
	return []plugins.PermissionRequest{
		{Permission: plugins.PermNotifiers, Reason: "Deliver push notifications via Gotify", Required: true},
		{Permission: plugins.PermNetwork, Reason: "Connect to Gotify server", Required: true},
		{Permission: plugins.PermConfig, Reason: "Store Gotify server URL and token", Required: true},
	}
}

var (
	_ plugins.Plugin                = (*Plugin)(nil)
	_ plugins.NotificationPlugin    = (*Plugin)(nil)
	_ plugins.ConfigPlugin          = (*Plugin)(nil)
	_ plugins.PluginWithPermissions = (*Plugin)(nil)
)

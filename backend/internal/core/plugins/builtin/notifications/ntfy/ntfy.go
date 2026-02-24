// Package ntfy provides the ntfy.sh notification plugin for HomeBoxNG.
// ntfy is a simple HTTP-based pub-sub notification service.
// Works with ntfy.sh (public) or self-hosted ntfy servers.
//
// Supported platforms:
// - Android (ntfy app, F-Droid or Play Store)
// - iOS (ntfy app)
// - Windows/Linux/macOS (browser or desktop via web push)
// - Any device with a browser
package ntfy

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins"
)

// Plugin delivers notifications via ntfy.sh or a self-hosted ntfy server.
type Plugin struct {
	logger    zerolog.Logger
	pctx      plugins.PluginContext
	serverURL string
	topic     string
	token     string // optional access token for private topics
	client    *http.Client
}

func New() *Plugin {
	return &Plugin{
		serverURL: "https://ntfy.sh",
		topic:     "homeboxng",
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "notify-ntfy",
		Version:     "1.0.0",
		Description: "ntfy.sh push notifications - works on Android, iOS, Windows, Linux, macOS, and browsers",
		Author:      "HomeBoxNG Team",
		BuiltIn:     true,
	}
}

func (p *Plugin) Platform() string      { return "ntfy" }
func (p *Plugin) PlatformLabel() string  { return "ntfy (Android/iOS/Desktop/Browser)" }
func (p *Plugin) PlatformIcon() string   { return "ntfy" }

func (p *Plugin) Init(ctx plugins.PluginContext) error {
	p.pctx = ctx
	p.logger = ctx.Logger
	p.logger.Info().Msg("ntfy notification plugin initialized")
	return nil
}

func (p *Plugin) Start(_ context.Context) error { return nil }
func (p *Plugin) Stop(_ context.Context) error  { return nil }

func (p *Plugin) Send(ctx context.Context, n plugins.Notification) error {
	if p.topic == "" {
		return fmt.Errorf("ntfy topic not configured")
	}

	url := fmt.Sprintf("%s/%s", p.serverURL, p.topic)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(n.Body))
	if err != nil {
		return fmt.Errorf("creating ntfy request: %w", err)
	}

	req.Header.Set("Title", n.Title)
	req.Header.Set("Priority", p.levelToPriority(n.Level))
	req.Header.Set("Tags", p.levelToTag(n.Level))

	if n.URL != "" {
		req.Header.Set("Click", n.URL)
	}
	if n.ImageURL != "" {
		req.Header.Set("Attach", n.ImageURL)
	}

	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending ntfy message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy returned status %d", resp.StatusCode)
	}

	p.logger.Info().Str("title", n.Title).Str("topic", p.topic).Msg("ntfy notification sent")
	return nil
}

func (p *Plugin) levelToPriority(level string) string {
	switch level {
	case "error":
		return "urgent"
	case "warning":
		return "high"
	case "success":
		return "default"
	case "info":
		return "low"
	default:
		return "default"
	}
}

func (p *Plugin) levelToTag(level string) string {
	switch level {
	case "error":
		return "rotating_light"
	case "warning":
		return "warning"
	case "success":
		return "white_check_mark"
	case "info":
		return "information_source"
	default:
		return "package"
	}
}

func (p *Plugin) TestNotification(ctx context.Context) error {
	return p.Send(ctx, plugins.Notification{
		Title:     "Test Notification",
		Body:      "ntfy push notifications are working! You'll receive HomeBoxNG alerts on all subscribed devices.",
		Level:     "success",
		Category:  "system.test",
		Timestamp: time.Now(),
	})
}

func (p *Plugin) ConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{Key: "server_url", Label: "ntfy Server URL", Description: "ntfy server URL (default: https://ntfy.sh for the public server)", Type: "string", Default: "https://ntfy.sh", Required: true},
		{Key: "topic", Label: "Topic", Description: "Notification topic name (choose something unique like homeboxng-yourname)", Type: "string", Default: "homeboxng", Required: true},
		{Key: "access_token", Label: "Access Token", Description: "Optional access token for private topics (leave empty for public topics)", Type: "secret", Required: false},
	}
}

func (p *Plugin) Configure(values map[string]string) error {
	if v, ok := values["server_url"]; ok && v != "" {
		p.serverURL = v
	}
	if v, ok := values["topic"]; ok && v != "" {
		p.topic = v
	}
	if v, ok := values["access_token"]; ok {
		p.token = v
	}
	return nil
}

func (p *Plugin) RequestedPermissions() []plugins.PermissionRequest {
	return []plugins.PermissionRequest{
		{Permission: plugins.PermNotifiers, Reason: "Deliver push notifications via ntfy", Required: true},
		{Permission: plugins.PermNetwork, Reason: "Connect to ntfy server", Required: true},
		{Permission: plugins.PermConfig, Reason: "Store ntfy server URL, topic, and token", Required: true},
	}
}

var (
	_ plugins.Plugin                = (*Plugin)(nil)
	_ plugins.NotificationPlugin    = (*Plugin)(nil)
	_ plugins.ConfigPlugin          = (*Plugin)(nil)
	_ plugins.PluginWithPermissions = (*Plugin)(nil)
)

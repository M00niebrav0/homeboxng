// Package discord provides the Discord notification plugin for HomeBoxNG.
// Sends notifications via Discord webhooks to configured channels.
package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins"
)

// Plugin delivers notifications via Discord webhooks.
type Plugin struct {
	logger     zerolog.Logger
	pctx       plugins.PluginContext
	webhookURL string
	username   string
	avatarURL  string
	client     *http.Client
}

func New() *Plugin {
	return &Plugin{
		username:  "HomeBoxNG",
		avatarURL: "",
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "notify-discord",
		Version:     "1.0.0",
		Description: "Discord notifications via webhooks - send alerts to any Discord channel",
		Author:      "HomeBoxNG Team",
		BuiltIn:     true,
	}
}

func (p *Plugin) Platform() string      { return "discord" }
func (p *Plugin) PlatformLabel() string  { return "Discord" }
func (p *Plugin) PlatformIcon() string   { return "discord" }

func (p *Plugin) Init(ctx plugins.PluginContext) error {
	p.pctx = ctx
	p.logger = ctx.Logger
	p.logger.Info().Msg("discord notification plugin initialized")
	return nil
}

func (p *Plugin) Start(_ context.Context) error { return nil }
func (p *Plugin) Stop(_ context.Context) error  { return nil }

// discordWebhookPayload is the Discord webhook message format.
type discordWebhookPayload struct {
	Username  string         `json:"username,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	Embeds    []discordEmbed `json:"embeds"`
}

type discordEmbed struct {
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Color       int                 `json:"color"`
	URL         string              `json:"url,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
	Footer      *discordEmbedFooter `json:"footer,omitempty"`
	Thumbnail   *discordEmbedImage  `json:"thumbnail,omitempty"`
}

type discordEmbedFooter struct {
	Text string `json:"text"`
}

type discordEmbedImage struct {
	URL string `json:"url"`
}

func (p *Plugin) Send(ctx context.Context, n plugins.Notification) error {
	if p.webhookURL == "" {
		return fmt.Errorf("discord webhook URL not configured")
	}

	color := map[string]int{
		"info":    0x4FC3F7, // light blue
		"success": 0x66BB6A, // green
		"warning": 0xFFA726, // orange
		"error":   0xEF5350, // red
	}

	embedColor, ok := color[n.Level]
	if !ok {
		embedColor = 0x4FC3F7
	}

	embed := discordEmbed{
		Title:       n.Title,
		Description: n.Body,
		Color:       embedColor,
		URL:         n.URL,
		Timestamp:   n.Timestamp.Format(time.RFC3339),
		Footer:      &discordEmbedFooter{Text: n.Category},
	}

	if n.ImageURL != "" {
		embed.Thumbnail = &discordEmbedImage{URL: n.ImageURL}
	}

	payload := discordWebhookPayload{
		Username:  p.username,
		AvatarURL: p.avatarURL,
		Embeds:    []discordEmbed{embed},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling discord payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending discord webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook returned status %d", resp.StatusCode)
	}

	p.logger.Info().Str("title", n.Title).Msg("discord notification sent")
	return nil
}

func (p *Plugin) TestNotification(ctx context.Context) error {
	return p.Send(ctx, plugins.Notification{
		Title:     "Test Notification",
		Body:      "This is a test message from HomeBoxNG. If you see this, Discord notifications are working correctly!",
		Level:     "success",
		Category:  "system.test",
		Timestamp: time.Now(),
	})
}

func (p *Plugin) ConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{Key: "webhook_url", Label: "Webhook URL", Description: "Discord webhook URL (Server Settings > Integrations > Webhooks)", Type: "secret", Required: true},
		{Key: "username", Label: "Bot Username", Description: "Display name for the webhook messages", Type: "string", Default: "HomeBoxNG", Required: false},
		{Key: "avatar_url", Label: "Avatar URL", Description: "URL to an image for the webhook avatar", Type: "string", Required: false},
	}
}

func (p *Plugin) Configure(values map[string]string) error {
	if v, ok := values["webhook_url"]; ok {
		p.webhookURL = v
	}
	if v, ok := values["username"]; ok && v != "" {
		p.username = v
	}
	if v, ok := values["avatar_url"]; ok {
		p.avatarURL = v
	}
	return nil
}

func (p *Plugin) RequestedPermissions() []plugins.PermissionRequest {
	return []plugins.PermissionRequest{
		{Permission: plugins.PermNotifiers, Reason: "Deliver Discord notifications", Required: true},
		{Permission: plugins.PermNetwork, Reason: "Send messages to Discord webhook", Required: true},
		{Permission: plugins.PermConfig, Reason: "Store Discord webhook URL", Required: true},
	}
}

var (
	_ plugins.Plugin                = (*Plugin)(nil)
	_ plugins.NotificationPlugin    = (*Plugin)(nil)
	_ plugins.ConfigPlugin          = (*Plugin)(nil)
	_ plugins.PluginWithPermissions = (*Plugin)(nil)
)

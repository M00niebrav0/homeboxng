// Package email provides the Email notification plugin for HomeBoxNG.
// Sends notifications via SMTP to configured email addresses.
package email

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/rs/zerolog"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins"
)

// Plugin delivers notifications via SMTP email.
type Plugin struct {
	logger zerolog.Logger
	pctx   plugins.PluginContext

	smtpHost string
	smtpPort string
	username string
	password string
	fromAddr string
	toAddrs  []string
}

func New() *Plugin {
	return &Plugin{
		smtpPort: "587",
	}
}

func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "notify-email",
		Version:     "1.0.0",
		Description: "Email notifications via SMTP (Gmail, Outlook, custom SMTP servers)",
		Author:      "HomeBoxNG Team",
		BuiltIn:     true,
	}
}

func (p *Plugin) Platform() string      { return "email" }
func (p *Plugin) PlatformLabel() string  { return "Email" }
func (p *Plugin) PlatformIcon() string   { return "email" }

func (p *Plugin) Init(ctx plugins.PluginContext) error {
	p.pctx = ctx
	p.logger = ctx.Logger
	p.logger.Info().Msg("email notification plugin initialized")
	return nil
}

func (p *Plugin) Start(_ context.Context) error { return nil }
func (p *Plugin) Stop(_ context.Context) error  { return nil }

func (p *Plugin) Send(ctx context.Context, n plugins.Notification) error {
	if p.smtpHost == "" || len(p.toAddrs) == 0 {
		return fmt.Errorf("email not configured: set SMTP host and recipient addresses")
	}

	subject := fmt.Sprintf("[HomeBoxNG] %s", n.Title)
	body := p.buildBody(n)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		p.fromAddr,
		strings.Join(p.toAddrs, ", "),
		subject,
		body,
	)

	addr := fmt.Sprintf("%s:%s", p.smtpHost, p.smtpPort)
	auth := smtp.PlainAuth("", p.username, p.password, p.smtpHost)

	if err := smtp.SendMail(addr, auth, p.fromAddr, p.toAddrs, []byte(msg)); err != nil {
		return fmt.Errorf("sending email: %w", err)
	}

	p.logger.Info().Str("to", strings.Join(p.toAddrs, ", ")).Str("subject", subject).Msg("email notification sent")
	return nil
}

func (p *Plugin) buildBody(n plugins.Notification) string {
	levelColor := map[string]string{
		"info":    "#4fc3f7",
		"success": "#66bb6a",
		"warning": "#ffa726",
		"error":   "#ef5350",
	}
	color, ok := levelColor[n.Level]
	if !ok {
		color = "#4fc3f7"
	}

	linkHTML := ""
	if n.URL != "" {
		linkHTML = fmt.Sprintf(`<p><a href="%s" style="color: %s; text-decoration: underline;">View in HomeBoxNG</a></p>`, n.URL, color)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html><body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #1a1a2e; color: #e0e0e0; padding: 20px;">
<div style="max-width: 600px; margin: 0 auto; background: #16213e; border-radius: 12px; overflow: hidden;">
<div style="background: %s; padding: 16px 24px;">
<h2 style="margin: 0; color: #fff; font-size: 18px;">%s</h2>
</div>
<div style="padding: 24px;">
<p style="margin: 0 0 16px; line-height: 1.6;">%s</p>
%s
<p style="margin: 16px 0 0; color: #666; font-size: 12px;">%s &bull; %s</p>
</div>
</div>
</body></html>`, color, n.Title, n.Body, linkHTML, n.Category, n.Timestamp.Format("2006-01-02 15:04"))
}

func (p *Plugin) TestNotification(ctx context.Context) error {
	return p.Send(ctx, plugins.Notification{
		Title:    "Test Notification",
		Body:     "This is a test email from HomeBoxNG. If you received this, email notifications are working correctly.",
		Level:    "info",
		Category: "system.test",
	})
}

func (p *Plugin) ConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{Key: "smtp_host", Label: "SMTP Server", Description: "SMTP server hostname (e.g., smtp.gmail.com)", Type: "string", EnvVar: "HBOX_SMTP_HOST", Required: true},
		{Key: "smtp_port", Label: "SMTP Port", Description: "SMTP server port (587 for TLS, 465 for SSL)", Type: "string", Default: "587", EnvVar: "HBOX_SMTP_PORT", Required: true},
		{Key: "username", Label: "Username", Description: "SMTP authentication username (usually your email)", Type: "string", EnvVar: "HBOX_SMTP_USERNAME", Required: true},
		{Key: "password", Label: "Password", Description: "SMTP authentication password or app password", Type: "secret", EnvVar: "HBOX_SMTP_PASSWORD", Required: true},
		{Key: "from_address", Label: "From Address", Description: "Sender email address", Type: "string", Required: true},
		{Key: "to_addresses", Label: "Recipients", Description: "Comma-separated recipient email addresses", Type: "string", Required: true},
	}
}

func (p *Plugin) Configure(values map[string]string) error {
	if v, ok := values["smtp_host"]; ok {
		p.smtpHost = v
	}
	if v, ok := values["smtp_port"]; ok && v != "" {
		p.smtpPort = v
	}
	if v, ok := values["username"]; ok {
		p.username = v
	}
	if v, ok := values["password"]; ok {
		p.password = v
	}
	if v, ok := values["from_address"]; ok {
		p.fromAddr = v
	}
	if v, ok := values["to_addresses"]; ok && v != "" {
		addrs := strings.Split(v, ",")
		p.toAddrs = make([]string, 0, len(addrs))
		for _, addr := range addrs {
			trimmed := strings.TrimSpace(addr)
			if trimmed != "" {
				p.toAddrs = append(p.toAddrs, trimmed)
			}
		}
	}
	return nil
}

func (p *Plugin) RequestedPermissions() []plugins.PermissionRequest {
	return []plugins.PermissionRequest{
		{Permission: plugins.PermNotifiers, Reason: "Deliver email notifications", Required: true},
		{Permission: plugins.PermNetwork, Reason: "Connect to SMTP server to send emails", Required: true},
		{Permission: plugins.PermConfig, Reason: "Store SMTP server settings", Required: true},
	}
}

// Compile-time interface checks.
var (
	_ plugins.Plugin                = (*Plugin)(nil)
	_ plugins.NotificationPlugin    = (*Plugin)(nil)
	_ plugins.ConfigPlugin          = (*Plugin)(nil)
	_ plugins.PluginWithPermissions = (*Plugin)(nil)
)

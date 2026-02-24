// Package example provides a sample plugin demonstrating the HomeBoxNG plugin architecture.
// Use this as a template for building your own plugins.
package example

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hay-kot/httpkit/server"
	"github.com/rs/zerolog"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/reporting/eventbus"
)

// ExamplePlugin demonstrates all plugin capabilities.
// It implements Plugin, RoutePlugin, EventPlugin, ScheduledPlugin, and ConfigPlugin.
type ExamplePlugin struct {
	logger  zerolog.Logger
	pctx    plugins.PluginContext
	message string
}

// New creates a new ExamplePlugin instance.
func New() *ExamplePlugin {
	return &ExamplePlugin{
		message: "Hello from ExamplePlugin!",
	}
}

// Info returns plugin metadata.
func (p *ExamplePlugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "example",
		Version:     "1.0.0",
		Description: "Example plugin demonstrating the HomeBoxNG plugin architecture",
		Author:      "HomeBoxNG Team",
		BuiltIn:     true,
	}
}

// Init initializes the plugin with HomeBoxNG dependencies.
func (p *ExamplePlugin) Init(ctx plugins.PluginContext) error {
	p.pctx = ctx
	p.logger = ctx.Logger
	p.logger.Info().Msg("example plugin initialized")
	return nil
}

// Start begins any background processes.
func (p *ExamplePlugin) Start(_ context.Context) error {
	p.logger.Info().Msg("example plugin started")
	return nil
}

// Stop gracefully shuts down the plugin.
func (p *ExamplePlugin) Stop(_ context.Context) error {
	p.logger.Info().Msg("example plugin stopped")
	return nil
}

// Routes registers HTTP endpoints under /api/plugins/example/.
func (p *ExamplePlugin) Routes(r chi.Router) {
	r.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
		_ = server.JSON(w, http.StatusOK, map[string]string{
			"message": p.message,
			"plugin":  "example",
			"version": "1.0.0",
		})
	})

	r.Get("/items/count", func(w http.ResponseWriter, r *http.Request) {
		// Demonstrates accessing HomeBoxNG services
		_ = server.JSON(w, http.StatusOK, map[string]string{
			"info": "This endpoint could query items via p.pctx.Repos",
		})
	})
}

// SubscribeEvents registers event handlers.
func (p *ExamplePlugin) SubscribeEvents(bus *eventbus.EventBus) {
	bus.Subscribe(eventbus.EventItemMutation, func(data any) {
		p.logger.Debug().Msg("example plugin received item mutation event")
	})
}

// Schedule returns recurring background tasks.
func (p *ExamplePlugin) Schedule() []plugins.ScheduledTask {
	return []plugins.ScheduledTask{
		{
			Name: "heartbeat",
			Fn: func(ctx context.Context) {
				p.logger.Debug().Msg("example plugin heartbeat")
			},
		},
	}
}

// ConfigSchema returns the plugin's configuration options.
func (p *ExamplePlugin) ConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{
			Key:         "message",
			Label:       "Greeting Message",
			Description: "The message returned by the /hello endpoint",
			Type:        "string",
			Default:     "Hello from ExamplePlugin!",
			Required:    false,
		},
	}
}

// Configure applies configuration values.
func (p *ExamplePlugin) Configure(values map[string]string) error {
	if msg, ok := values["message"]; ok && msg != "" {
		p.message = msg
	}
	return nil
}

// RequestedPermissions declares what data/capabilities this plugin needs.
// Users see this list in the Plugin Manager before enabling the plugin.
func (p *ExamplePlugin) RequestedPermissions() []plugins.PermissionRequest {
	return []plugins.PermissionRequest{
		{
			Permission: plugins.PermReadItems,
			Reason:     "Display item count in the hello endpoint",
			Required:   false,
		},
		{
			Permission: plugins.PermEvents,
			Reason:     "Log item mutation events for demonstration",
			Required:   false,
		},
		{
			Permission: plugins.PermAPIRoutes,
			Reason:     "Register /hello and /items/count API endpoints",
			Required:   true,
		},
		{
			Permission: plugins.PermConfig,
			Reason:     "Store the greeting message setting",
			Required:   true,
		},
	}
}

// Verify interface compliance at compile time.
var (
	_ plugins.Plugin                = (*ExamplePlugin)(nil)
	_ plugins.RoutePlugin           = (*ExamplePlugin)(nil)
	_ plugins.EventPlugin           = (*ExamplePlugin)(nil)
	_ plugins.ScheduledPlugin       = (*ExamplePlugin)(nil)
	_ plugins.ConfigPlugin          = (*ExamplePlugin)(nil)
	_ plugins.PluginWithPermissions = (*ExamplePlugin)(nil)
)

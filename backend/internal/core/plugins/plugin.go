// Package plugins provides the plugin architecture for HomeBoxNG.
// Plugins extend HomeBoxNG with custom functionality without modifying core code.
// They can add API endpoints, subscribe to events, run scheduled tasks, and register
// configuration sections.
package plugins

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/reporting/eventbus"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
)

// PluginInfo contains metadata about a plugin.
type PluginInfo struct {
	// Name is the unique identifier for the plugin (e.g., "ai-vision", "discord-bot").
	// Used for API route prefixes: /api/plugins/{name}/
	Name string `json:"name"`

	// Version follows semver (e.g., "1.0.0").
	Version string `json:"version"`

	// Description is a short human-readable description.
	Description string `json:"description"`

	// Author identifies the plugin creator.
	Author string `json:"author"`

	// BuiltIn marks plugins that ship with HomeBoxNG.
	BuiltIn bool `json:"builtIn"`
}

// PluginContext provides plugins with access to HomeBoxNG's core dependencies.
type PluginContext struct {
	Config   *config.Config
	DB       *ent.Client
	Repos    *repo.AllRepos
	Services *services.AllServices
	Bus      *eventbus.EventBus
	Logger   zerolog.Logger
}

// Plugin is the base interface that all plugins must implement.
type Plugin interface {
	// Info returns the plugin's metadata.
	Info() PluginInfo

	// Init initializes the plugin with HomeBoxNG's core dependencies.
	// Called once during application startup after all core services are ready.
	Init(ctx PluginContext) error

	// Start begins any background processes the plugin needs.
	// Called after Init. The context is cancelled on shutdown.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the plugin.
	// Called during application shutdown before database connections close.
	Stop(ctx context.Context) error
}

// RoutePlugin is implemented by plugins that register HTTP API endpoints.
// Routes are mounted under /api/plugins/{name}/.
type RoutePlugin interface {
	Plugin

	// Routes registers the plugin's HTTP routes on the provided router.
	// The router is already scoped to /api/plugins/{name}/.
	Routes(r chi.Router)
}

// EventPlugin is implemented by plugins that subscribe to HomeBoxNG events.
type EventPlugin interface {
	Plugin

	// SubscribeEvents registers event handlers on the event bus.
	// Called after Init but before Start.
	SubscribeEvents(bus *eventbus.EventBus)
}

// ScheduledTask defines a recurring background task.
type ScheduledTask struct {
	// Name identifies the task (for logging and management).
	Name string

	// Fn is the function to execute on each tick.
	Fn func(ctx context.Context)
}

// ScheduledPlugin is implemented by plugins that need recurring background tasks.
type ScheduledPlugin interface {
	Plugin

	// Schedule returns the list of recurring tasks the plugin needs.
	Schedule() []ScheduledTask
}

// ConfigPlugin is implemented by plugins that expose configurable settings.
type ConfigPlugin interface {
	Plugin

	// ConfigSchema returns the plugin's configuration keys with descriptions and defaults.
	ConfigSchema() []ConfigField

	// Configure applies the provided configuration values.
	// Called after Init with values loaded from the database or environment.
	Configure(values map[string]string) error
}

// ConfigField describes a single configuration option for a plugin.
type ConfigField struct {
	// Key is the configuration key (e.g., "api_url", "poll_interval").
	Key string `json:"key"`

	// Label is the human-readable label for UI display.
	Label string `json:"label"`

	// Description explains what this setting does.
	Description string `json:"description"`

	// Type is the field type: "string", "number", "boolean", "secret", "select".
	Type string `json:"type"`

	// Default is the default value if not configured.
	Default string `json:"default"`

	// EnvVar is an optional environment variable name (e.g., "HBOX_PAPERLESS_TOKEN").
	// When set, the plugin will fall back to this env var if no user-configured value exists.
	// Priority order: user-configured value > environment variable > Default.
	EnvVar string `json:"envVar,omitempty"`

	// Options lists valid values for "select" type fields.
	Options []string `json:"options,omitempty"`

	// Required marks whether this field must be set for the plugin to function.
	Required bool `json:"required"`
}

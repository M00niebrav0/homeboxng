package plugins

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/reporting/eventbus"
)

// State represents the lifecycle state of a plugin.
type State string

const (
	StateRegistered State = "registered"
	StateConfigured State = "configured"
	StateStarted    State = "started"
	StateStopped    State = "stopped"
	StateError      State = "error"
)

// PluginEntry tracks a plugin and its lifecycle state.
type PluginEntry struct {
	Plugin Plugin
	State  State
	Error  string
}

// Registry manages the lifecycle of all plugins.
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]*PluginEntry
	pctx    PluginContext
	logger  zerolog.Logger
	order   []string // tracks registration order for ordered startup/shutdown
}

// NewRegistry creates a plugin registry with the given context.
func NewRegistry(pctx PluginContext) *Registry {
	return &Registry{
		plugins: make(map[string]*PluginEntry),
		pctx:    pctx,
		logger:  pctx.Logger.With().Str("component", "plugin-registry").Logger(),
	}
}

// Register adds a plugin to the registry and initializes it.
func (r *Registry) Register(p Plugin) error {
	info := p.Info()
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[info.Name]; exists {
		return fmt.Errorf("plugin %q is already registered", info.Name)
	}

	entry := &PluginEntry{
		Plugin: p,
		State:  StateRegistered,
	}

	// Initialize the plugin
	pluginLogger := r.logger.With().Str("plugin", info.Name).Logger()
	pctx := r.pctx
	pctx.Logger = pluginLogger

	if err := p.Init(pctx); err != nil {
		entry.State = StateError
		entry.Error = err.Error()
		r.plugins[info.Name] = entry
		r.order = append(r.order, info.Name)
		return fmt.Errorf("failed to initialize plugin %q: %w", info.Name, err)
	}

	entry.State = StateConfigured
	r.plugins[info.Name] = entry
	r.order = append(r.order, info.Name)

	pluginLogger.Info().
		Str("version", info.Version).
		Msg("plugin registered")

	return nil
}

// SubscribeAll subscribes all EventPlugin implementations to the event bus.
func (r *Registry) SubscribeAll(bus *eventbus.EventBus) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, name := range r.order {
		entry := r.plugins[name]
		if entry.State == StateError {
			continue
		}
		if ep, ok := entry.Plugin.(EventPlugin); ok {
			ep.SubscribeEvents(bus)
			r.logger.Debug().Str("plugin", name).Msg("subscribed to events")
		}
	}
}

// MountRoutes mounts all RoutePlugin routes under /api/plugins/{name}/.
func (r *Registry) MountRoutes(router chi.Router) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, name := range r.order {
		entry := r.plugins[name]
		if entry.State == StateError {
			continue
		}
		if rp, ok := entry.Plugin.(RoutePlugin); ok {
			prefix := fmt.Sprintf("/plugins/%s", name)
			router.Route(prefix, func(sub chi.Router) {
				rp.Routes(sub)
			})
			r.logger.Debug().Str("plugin", name).Str("prefix", prefix).Msg("mounted routes")
		}
	}
}

// StartAll starts all registered plugins.
// Before starting each plugin, it resolves environment variable fallbacks
// for any ConfigPlugin fields that have EnvVar set.
func (r *Registry) StartAll(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, name := range r.order {
		entry := r.plugins[name]
		if entry.State == StateError {
			r.logger.Warn().Str("plugin", name).Str("error", entry.Error).Msg("skipping errored plugin")
			continue
		}

		// Resolve environment variable fallbacks for ConfigPlugin implementations.
		// Priority: user-configured value > environment variable > default.
		if cp, ok := entry.Plugin.(ConfigPlugin); ok {
			r.applyEnvVarDefaults(name, cp)
		}

		if err := entry.Plugin.Start(ctx); err != nil {
			entry.State = StateError
			entry.Error = err.Error()
			r.logger.Error().Err(err).Str("plugin", name).Msg("failed to start plugin")
			continue
		}

		entry.State = StateStarted
		r.logger.Info().Str("plugin", name).Msg("plugin started")
	}
	return nil
}

// applyEnvVarDefaults checks each ConfigField for an EnvVar declaration.
// If the env var is set, it builds a config map and calls Configure() so the
// plugin picks up the value. The plugin's Configure method already handles
// the "skip if empty" logic, so only non-empty env var values are applied.
func (r *Registry) applyEnvVarDefaults(name string, cp ConfigPlugin) {
	schema := cp.ConfigSchema()
	envValues := make(map[string]string)
	applied := 0

	for _, field := range schema {
		if field.EnvVar == "" {
			continue
		}

		val := os.Getenv(field.EnvVar)
		if val != "" {
			envValues[field.Key] = val
			applied++
			r.logger.Debug().
				Str("plugin", name).
				Str("field", field.Key).
				Str("envVar", field.EnvVar).
				Msg("applying env var config")
		}
	}

	if applied > 0 {
		if err := cp.Configure(envValues); err != nil {
			r.logger.Warn().Err(err).
				Str("plugin", name).
				Int("fields", applied).
				Msg("failed to apply env var configuration")
		} else {
			r.logger.Info().
				Str("plugin", name).
				Int("fields", applied).
				Msg("applied env var configuration")
		}
	}
}

// StopAll stops all started plugins in reverse registration order.
func (r *Registry) StopAll(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Stop in reverse order
	for i := len(r.order) - 1; i >= 0; i-- {
		name := r.order[i]
		entry := r.plugins[name]
		if entry.State != StateStarted {
			continue
		}

		if err := entry.Plugin.Stop(ctx); err != nil {
			r.logger.Error().Err(err).Str("plugin", name).Msg("error stopping plugin")
		} else {
			entry.State = StateStopped
			r.logger.Info().Str("plugin", name).Msg("plugin stopped")
		}
	}
}

// GetScheduledTasks collects all scheduled tasks from ScheduledPlugin implementations.
func (r *Registry) GetScheduledTasks() []ScheduledTask {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tasks []ScheduledTask
	for _, name := range r.order {
		entry := r.plugins[name]
		if entry.State == StateError {
			continue
		}
		if sp, ok := entry.Plugin.(ScheduledPlugin); ok {
			for _, task := range sp.Schedule() {
				// Prefix task name with plugin name for uniqueness
				task.Name = fmt.Sprintf("%s/%s", name, task.Name)
				tasks = append(tasks, task)
			}
		}
	}
	return tasks
}

// List returns info and state for all registered plugins.
func (r *Registry) List() []PluginStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]PluginStatus, 0, len(r.plugins))
	for _, name := range r.order {
		entry := r.plugins[name]
		status := PluginStatus{
			Info:  entry.Plugin.Info(),
			State: entry.State,
			Error: entry.Error,
		}

		// Include permission declarations if the plugin supports them
		if pp, ok := entry.Plugin.(PluginWithPermissions); ok {
			status.Permissions = pp.RequestedPermissions()
		}

		result = append(result, status)
	}
	return result
}

// Get returns a specific plugin by name.
func (r *Registry) Get(name string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.plugins[name]
	if !ok {
		return nil, false
	}
	return entry.Plugin, true
}

// PluginStatus is the API response type for plugin listing.
type PluginStatus struct {
	Info        PluginInfo          `json:"info"`
	State       State               `json:"state"`
	Error       string              `json:"error,omitempty"`
	Permissions []PermissionRequest `json:"permissions,omitempty"`
}

// ConfigurePlugin applies configuration values to a ConfigPlugin.
func (r *Registry) ConfigurePlugin(name string, values map[string]string) error {
	r.mu.RLock()
	entry, ok := r.plugins[name]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("plugin %q not found", name)
	}

	cp, ok := entry.Plugin.(ConfigPlugin)
	if !ok {
		return fmt.Errorf("plugin %q does not support configuration", name)
	}

	if err := cp.Configure(values); err != nil {
		return fmt.Errorf("failed to configure plugin %q: %w", name, err)
	}

	log.Info().Str("plugin", name).Msg("plugin configured")
	return nil
}

// GetConfigSchema returns the configuration schema for a ConfigPlugin.
func (r *Registry) GetConfigSchema(name string) ([]ConfigField, error) {
	r.mu.RLock()
	entry, ok := r.plugins[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("plugin %q not found", name)
	}

	cp, ok := entry.Plugin.(ConfigPlugin)
	if !ok {
		return nil, fmt.Errorf("plugin %q does not support configuration", name)
	}

	return cp.ConfigSchema(), nil
}

// Unregister removes a plugin from the registry.
// Built-in plugins cannot be unregistered.
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.plugins[name]
	if !ok {
		return fmt.Errorf("plugin %q not found", name)
	}

	if entry.Plugin.Info().BuiltIn {
		return fmt.Errorf("cannot unregister built-in plugin %q", name)
	}

	delete(r.plugins, name)

	// Remove from order slice
	for i, n := range r.order {
		if n == name {
			r.order = append(r.order[:i], r.order[i+1:]...)
			break
		}
	}

	r.logger.Info().Str("plugin", name).Msg("plugin unregistered")
	return nil
}

// RegisterExternal creates and registers an external plugin from a registration request.
func (r *Registry) RegisterExternal(reg ExternalPluginRegistration) *ExternalPlugin {
	ep := &ExternalPlugin{
		reg:    reg,
		client: &http.Client{Timeout: 10 * time.Second},
		logger: r.logger.With().Str("plugin", reg.Name).Logger(),
	}

	if err := r.Register(ep); err != nil {
		r.logger.Error().Err(err).Str("plugin", reg.Name).Msg("failed to register external plugin")
		return nil
	}

	return ep
}

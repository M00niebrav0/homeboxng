package plugins

import (
	"context"
	"fmt"
	"sync"
)

// HookPoint identifies a specific point in the application lifecycle where
// plugins can intercept operations.
type HookPoint string

const (
	// Item lifecycle hooks.
	BeforeItemCreate HookPoint = "before_item_create"
	AfterItemCreate  HookPoint = "after_item_create"
	BeforeItemUpdate HookPoint = "before_item_update"
	AfterItemUpdate  HookPoint = "after_item_update"
	BeforeItemDelete HookPoint = "before_item_delete"
	AfterItemDelete  HookPoint = "after_item_delete"

	// Location lifecycle hooks.
	BeforeLocationCreate HookPoint = "before_location_create"
	AfterLocationCreate  HookPoint = "after_location_create"

	// Attachment lifecycle hooks.
	BeforeAttachmentUpload HookPoint = "before_attachment_upload"
	AfterAttachmentUpload  HookPoint = "after_attachment_upload"

	// Import/Export lifecycle hooks.
	BeforeExport HookPoint = "before_export"
	AfterImport  HookPoint = "after_import"
)

// AllHookPoints returns a list of all defined hook points.
func AllHookPoints() []HookPoint {
	return []HookPoint{
		BeforeItemCreate, AfterItemCreate,
		BeforeItemUpdate, AfterItemUpdate,
		BeforeItemDelete, AfterItemDelete,
		BeforeLocationCreate, AfterLocationCreate,
		BeforeAttachmentUpload, AfterAttachmentUpload,
		BeforeExport, AfterImport,
	}
}

// HookContext carries contextual data through a hook execution chain.
// Before-hooks can use Cancel() to veto the operation.
type HookContext struct {
	// HookPoint identifies which hook point is being executed.
	HookPoint HookPoint

	// EntityID is the identifier of the entity being operated on.
	EntityID string

	// EntityType describes the kind of entity (e.g., "item", "location", "attachment").
	EntityType string

	// Data carries arbitrary key-value data associated with the operation.
	Data map[string]interface{}

	// UserID identifies the user performing the operation.
	UserID string

	// GroupID identifies the group context for the operation.
	GroupID string

	// Cancel is a function that, when called, vetoes the operation.
	// Only meaningful for "Before" hook points.
	Cancel func()

	// Cancelled indicates whether a handler has vetoed the operation.
	Cancelled bool
}

// NewHookContext creates a new HookContext for the given hook point.
func NewHookContext(point HookPoint, entityID, entityType, userID, groupID string) *HookContext {
	hc := &HookContext{
		HookPoint:  point,
		EntityID:   entityID,
		EntityType: entityType,
		Data:       make(map[string]interface{}),
		UserID:     userID,
		GroupID:    groupID,
	}
	hc.Cancel = func() {
		hc.Cancelled = true
	}
	return hc
}

// HookHandler is a function that handles a hook invocation.
type HookHandler func(ctx context.Context, hc *HookContext) error

// HookPlugin is implemented by plugins that want to intercept core operations.
type HookPlugin interface {
	// RegisterHooks returns a map of hook points to the handlers the plugin
	// wants to register for each point.
	RegisterHooks() map[HookPoint][]HookHandler
}

// hookEntry stores a handler along with the name of the plugin that registered it.
type hookEntry struct {
	pluginName string
	handler    HookHandler
}

// HookRegistry manages the registration and execution of hook handlers.
// It is safe for concurrent use.
type HookRegistry struct {
	mu    sync.RWMutex
	hooks map[HookPoint][]hookEntry
}

// NewHookRegistry creates an empty HookRegistry.
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{
		hooks: make(map[HookPoint][]hookEntry),
	}
}

// Register adds a handler for a given hook point, attributed to the named plugin.
// Handlers for the same hook point execute in registration order.
func (hr *HookRegistry) Register(pluginName string, point HookPoint, handler HookHandler) {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	hr.hooks[point] = append(hr.hooks[point], hookEntry{
		pluginName: pluginName,
		handler:    handler,
	})
}

// RegisterAll registers all hooks declared by a HookPlugin under the given plugin name.
func (hr *HookRegistry) RegisterAll(pluginName string, hp HookPlugin) {
	hookMap := hp.RegisterHooks()
	for point, handlers := range hookMap {
		for _, handler := range handlers {
			hr.Register(pluginName, point, handler)
		}
	}
}

// Execute runs all registered handlers for a given hook point in registration order.
// Execution stops early if:
//   - A handler returns an error.
//   - A handler calls hc.Cancel() (the HookContext's Cancelled field becomes true).
//   - The provided context is cancelled.
//
// Returns nil if all handlers succeed without cancellation, or the first error encountered.
// For cancelled operations (via hc.Cancel()), Execute returns a HookCancelledError.
func (hr *HookRegistry) Execute(ctx context.Context, point HookPoint, hc *HookContext) error {
	hr.mu.RLock()
	entries := make([]hookEntry, len(hr.hooks[point]))
	copy(entries, hr.hooks[point])
	hr.mu.RUnlock()

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := entry.handler(ctx, hc); err != nil {
			return fmt.Errorf("hook handler from plugin %q at %s failed: %w", entry.pluginName, point, err)
		}

		if hc.Cancelled {
			return &HookCancelledError{
				PluginName: entry.pluginName,
				HookPoint:  point,
			}
		}
	}

	return nil
}

// GetRegistered returns a mapping of hook points to the names of plugins that
// have registered handlers for each point.
func (hr *HookRegistry) GetRegistered() map[HookPoint][]string {
	hr.mu.RLock()
	defer hr.mu.RUnlock()

	result := make(map[HookPoint][]string, len(hr.hooks))
	for point, entries := range hr.hooks {
		seen := make(map[string]bool)
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			if !seen[entry.pluginName] {
				seen[entry.pluginName] = true
				names = append(names, entry.pluginName)
			}
		}
		result[point] = names
	}
	return result
}

// HandlerCount returns the total number of handlers registered for a hook point.
func (hr *HookRegistry) HandlerCount(point HookPoint) int {
	hr.mu.RLock()
	defer hr.mu.RUnlock()

	return len(hr.hooks[point])
}

// Clear removes all registered handlers.
func (hr *HookRegistry) Clear() {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	hr.hooks = make(map[HookPoint][]hookEntry)
}

// ClearPlugin removes all handlers registered by a specific plugin.
func (hr *HookRegistry) ClearPlugin(pluginName string) {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	for point, entries := range hr.hooks {
		filtered := make([]hookEntry, 0, len(entries))
		for _, entry := range entries {
			if entry.pluginName != pluginName {
				filtered = append(filtered, entry)
			}
		}
		hr.hooks[point] = filtered
	}
}

// HookCancelledError is returned when a before-hook handler cancels an operation.
type HookCancelledError struct {
	PluginName string
	HookPoint  HookPoint
}

// Error implements the error interface.
func (e *HookCancelledError) Error() string {
	return fmt.Sprintf("operation cancelled by plugin %q at hook point %s", e.PluginName, e.HookPoint)
}

// IsHookCancelled returns true if the error is a HookCancelledError.
func IsHookCancelled(err error) bool {
	_, ok := err.(*HookCancelledError)
	return ok
}

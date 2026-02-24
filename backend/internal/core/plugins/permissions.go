package plugins

import (
	"fmt"
	"sync"
)

// Permission represents a specific capability a plugin can request.
type Permission string

const (
	// Data Access Permissions
	PermReadItems       Permission = "items:read"        // Read item data (names, descriptions, values)
	PermWriteItems      Permission = "items:write"       // Create, update, delete items
	PermReadLocations   Permission = "locations:read"    // Read location hierarchy
	PermWriteLocations  Permission = "locations:write"   // Create, update, delete locations
	PermReadTags        Permission = "tags:read"         // Read tags
	PermWriteTags       Permission = "tags:write"        // Create, update, delete tags
	PermReadAttachments Permission = "attachments:read"  // Read/download attachments and photos
	PermWriteAttachments Permission = "attachments:write" // Upload, delete attachments
	PermReadUsers       Permission = "users:read"        // Read user profiles
	PermReadGroups      Permission = "groups:read"       // Read group membership

	// Feature Permissions
	PermMaintenance    Permission = "maintenance:access"  // Read/write maintenance entries
	PermTemplates      Permission = "templates:access"    // Read/write item templates
	PermNotifiers      Permission = "notifiers:access"    // Access notification system
	PermLabels         Permission = "labels:access"       // Generate and print labels
	PermImportExport   Permission = "import_export:access" // CSV/Excel import and export

	// System Permissions
	PermEvents         Permission = "events:subscribe"    // Subscribe to real-time events (item/location/tag mutations)
	PermScheduledTasks Permission = "scheduled:run"       // Run background scheduled tasks
	PermAPIRoutes      Permission = "api:routes"          // Register custom API endpoints
	PermWebUI          Permission = "webui:pages"         // Add pages to the web interface
	PermConfig         Permission = "config:manage"       // Store and manage plugin configuration

	// External Access Permissions
	PermNetwork        Permission = "network:outbound"    // Make outbound HTTP/network requests
	PermWebhooks       Permission = "webhooks:receive"    // Receive incoming webhooks
	PermStorage        Permission = "storage:files"       // Access the file storage backend
	PermDatabase       Permission = "database:direct"     // Direct database access (advanced)
)

// PermissionInfo provides human-readable details about a permission.
type PermissionInfo struct {
	Permission  Permission `json:"permission"`
	Label       string     `json:"label"`
	Description string     `json:"description"`
	Category    string     `json:"category"`
	Risk        string     `json:"risk"` // "low", "medium", "high"
}

// AllPermissions returns info about every available permission.
func AllPermissions() []PermissionInfo {
	return []PermissionInfo{
		{PermReadItems, "Read Items", "View item names, descriptions, quantities, and values", "Data", "low"},
		{PermWriteItems, "Modify Items", "Create, edit, and delete inventory items", "Data", "medium"},
		{PermReadLocations, "Read Locations", "View your location hierarchy and structure", "Data", "low"},
		{PermWriteLocations, "Modify Locations", "Create, edit, and delete storage locations", "Data", "medium"},
		{PermReadTags, "Read Tags", "View tag names and assignments", "Data", "low"},
		{PermWriteTags, "Modify Tags", "Create, edit, and delete tags", "Data", "medium"},
		{PermReadAttachments, "Read Attachments", "View and download photos and documents", "Data", "low"},
		{PermWriteAttachments, "Modify Attachments", "Upload and delete photos and documents", "Data", "medium"},
		{PermReadUsers, "Read User Profiles", "View user names and email addresses", "Data", "medium"},
		{PermReadGroups, "Read Groups", "View group membership and organization structure", "Data", "low"},

		{PermMaintenance, "Maintenance Access", "Read and write maintenance log entries", "Features", "low"},
		{PermTemplates, "Template Access", "Use and manage item templates", "Features", "low"},
		{PermNotifiers, "Notification Access", "Send notifications through configured channels", "Features", "medium"},
		{PermLabels, "Label Printing", "Generate QR codes and print labels", "Features", "low"},
		{PermImportExport, "Import/Export", "Import and export inventory data (CSV, Excel)", "Features", "medium"},

		{PermEvents, "Real-time Events", "Receive notifications when items, locations, or tags change", "System", "low"},
		{PermScheduledTasks, "Background Tasks", "Run periodic background tasks on a schedule", "System", "medium"},
		{PermAPIRoutes, "Custom API Endpoints", "Add new API endpoints to HomeBoxNG", "System", "medium"},
		{PermWebUI, "Web UI Pages", "Add custom pages to the HomeBoxNG web interface", "System", "medium"},
		{PermConfig, "Plugin Settings", "Store and manage its own configuration", "System", "low"},

		{PermNetwork, "Outbound Network", "Connect to external services and APIs over the internet", "External", "high"},
		{PermWebhooks, "Incoming Webhooks", "Receive data from external services", "External", "medium"},
		{PermStorage, "File Storage", "Read and write files in the storage backend", "External", "high"},
		{PermDatabase, "Direct Database", "Directly query the database (advanced, full access)", "External", "high"},
	}
}

// PermissionGrant tracks whether a user has granted a permission to a plugin.
type PermissionGrant struct {
	PluginName string     `json:"pluginName"`
	Permission Permission `json:"permission"`
	Granted    bool       `json:"granted"`
	GrantedBy  string     `json:"grantedBy,omitempty"` // user who granted it
	GrantedAt  string     `json:"grantedAt,omitempty"` // ISO timestamp
}

// PermissionRequest is what a plugin declares it needs.
type PermissionRequest struct {
	Permission Permission `json:"permission"`
	Reason     string     `json:"reason"` // Why the plugin needs this permission
	Required   bool       `json:"required"` // If false, plugin works without it (degraded)
}

// PluginWithPermissions extends the Plugin interface with permission declarations.
type PluginWithPermissions interface {
	Plugin

	// RequestedPermissions returns the permissions this plugin needs and why.
	RequestedPermissions() []PermissionRequest
}

// PermissionManager tracks granted permissions per plugin.
type PermissionManager struct {
	mu     sync.RWMutex
	grants map[string]map[Permission]bool // pluginName -> permission -> granted
}

// NewPermissionManager creates a new permission manager.
func NewPermissionManager() *PermissionManager {
	return &PermissionManager{
		grants: make(map[string]map[Permission]bool),
	}
}

// Grant gives a plugin a specific permission.
func (pm *PermissionManager) Grant(pluginName string, perm Permission) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.grants[pluginName] == nil {
		pm.grants[pluginName] = make(map[Permission]bool)
	}
	pm.grants[pluginName][perm] = true
}

// Revoke removes a permission from a plugin.
func (pm *PermissionManager) Revoke(pluginName string, perm Permission) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.grants[pluginName] != nil {
		delete(pm.grants[pluginName], perm)
	}
}

// GrantAll grants all declared permissions for a plugin.
func (pm *PermissionManager) GrantAll(pluginName string, perms []PermissionRequest) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.grants[pluginName] == nil {
		pm.grants[pluginName] = make(map[Permission]bool)
	}
	for _, p := range perms {
		pm.grants[pluginName][p.Permission] = true
	}
}

// RevokeAll removes all permissions from a plugin.
func (pm *PermissionManager) RevokeAll(pluginName string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.grants, pluginName)
}

// HasPermission checks if a plugin has a specific permission.
func (pm *PermissionManager) HasPermission(pluginName string, perm Permission) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.grants[pluginName] == nil {
		return false
	}
	return pm.grants[pluginName][perm]
}

// Check verifies a plugin has a permission and returns an error if not.
func (pm *PermissionManager) Check(pluginName string, perm Permission) error {
	if !pm.HasPermission(pluginName, perm) {
		return fmt.Errorf("plugin %q does not have permission %q", pluginName, perm)
	}
	return nil
}

// GetGrants returns all granted permissions for a plugin.
func (pm *PermissionManager) GetGrants(pluginName string) []Permission {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var result []Permission
	for perm, granted := range pm.grants[pluginName] {
		if granted {
			result = append(result, perm)
		}
	}
	return result
}

// GetAllGrants returns grants for all plugins.
func (pm *PermissionManager) GetAllGrants() map[string][]Permission {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[string][]Permission)
	for name, perms := range pm.grants {
		for perm, granted := range perms {
			if granted {
				result[name] = append(result[name], perm)
			}
		}
	}
	return result
}

// PermissionSummary provides a full view of a plugin's permission state.
type PermissionSummary struct {
	PluginName string              `json:"pluginName"`
	Requested  []PermissionRequest `json:"requested"`
	Granted    []Permission        `json:"granted"`
	Denied     []Permission        `json:"denied"`
}

// GetSummary builds a complete permission summary for a plugin.
func (pm *PermissionManager) GetSummary(pluginName string, requested []PermissionRequest) PermissionSummary {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	granted := make([]Permission, 0)
	denied := make([]Permission, 0)

	for _, req := range requested {
		if pm.grants[pluginName] != nil && pm.grants[pluginName][req.Permission] {
			granted = append(granted, req.Permission)
		} else {
			denied = append(denied, req.Permission)
		}
	}

	return PermissionSummary{
		PluginName: pluginName,
		Requested:  requested,
		Granted:    granted,
		Denied:     denied,
	}
}

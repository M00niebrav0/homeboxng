package v1

import (
	"net/http"
	"time"

	"github.com/hay-kot/httpkit/errchain"
	"github.com/hay-kot/httpkit/server"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins"
	"github.com/sysadminsmedia/homebox/backend/internal/web/adapters"
)

// HandlePluginsList godoc
//
//	@Summary	List All Plugins
//	@Tags		Plugins
//	@Produce	json
//	@Success	200	{array}	plugins.PluginStatus
//	@Router		/v1/plugins [GET]
//	@Security	Bearer
func (ctrl *V1Controller) HandlePluginsList() errchain.HandlerFunc {
	return adapters.Command(func(r *http.Request) ([]plugins.PluginStatus, error) {
		if ctrl.pluginRegistry == nil {
			return []plugins.PluginStatus{}, nil
		}
		return ctrl.pluginRegistry.List(), nil
	}, http.StatusOK)
}

// HandlePluginsCatalog godoc
//
//	@Summary	Browse Available Plugins
//	@Tags		Plugins
//	@Produce	json
//	@Param		q	query	string	false	"Search query"
//	@Success	200	{array}	plugins.ExternalPluginManifest
//	@Router		/v1/plugins/catalog [GET]
//	@Security	Bearer
func (ctrl *V1Controller) HandlePluginsCatalog() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if ctrl.pluginCatalog == nil {
			return server.JSON(w, http.StatusOK, []plugins.ExternalPluginManifest{})
		}

		query := r.URL.Query().Get("q")
		results := ctrl.pluginCatalog.Search(query)
		return server.JSON(w, http.StatusOK, results)
	}
}

// HandlePluginsCatalogRefresh godoc
//
//	@Summary	Refresh Plugin Catalog
//	@Tags		Plugins
//	@Produce	json
//	@Success	200	{object}	map[string]string
//	@Router		/v1/plugins/catalog/refresh [POST]
//	@Security	Bearer
func (ctrl *V1Controller) HandlePluginsCatalogRefresh() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if ctrl.pluginCatalog == nil {
			return server.JSON(w, http.StatusOK, map[string]string{"status": "no catalog configured"})
		}

		if err := ctrl.pluginCatalog.Refresh(r.Context()); err != nil {
			return server.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return server.JSON(w, http.StatusOK, map[string]string{"status": "refreshed"})
	}
}

// HandlePluginEnable godoc
//
//	@Summary	Enable a Plugin
//	@Tags		Plugins
//	@Produce	json
//	@Param		name	path	string	true	"Plugin name"
//	@Success	200		{object}	map[string]string
//	@Router		/v1/plugins/{name}/enable [POST]
//	@Security	Bearer
func (ctrl *V1Controller) HandlePluginEnable() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		name := r.PathValue("name")
		if ctrl.pluginRegistry == nil {
			return server.JSON(w, http.StatusNotFound, map[string]string{"error": "plugin system not initialized"})
		}

		p, ok := ctrl.pluginRegistry.Get(name)
		if !ok {
			return server.JSON(w, http.StatusNotFound, map[string]string{"error": "plugin not found"})
		}

		if err := p.Start(r.Context()); err != nil {
			return server.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return server.JSON(w, http.StatusOK, map[string]string{"status": "enabled"})
	}
}

// HandlePluginDisable godoc
//
//	@Summary	Disable a Plugin
//	@Tags		Plugins
//	@Produce	json
//	@Param		name	path	string	true	"Plugin name"
//	@Success	200		{object}	map[string]string
//	@Router		/v1/plugins/{name}/disable [POST]
//	@Security	Bearer
func (ctrl *V1Controller) HandlePluginDisable() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		name := r.PathValue("name")
		if ctrl.pluginRegistry == nil {
			return server.JSON(w, http.StatusNotFound, map[string]string{"error": "plugin system not initialized"})
		}

		p, ok := ctrl.pluginRegistry.Get(name)
		if !ok {
			return server.JSON(w, http.StatusNotFound, map[string]string{"error": "plugin not found"})
		}

		if err := p.Stop(r.Context()); err != nil {
			return server.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return server.JSON(w, http.StatusOK, map[string]string{"status": "disabled"})
	}
}

// HandlePluginDelete godoc
//
//	@Summary	Delete/Uninstall a Plugin
//	@Tags		Plugins
//	@Produce	json
//	@Param		name	path	string	true	"Plugin name"
//	@Success	200		{object}	map[string]string
//	@Router		/v1/plugins/{name} [DELETE]
//	@Security	Bearer
func (ctrl *V1Controller) HandlePluginDelete() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		name := r.PathValue("name")
		if ctrl.pluginRegistry == nil {
			return server.JSON(w, http.StatusNotFound, map[string]string{"error": "plugin system not initialized"})
		}

		p, ok := ctrl.pluginRegistry.Get(name)
		if !ok {
			return server.JSON(w, http.StatusNotFound, map[string]string{"error": "plugin not found"})
		}

		// Don't allow deleting built-in plugins
		info := p.Info()
		if info.BuiltIn {
			return server.JSON(w, http.StatusForbidden, map[string]string{"error": "cannot delete built-in plugin"})
		}

		// Stop the plugin first
		_ = p.Stop(r.Context())

		// Remove from registry
		if err := ctrl.pluginRegistry.Unregister(name); err != nil {
			return server.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return server.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

// HandlePluginReset godoc
//
//	@Summary	Reset a Plugin to Default Configuration
//	@Tags		Plugins
//	@Produce	json
//	@Param		name	path	string	true	"Plugin name"
//	@Success	200		{object}	map[string]string
//	@Router		/v1/plugins/{name}/reset [POST]
//	@Security	Bearer
func (ctrl *V1Controller) HandlePluginReset() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		name := r.PathValue("name")
		if ctrl.pluginRegistry == nil {
			return server.JSON(w, http.StatusNotFound, map[string]string{"error": "plugin system not initialized"})
		}

		// Reset config to defaults
		schema, err := ctrl.pluginRegistry.GetConfigSchema(name)
		if err != nil {
			return server.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		defaults := make(map[string]string)
		for _, field := range schema {
			defaults[field.Key] = field.Default
		}

		if err := ctrl.pluginRegistry.ConfigurePlugin(name, defaults); err != nil {
			return server.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return server.JSON(w, http.StatusOK, map[string]string{"status": "reset to defaults"})
	}
}

// HandlePluginConfig godoc
//
//	@Summary	Get Plugin Configuration
//	@Tags		Plugins
//	@Produce	json
//	@Param		name	path	string	true	"Plugin name"
//	@Success	200		{array}	plugins.ConfigField
//	@Router		/v1/plugins/{name}/config [GET]
//	@Security	Bearer
func (ctrl *V1Controller) HandlePluginConfig() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		name := r.PathValue("name")
		if ctrl.pluginRegistry == nil {
			return server.JSON(w, http.StatusNotFound, map[string]string{"error": "plugin system not initialized"})
		}

		schema, err := ctrl.pluginRegistry.GetConfigSchema(name)
		if err != nil {
			return server.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		return server.JSON(w, http.StatusOK, schema)
	}
}

// HandlePluginConfigUpdate godoc
//
//	@Summary	Update Plugin Configuration
//	@Tags		Plugins
//	@Accept		json
//	@Produce	json
//	@Param		name	path	string				true	"Plugin name"
//	@Param		body	body	map[string]string	true	"Configuration values"
//	@Success	200		{object}	map[string]string
//	@Router		/v1/plugins/{name}/config [PUT]
//	@Security	Bearer
func (ctrl *V1Controller) HandlePluginConfigUpdate() errchain.HandlerFunc {
	return adapters.Action(func(r *http.Request, body map[string]string) (map[string]string, error) {
		name := r.PathValue("name")
		if ctrl.pluginRegistry == nil {
			return map[string]string{"error": "plugin system not initialized"}, nil
		}

		if err := ctrl.pluginRegistry.ConfigurePlugin(name, body); err != nil {
			return map[string]string{"error": err.Error()}, nil
		}

		return map[string]string{"status": "configured"}, nil
	}, http.StatusOK)
}

// HandlePluginRegisterExternal godoc
//
//	@Summary	Register an External Plugin
//	@Tags		Plugins
//	@Accept		json
//	@Produce	json
//	@Param		body	body	plugins.ExternalPluginRegistration	true	"Plugin registration"
//	@Success	201		{object}	map[string]string
//	@Router		/v1/plugins/register [POST]
//	@Security	Bearer
func (ctrl *V1Controller) HandlePluginRegisterExternal() errchain.HandlerFunc {
	return adapters.Action(func(r *http.Request, reg plugins.ExternalPluginRegistration) (map[string]string, error) {
		if ctrl.pluginRegistry == nil {
			return map[string]string{"error": "plugin system not initialized"}, nil
		}

		ep := ctrl.pluginRegistry.RegisterExternal(reg)
		if ep == nil {
			return map[string]string{"error": "failed to register external plugin"}, nil
		}

		return map[string]string{
			"status": "registered",
			"name":   reg.Name,
		}, nil
	}, http.StatusCreated)
}

// HandlePluginPermissions godoc
//
//	@Summary	Get Plugin Permissions
//	@Tags		Plugins
//	@Produce	json
//	@Param		name	path	string	true	"Plugin name"
//	@Success	200		{object}	plugins.PermissionSummary
//	@Router		/v1/plugins/{name}/permissions [GET]
//	@Security	Bearer
func (ctrl *V1Controller) HandlePluginPermissions() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		name := r.PathValue("name")
		if ctrl.pluginRegistry == nil {
			return server.JSON(w, http.StatusNotFound, map[string]string{"error": "plugin system not initialized"})
		}

		p, ok := ctrl.pluginRegistry.Get(name)
		if !ok {
			return server.JSON(w, http.StatusNotFound, map[string]string{"error": "plugin not found"})
		}

		pp, ok := p.(plugins.PluginWithPermissions)
		if !ok {
			return server.JSON(w, http.StatusOK, plugins.PermissionSummary{
				PluginName: name,
				Requested:  []plugins.PermissionRequest{},
				Granted:    []plugins.Permission{},
				Denied:     []plugins.Permission{},
			})
		}

		if ctrl.permissionManager == nil {
			return server.JSON(w, http.StatusOK, plugins.PermissionSummary{
				PluginName: name,
				Requested:  pp.RequestedPermissions(),
			})
		}

		summary := ctrl.permissionManager.GetSummary(name, pp.RequestedPermissions())
		return server.JSON(w, http.StatusOK, summary)
	}
}

// HandlePluginGrantPermission godoc
//
//	@Summary	Grant a Permission to a Plugin
//	@Tags		Plugins
//	@Accept		json
//	@Produce	json
//	@Param		name	path	string	true	"Plugin name"
//	@Success	200		{object}	map[string]string
//	@Router		/v1/plugins/{name}/permissions/grant [POST]
//	@Security	Bearer
func (ctrl *V1Controller) HandlePluginGrantPermission() errchain.HandlerFunc {
	type grantRequest struct {
		Permission string `json:"permission"`
	}

	return adapters.Action(func(r *http.Request, body grantRequest) (map[string]string, error) {
		name := r.PathValue("name")
		if ctrl.permissionManager == nil {
			return map[string]string{"error": "permission system not initialized"}, nil
		}

		ctrl.permissionManager.Grant(name, plugins.Permission(body.Permission))
		return map[string]string{"status": "granted", "permission": body.Permission}, nil
	}, http.StatusOK)
}

// HandlePluginRevokePermission godoc
//
//	@Summary	Revoke a Permission from a Plugin
//	@Tags		Plugins
//	@Accept		json
//	@Produce	json
//	@Param		name	path	string	true	"Plugin name"
//	@Success	200		{object}	map[string]string
//	@Router		/v1/plugins/{name}/permissions/revoke [POST]
//	@Security	Bearer
func (ctrl *V1Controller) HandlePluginRevokePermission() errchain.HandlerFunc {
	type revokeRequest struct {
		Permission string `json:"permission"`
	}

	return adapters.Action(func(r *http.Request, body revokeRequest) (map[string]string, error) {
		name := r.PathValue("name")
		if ctrl.permissionManager == nil {
			return map[string]string{"error": "permission system not initialized"}, nil
		}

		ctrl.permissionManager.Revoke(name, plugins.Permission(body.Permission))
		return map[string]string{"status": "revoked", "permission": body.Permission}, nil
	}, http.StatusOK)
}

// HandlePluginGrantAll godoc
//
//	@Summary	Grant All Requested Permissions to a Plugin
//	@Tags		Plugins
//	@Produce	json
//	@Param		name	path	string	true	"Plugin name"
//	@Success	200		{object}	map[string]string
//	@Router		/v1/plugins/{name}/permissions/grant-all [POST]
//	@Security	Bearer
func (ctrl *V1Controller) HandlePluginGrantAll() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		name := r.PathValue("name")
		if ctrl.permissionManager == nil {
			return server.JSON(w, http.StatusBadRequest, map[string]string{"error": "permission system not initialized"})
		}

		p, ok := ctrl.pluginRegistry.Get(name)
		if !ok {
			return server.JSON(w, http.StatusNotFound, map[string]string{"error": "plugin not found"})
		}

		pp, ok := p.(plugins.PluginWithPermissions)
		if !ok {
			return server.JSON(w, http.StatusOK, map[string]string{"status": "no permissions to grant"})
		}

		ctrl.permissionManager.GrantAll(name, pp.RequestedPermissions())
		return server.JSON(w, http.StatusOK, map[string]string{"status": "all permissions granted"})
	}
}

// HandleAllPermissions godoc
//
//	@Summary	List All Available Permissions
//	@Tags		Plugins
//	@Produce	json
//	@Success	200	{array}	plugins.PermissionInfo
//	@Router		/v1/plugins/permissions [GET]
//	@Security	Bearer
func (ctrl *V1Controller) HandleAllPermissions() errchain.HandlerFunc {
	return adapters.Command(func(r *http.Request) ([]plugins.PermissionInfo, error) {
		return plugins.AllPermissions(), nil
	}, http.StatusOK)
}

// HandlePluginSources godoc
//
//	@Summary	List Plugin Sources (HACS-like repos)
//	@Tags		Plugins
//	@Produce	json
//	@Success	200	{array}	plugins.CatalogSource
//	@Router		/v1/plugins/sources [GET]
//	@Security	Bearer
func (ctrl *V1Controller) HandlePluginSources() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if ctrl.multiCatalog == nil {
			return server.JSON(w, http.StatusOK, []plugins.CatalogSource{})
		}
		return server.JSON(w, http.StatusOK, ctrl.multiCatalog.Sources())
	}
}

// HandlePluginSourceAdd godoc
//
//	@Summary	Add a Plugin Source (GitHub repo)
//	@Tags		Plugins
//	@Accept		json
//	@Produce	json
//	@Success	201	{object}	map[string]string
//	@Router		/v1/plugins/sources [POST]
//	@Security	Bearer
func (ctrl *V1Controller) HandlePluginSourceAdd() errchain.HandlerFunc {
	type addSourceRequest struct {
		// Repository is the GitHub repo path (e.g., "user/homeboxng-plugin-foo")
		// or a full URL to a registry.json file.
		Repository string `json:"repository"`
		Name       string `json:"name,omitempty"`
	}

	return adapters.Action(func(r *http.Request, body addSourceRequest) (map[string]string, error) {
		if ctrl.multiCatalog == nil {
			return map[string]string{"error": "catalog not initialized"}, nil
		}

		ctrl.multiCatalog.AddSourceFromGitHub(body.Repository)
		return map[string]string{
			"status":     "added",
			"repository": body.Repository,
		}, nil
	}, http.StatusCreated)
}

// HandlePluginSourceRemove godoc
//
//	@Summary	Remove a Plugin Source
//	@Tags		Plugins
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	map[string]string
//	@Router		/v1/plugins/sources [DELETE]
//	@Security	Bearer
func (ctrl *V1Controller) HandlePluginSourceRemove() errchain.HandlerFunc {
	type removeSourceRequest struct {
		URL string `json:"url"`
	}

	return adapters.Action(func(r *http.Request, body removeSourceRequest) (map[string]string, error) {
		if ctrl.multiCatalog == nil {
			return map[string]string{"error": "catalog not initialized"}, nil
		}

		if err := ctrl.multiCatalog.RemoveSource(body.URL); err != nil {
			return map[string]string{"error": err.Error()}, nil
		}

		return map[string]string{"status": "removed"}, nil
	}, http.StatusOK)
}

// HandlePluginLogs godoc
//
//	@Summary	Get Plugin Logs
//	@Tags		Plugins
//	@Produce	json
//	@Param		name	path	string	true	"Plugin name"
//	@Success	200		{array}	plugins.LogEntry
//	@Router		/v1/plugins/{name}/logs [GET]
//	@Security	Bearer
func (ctrl *V1Controller) HandlePluginLogs() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		name := r.PathValue("name")
		if ctrl.pluginLogs == nil {
			return server.JSON(w, http.StatusOK, []plugins.LogEntry{})
		}

		logs := ctrl.pluginLogs.GetLogs(name)
		if logs == nil {
			logs = []plugins.LogEntry{}
		}
		return server.JSON(w, http.StatusOK, logs)
	}
}

// ============================================================================
// Notification Handlers
// ============================================================================

// HandleNotificationPlatforms godoc
//
//	@Summary	List Notification Platforms
//	@Tags		Notifications
//	@Produce	json
//	@Success	200	{array}	plugins.NotificationPlatformStatus
//	@Router		/v1/notifications/platforms [GET]
//	@Security	Bearer
func (ctrl *V1Controller) HandleNotificationPlatforms() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if ctrl.notificationDispatcher == nil {
			return server.JSON(w, http.StatusOK, []plugins.NotificationPlatformStatus{})
		}
		return server.JSON(w, http.StatusOK, ctrl.notificationDispatcher.ListPlatforms())
	}
}

// HandleNotificationSend godoc
//
//	@Summary	Send a Notification
//	@Tags		Notifications
//	@Accept		json
//	@Produce	json
//	@Success	200	{array}	plugins.NotificationResult
//	@Router		/v1/notifications/send [POST]
//	@Security	Bearer
func (ctrl *V1Controller) HandleNotificationSend() errchain.HandlerFunc {
	type sendRequest struct {
		Title    string            `json:"title"`
		Body     string            `json:"body"`
		Level    string            `json:"level"`
		Category string            `json:"category"`
		URL      string            `json:"url,omitempty"`
		ImageURL string            `json:"imageUrl,omitempty"`
		Metadata map[string]string `json:"metadata,omitempty"`
	}

	return adapters.Action(func(r *http.Request, body sendRequest) ([]plugins.NotificationResult, error) {
		if ctrl.notificationDispatcher == nil {
			return []plugins.NotificationResult{}, nil
		}

		n := plugins.Notification{
			Title:     body.Title,
			Body:      body.Body,
			Level:     body.Level,
			Category:  body.Category,
			URL:       body.URL,
			ImageURL:  body.ImageURL,
			Metadata:  body.Metadata,
			Timestamp: time.Now(),
		}

		results := ctrl.notificationDispatcher.Dispatch(r.Context(), n)
		return results, nil
	}, http.StatusOK)
}

// HandleNotificationTest godoc
//
//	@Summary	Test a Notification Platform
//	@Tags		Notifications
//	@Produce	json
//	@Param		platform	path	string	true	"Platform ID"
//	@Success	200			{object}	map[string]string
//	@Router		/v1/notifications/test/{platform} [POST]
//	@Security	Bearer
func (ctrl *V1Controller) HandleNotificationTest() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		platform := r.PathValue("platform")
		if ctrl.notificationDispatcher == nil {
			return server.JSON(w, http.StatusBadRequest, map[string]string{"error": "notifications not initialized"})
		}

		if err := ctrl.notificationDispatcher.TestPlatform(r.Context(), platform); err != nil {
			return server.JSON(w, http.StatusInternalServerError, map[string]string{
				"error":    err.Error(),
				"platform": platform,
			})
		}

		return server.JSON(w, http.StatusOK, map[string]string{
			"status":   "sent",
			"platform": platform,
		})
	}
}

// HandleNotificationCategories godoc
//
//	@Summary	List Notification Categories
//	@Tags		Notifications
//	@Produce	json
//	@Success	200	{array}	plugins.NotificationCategory
//	@Router		/v1/notifications/categories [GET]
//	@Security	Bearer
func (ctrl *V1Controller) HandleNotificationCategories() errchain.HandlerFunc {
	return adapters.Command(func(r *http.Request) ([]plugins.NotificationCategory, error) {
		return plugins.NotificationCategories(), nil
	}, http.StatusOK)
}

// HandleNotificationPreferences godoc
//
//	@Summary	Get User Notification Preferences
//	@Tags		Notifications
//	@Produce	json
//	@Success	200	{array}	plugins.NotificationPreference
//	@Router		/v1/notifications/preferences [GET]
//	@Security	Bearer
func (ctrl *V1Controller) HandleNotificationPreferences() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if ctrl.notificationDispatcher == nil {
			return server.JSON(w, http.StatusOK, []plugins.NotificationPreference{})
		}

		// Use "default" as user ID for now; will integrate with auth context later
		prefs := ctrl.notificationDispatcher.GetPreferences("default")
		if prefs == nil {
			prefs = []plugins.NotificationPreference{}
		}
		return server.JSON(w, http.StatusOK, prefs)
	}
}

// HandleNotificationPreferenceUpdate godoc
//
//	@Summary	Update Notification Preference
//	@Tags		Notifications
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	map[string]string
//	@Router		/v1/notifications/preferences [PUT]
//	@Security	Bearer
func (ctrl *V1Controller) HandleNotificationPreferenceUpdate() errchain.HandlerFunc {
	return adapters.Action(func(r *http.Request, body plugins.NotificationPreference) (map[string]string, error) {
		if ctrl.notificationDispatcher == nil {
			return map[string]string{"error": "notifications not initialized"}, nil
		}

		if body.UserID == "" {
			body.UserID = "default"
		}

		ctrl.notificationDispatcher.SetPreference(body)
		return map[string]string{"status": "updated"}, nil
	}, http.StatusOK)
}

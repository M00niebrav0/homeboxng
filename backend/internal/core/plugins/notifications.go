package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// NotificationPlugin is implemented by plugins that can deliver notifications.
// Each platform (email, Discord, Windows, iOS, Android, Chrome, Firefox, Linux)
// is a separate NotificationPlugin that users enable/disable independently.
type NotificationPlugin interface {
	Plugin

	// Platform returns the notification platform identifier (e.g., "email", "discord", "windows").
	Platform() string

	// PlatformLabel returns a human-readable platform name (e.g., "Email", "Discord", "Windows Desktop").
	PlatformLabel() string

	// PlatformIcon returns an icon identifier for the UI (e.g., "email", "discord", "windows").
	PlatformIcon() string

	// Send delivers a notification to the platform.
	Send(ctx context.Context, notification Notification) error

	// TestNotification sends a test message to verify the platform is configured correctly.
	TestNotification(ctx context.Context) error
}

// Notification represents a message to deliver to one or more platforms.
type Notification struct {
	// Title is the notification headline.
	Title string `json:"title"`

	// Body is the notification message content.
	Body string `json:"body"`

	// Level indicates the severity: "info", "warning", "error", "success".
	Level string `json:"level"`

	// Category groups notifications (e.g., "item.added", "maintenance.due", "backup.complete").
	Category string `json:"category"`

	// URL is an optional link to open when the notification is clicked.
	URL string `json:"url,omitempty"`

	// ImageURL is an optional image to include in the notification.
	ImageURL string `json:"imageUrl,omitempty"`

	// Metadata holds additional key-value pairs for platform-specific rendering.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Timestamp is when the notification was created.
	Timestamp time.Time `json:"timestamp"`
}

// NotificationPreference controls per-user, per-platform, per-category notification settings.
type NotificationPreference struct {
	UserID   string `json:"userId"`
	Platform string `json:"platform"` // "email", "discord", "windows", etc.
	Category string `json:"category"` // "item.added", "maintenance.due", "*" for all
	Enabled  bool   `json:"enabled"`
}

// NotificationDispatcher manages all notification plugins and routes notifications
// to the correct platforms based on user preferences.
type NotificationDispatcher struct {
	mu        sync.RWMutex
	platforms map[string]NotificationPlugin // platform ID -> plugin
	prefs     map[string][]NotificationPreference // userID -> preferences
	logger    func(level, msg string)
}

// NewNotificationDispatcher creates a dispatcher for routing notifications.
func NewNotificationDispatcher() *NotificationDispatcher {
	return &NotificationDispatcher{
		platforms: make(map[string]NotificationPlugin),
		prefs:    make(map[string][]NotificationPreference),
	}
}

// RegisterPlatform adds a notification platform to the dispatcher.
func (d *NotificationDispatcher) RegisterPlatform(np NotificationPlugin) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.platforms[np.Platform()] = np
}

// UnregisterPlatform removes a notification platform.
func (d *NotificationDispatcher) UnregisterPlatform(platform string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.platforms, platform)
}

// ListPlatforms returns all registered notification platforms with their status.
func (d *NotificationDispatcher) ListPlatforms() []NotificationPlatformStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var result []NotificationPlatformStatus
	for _, np := range d.platforms {
		result = append(result, NotificationPlatformStatus{
			Platform: np.Platform(),
			Label:    np.PlatformLabel(),
			Icon:     np.PlatformIcon(),
			Info:     np.Info(),
		})
	}
	return result
}

// NotificationPlatformStatus describes a registered notification platform.
type NotificationPlatformStatus struct {
	Platform string     `json:"platform"`
	Label    string     `json:"label"`
	Icon     string     `json:"icon"`
	Info     PluginInfo `json:"info"`
}

// Dispatch sends a notification to all enabled platforms for the specified users.
// If userIDs is empty, sends to all users with matching preferences.
func (d *NotificationDispatcher) Dispatch(ctx context.Context, notification Notification, userIDs ...string) []NotificationResult {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if notification.Timestamp.IsZero() {
		notification.Timestamp = time.Now()
	}

	var results []NotificationResult

	for platformID, np := range d.platforms {
		// Check if any target user has this platform enabled for this category
		shouldSend := len(userIDs) == 0 // broadcast if no specific users
		if !shouldSend {
			for _, uid := range userIDs {
				if d.isEnabledForUser(uid, platformID, notification.Category) {
					shouldSend = true
					break
				}
			}
		}

		if !shouldSend {
			continue
		}

		err := np.Send(ctx, notification)
		results = append(results, NotificationResult{
			Platform: platformID,
			Success:  err == nil,
			Error:    errString(err),
		})
	}

	return results
}

// NotificationResult reports the outcome of sending to a single platform.
type NotificationResult struct {
	Platform string `json:"platform"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}

// SetPreference updates a user's notification preference for a platform/category.
func (d *NotificationDispatcher) SetPreference(pref NotificationPreference) {
	d.mu.Lock()
	defer d.mu.Unlock()

	userPrefs := d.prefs[pref.UserID]

	// Update existing preference if found
	for i, existing := range userPrefs {
		if existing.Platform == pref.Platform && existing.Category == pref.Category {
			userPrefs[i] = pref
			d.prefs[pref.UserID] = userPrefs
			return
		}
	}

	// Add new preference
	d.prefs[pref.UserID] = append(userPrefs, pref)
}

// GetPreferences returns all notification preferences for a user.
func (d *NotificationDispatcher) GetPreferences(userID string) []NotificationPreference {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.prefs[userID]
}

// isEnabledForUser checks if a user has enabled notifications for a platform/category.
// Defaults to true (opt-out model) if no explicit preference is set.
func (d *NotificationDispatcher) isEnabledForUser(userID, platform, category string) bool {
	prefs, ok := d.prefs[userID]
	if !ok {
		return true // No preferences set = everything enabled (opt-out model)
	}

	for _, p := range prefs {
		if p.Platform == platform {
			// Check specific category match
			if p.Category == category {
				return p.Enabled
			}
			// Check wildcard match
			if p.Category == "*" {
				return p.Enabled
			}
		}
	}

	return true // No matching preference = enabled by default
}

// TestPlatform sends a test notification to verify platform configuration.
func (d *NotificationDispatcher) TestPlatform(ctx context.Context, platform string) error {
	d.mu.RLock()
	np, ok := d.platforms[platform]
	d.mu.RUnlock()

	if !ok {
		return fmt.Errorf("notification platform %q not registered", platform)
	}

	return np.TestNotification(ctx)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// NotificationCategories returns all available notification categories.
func NotificationCategories() []NotificationCategory {
	return []NotificationCategory{
		{"item.added", "Item Added", "A new item was added to inventory"},
		{"item.updated", "Item Updated", "An existing item was modified"},
		{"item.deleted", "Item Deleted", "An item was removed from inventory"},
		{"item.moved", "Item Moved", "An item was moved to a different location"},
		{"location.added", "Location Added", "A new storage location was created"},
		{"maintenance.due", "Maintenance Due", "An item is due for scheduled maintenance"},
		{"maintenance.overdue", "Maintenance Overdue", "An item is past its maintenance date"},
		{"warranty.expiring", "Warranty Expiring", "An item's warranty is about to expire"},
		{"lending.due", "Lending Due", "A lent item is due for return"},
		{"lending.overdue", "Lending Overdue", "A lent item is past its return date"},
		{"stock.low", "Low Stock", "An item quantity is below the minimum threshold"},
		{"backup.complete", "Backup Complete", "A system backup completed successfully"},
		{"backup.failed", "Backup Failed", "A system backup failed"},
		{"plugin.error", "Plugin Error", "A plugin encountered an error"},
		{"import.complete", "Import Complete", "A bulk import or scan job finished"},
		{"vision.complete", "Vision Complete", "AI photo identification completed"},
		{"label.printed", "Label Printed", "A label was sent to the printer"},
		{"security.login", "Login Alert", "A new device or location logged in"},
	}
}

// NotificationCategory describes a type of event that can trigger notifications.
type NotificationCategory struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

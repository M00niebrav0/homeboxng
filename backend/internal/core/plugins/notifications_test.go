package plugins

import (
	"context"
	"fmt"
	"testing"
)

// mockNotificationPlugin simulates a notification platform.
type mockNotificationPlugin struct {
	testPlugin
	platform      string
	label         string
	icon          string
	sendErr       error
	sentMessages  []Notification
	testCalled    bool
}

func newMockNotifier(platform, label, icon string) *mockNotificationPlugin {
	return &mockNotificationPlugin{
		testPlugin: *newTestPlugin("notify-" + platform),
		platform:   platform,
		label:      label,
		icon:       icon,
	}
}

func (m *mockNotificationPlugin) Platform() string      { return m.platform }
func (m *mockNotificationPlugin) PlatformLabel() string  { return m.label }
func (m *mockNotificationPlugin) PlatformIcon() string   { return m.icon }

func (m *mockNotificationPlugin) Send(_ context.Context, n Notification) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sentMessages = append(m.sentMessages, n)
	return nil
}

func (m *mockNotificationPlugin) TestNotification(_ context.Context) error {
	m.testCalled = true
	return nil
}

func TestNotificationDispatcher_RegisterPlatform(t *testing.T) {
	d := NewNotificationDispatcher()
	email := newMockNotifier("email", "Email", "mdi-email")
	discord := newMockNotifier("discord", "Discord", "mdi-discord")

	d.RegisterPlatform(email)
	d.RegisterPlatform(discord)

	platforms := d.ListPlatforms()
	if len(platforms) != 2 {
		t.Fatalf("expected 2 platforms, got %d", len(platforms))
	}
}

func TestNotificationDispatcher_UnregisterPlatform(t *testing.T) {
	d := NewNotificationDispatcher()
	d.RegisterPlatform(newMockNotifier("email", "Email", "mdi-email"))
	d.UnregisterPlatform("email")

	platforms := d.ListPlatforms()
	if len(platforms) != 0 {
		t.Fatalf("expected 0 platforms after unregister, got %d", len(platforms))
	}
}

func TestNotificationDispatcher_Dispatch_Broadcast(t *testing.T) {
	d := NewNotificationDispatcher()
	email := newMockNotifier("email", "Email", "mdi-email")
	discord := newMockNotifier("discord", "Discord", "mdi-discord")
	d.RegisterPlatform(email)
	d.RegisterPlatform(discord)

	notification := Notification{
		Title:    "Test",
		Body:     "Test body",
		Level:    "info",
		Category: "item.added",
	}

	// Broadcast (no user IDs = send to all platforms)
	results := d.Dispatch(context.Background(), notification)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if !r.Success {
			t.Errorf("platform %q failed: %s", r.Platform, r.Error)
		}
	}

	if len(email.sentMessages) != 1 {
		t.Error("expected email to receive 1 message")
	}
	if len(discord.sentMessages) != 1 {
		t.Error("expected discord to receive 1 message")
	}
}

func TestNotificationDispatcher_Dispatch_WithSendError(t *testing.T) {
	d := NewNotificationDispatcher()
	failing := newMockNotifier("email", "Email", "mdi-email")
	failing.sendErr = fmt.Errorf("SMTP connection refused")
	d.RegisterPlatform(failing)

	results := d.Dispatch(context.Background(), Notification{
		Title: "Test", Category: "item.added",
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Success {
		t.Error("expected failure result")
	}
	if results[0].Error != "SMTP connection refused" {
		t.Errorf("unexpected error: %s", results[0].Error)
	}
}

func TestNotificationDispatcher_Dispatch_UserPreferences(t *testing.T) {
	d := NewNotificationDispatcher()
	email := newMockNotifier("email", "Email", "mdi-email")
	discord := newMockNotifier("discord", "Discord", "mdi-discord")
	d.RegisterPlatform(email)
	d.RegisterPlatform(discord)

	// User disables email for item.added
	d.SetPreference(NotificationPreference{
		UserID:   "user1",
		Platform: "email",
		Category: "item.added",
		Enabled:  false,
	})

	results := d.Dispatch(context.Background(), Notification{
		Title: "New Item", Category: "item.added",
	}, "user1")

	// Only discord should send (email disabled for this category)
	sentPlatforms := map[string]bool{}
	for _, r := range results {
		if r.Success {
			sentPlatforms[r.Platform] = true
		}
	}

	if sentPlatforms["email"] {
		t.Error("email should NOT have been sent (disabled)")
	}
	if !sentPlatforms["discord"] {
		t.Error("discord should have been sent")
	}
}

func TestNotificationDispatcher_Dispatch_WildcardPreference(t *testing.T) {
	d := NewNotificationDispatcher()
	email := newMockNotifier("email", "Email", "mdi-email")
	d.RegisterPlatform(email)

	// User disables all email notifications
	d.SetPreference(NotificationPreference{
		UserID:   "user1",
		Platform: "email",
		Category: "*",
		Enabled:  false,
	})

	results := d.Dispatch(context.Background(), Notification{
		Title: "Anything", Category: "backup.complete",
	}, "user1")

	if len(results) != 0 {
		for _, r := range results {
			if r.Platform == "email" && r.Success {
				t.Error("email should be suppressed by wildcard disable")
			}
		}
	}
}

func TestNotificationDispatcher_OptOutModel(t *testing.T) {
	d := NewNotificationDispatcher()
	email := newMockNotifier("email", "Email", "mdi-email")
	d.RegisterPlatform(email)

	// No preferences set = everything enabled (opt-out model)
	results := d.Dispatch(context.Background(), Notification{
		Title: "Test", Category: "item.added",
	}, "user-with-no-prefs")

	sentEmail := false
	for _, r := range results {
		if r.Platform == "email" && r.Success {
			sentEmail = true
		}
	}
	if !sentEmail {
		t.Error("expected email to be sent (opt-out model, no prefs = enabled)")
	}
}

func TestNotificationDispatcher_SetPreference_Update(t *testing.T) {
	d := NewNotificationDispatcher()

	// Set initial preference
	d.SetPreference(NotificationPreference{
		UserID: "user1", Platform: "email", Category: "item.added", Enabled: false,
	})

	// Update same preference
	d.SetPreference(NotificationPreference{
		UserID: "user1", Platform: "email", Category: "item.added", Enabled: true,
	})

	prefs := d.GetPreferences("user1")
	if len(prefs) != 1 {
		t.Fatalf("expected 1 preference (updated, not duplicate), got %d", len(prefs))
	}
	if !prefs[0].Enabled {
		t.Error("expected preference to be updated to enabled")
	}
}

func TestNotificationDispatcher_GetPreferences_Empty(t *testing.T) {
	d := NewNotificationDispatcher()
	prefs := d.GetPreferences("nonexistent")
	if prefs != nil {
		t.Error("expected nil prefs for unknown user")
	}
}

func TestNotificationDispatcher_TestPlatform(t *testing.T) {
	d := NewNotificationDispatcher()
	email := newMockNotifier("email", "Email", "mdi-email")
	d.RegisterPlatform(email)

	err := d.TestPlatform(context.Background(), "email")
	if err != nil {
		t.Fatalf("TestPlatform() error = %v", err)
	}
	if !email.testCalled {
		t.Error("expected TestNotification to be called")
	}
}

func TestNotificationDispatcher_TestPlatform_NotFound(t *testing.T) {
	d := NewNotificationDispatcher()
	err := d.TestPlatform(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for unregistered platform")
	}
}

func TestNotificationCategories(t *testing.T) {
	cats := NotificationCategories()
	if len(cats) != 18 {
		t.Errorf("expected 18 notification categories, got %d", len(cats))
	}

	// Verify each has required fields
	for _, cat := range cats {
		if cat.ID == "" {
			t.Error("category ID should not be empty")
		}
		if cat.Label == "" {
			t.Errorf("category %q should have a label", cat.ID)
		}
		if cat.Description == "" {
			t.Errorf("category %q should have a description", cat.ID)
		}
	}
}

func TestNotification_TimestampAutoSet(t *testing.T) {
	d := NewNotificationDispatcher()
	email := newMockNotifier("email", "Email", "mdi-email")
	d.RegisterPlatform(email)

	// Send without timestamp
	d.Dispatch(context.Background(), Notification{
		Title: "Test", Category: "item.added",
	})

	if len(email.sentMessages) == 1 {
		if email.sentMessages[0].Timestamp.IsZero() {
			t.Error("expected timestamp to be auto-set")
		}
	}
}

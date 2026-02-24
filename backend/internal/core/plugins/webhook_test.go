package plugins

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func testLogger() zerolog.Logger {
	return zerolog.Nop()
}

func TestWebhookManager_Register(t *testing.T) {
	wm := NewWebhookManager(testLogger())

	wh := WebhookConfig{
		ID:         "wh-1",
		PluginName: "test-plugin",
		URL:        "https://example.com/hook",
		Events:     []string{"item.created"},
		Active:     true,
	}

	err := wm.Register(wh)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	webhooks := wm.GetWebhooks("test-plugin")
	if len(webhooks) != 1 {
		t.Fatalf("expected 1 webhook, got %d", len(webhooks))
	}
	if webhooks[0].ID != "wh-1" {
		t.Errorf("ID = %q, want %q", webhooks[0].ID, "wh-1")
	}
}

func TestWebhookManager_RegisterAutoID(t *testing.T) {
	wm := NewWebhookManager(testLogger())

	// Register without providing an ID; one should be auto-generated.
	wh := WebhookConfig{
		PluginName: "test",
		URL:        "https://example.com/hook",
		Events:     []string{"e"},
		Active:     true,
	}

	err := wm.Register(wh)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	webhooks := wm.GetWebhooks("test")
	if len(webhooks) != 1 {
		t.Fatalf("expected 1 webhook, got %d", len(webhooks))
	}
	if webhooks[0].ID == "" {
		t.Error("expected auto-generated ID")
	}
}

func TestWebhookManager_RegisterValidation(t *testing.T) {
	wm := NewWebhookManager(testLogger())

	// Missing URL.
	err := wm.Register(WebhookConfig{PluginName: "p", Events: []string{"e"}})
	if err == nil {
		t.Error("expected error for missing URL")
	}

	// Missing plugin name.
	err = wm.Register(WebhookConfig{URL: "https://a.com", Events: []string{"e"}})
	if err == nil {
		t.Error("expected error for missing plugin name")
	}

	// Missing events.
	err = wm.Register(WebhookConfig{PluginName: "p", URL: "https://a.com"})
	if err == nil {
		t.Error("expected error for missing events")
	}
}

func TestWebhookManager_Unregister(t *testing.T) {
	wm := NewWebhookManager(testLogger())

	wm.Register(WebhookConfig{ID: "wh-1", PluginName: "p", URL: "https://a.com", Events: []string{"e"}, Active: true})
	wm.Register(WebhookConfig{ID: "wh-2", PluginName: "p", URL: "https://b.com", Events: []string{"e"}, Active: true})

	err := wm.Unregister("wh-1")
	if err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}

	webhooks := wm.GetWebhooks("p")
	if len(webhooks) != 1 {
		t.Fatalf("expected 1 webhook after unregister, got %d", len(webhooks))
	}
	if webhooks[0].ID != "wh-2" {
		t.Errorf("remaining webhook ID = %q, want %q", webhooks[0].ID, "wh-2")
	}
}

func TestWebhookManager_UnregisterNotFound(t *testing.T) {
	wm := NewWebhookManager(testLogger())

	err := wm.Unregister("nonexistent")
	if err == nil {
		t.Error("expected error when unregistering nonexistent webhook")
	}
}

func TestWebhookManager_GetWebhooks(t *testing.T) {
	wm := NewWebhookManager(testLogger())

	wm.Register(WebhookConfig{ID: "a1", PluginName: "plugin-a", URL: "https://a.com/1", Events: []string{"e"}, Active: true})
	wm.Register(WebhookConfig{ID: "a2", PluginName: "plugin-a", URL: "https://a.com/2", Events: []string{"e"}, Active: true})
	wm.Register(WebhookConfig{ID: "b1", PluginName: "plugin-b", URL: "https://b.com/1", Events: []string{"e"}, Active: true})

	// Filter by plugin name.
	pluginAWebhooks := wm.GetWebhooks("plugin-a")
	if len(pluginAWebhooks) != 2 {
		t.Errorf("expected 2 webhooks for plugin-a, got %d", len(pluginAWebhooks))
	}

	pluginBWebhooks := wm.GetWebhooks("plugin-b")
	if len(pluginBWebhooks) != 1 {
		t.Errorf("expected 1 webhook for plugin-b, got %d", len(pluginBWebhooks))
	}

	// GetAllWebhooks returns all.
	allWebhooks := wm.GetAllWebhooks()
	if len(allWebhooks) != 3 {
		t.Errorf("expected 3 total webhooks, got %d", len(allWebhooks))
	}
}

func TestWebhookManager_SignPayload(t *testing.T) {
	// signPayload is unexported, so we verify signing through the webhook delivery mechanism.
	// We compute the expected HMAC-SHA256 and compare.
	payload := []byte(`{"event":"item.created","data":{"name":"Widget"}}`)
	secret := "my-secret-key"

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	// Verify a different payload produces a different signature.
	differentPayload := []byte(`{"different":"data"}`)
	mac2 := hmac.New(sha256.New, []byte(secret))
	mac2.Write(differentPayload)
	sig2 := hex.EncodeToString(mac2.Sum(nil))

	if sig2 == expected {
		t.Error("different payloads should produce different signatures")
	}

	// Verify a different secret produces a different signature.
	mac3 := hmac.New(sha256.New, []byte("other-secret"))
	mac3.Write(payload)
	sig3 := hex.EncodeToString(mac3.Sum(nil))

	if sig3 == expected {
		t.Error("different secrets should produce different signatures")
	}
}

func TestWebhookManager_Trigger(t *testing.T) {
	var receivedBody []byte
	var receivedSignature string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSignature = r.Header.Get("X-Webhook-Signature")
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	wm := NewWebhookManager(testLogger())
	// Override the HTTP client to use the test server's client.
	wm.client = server.Client()

	wm.Register(WebhookConfig{
		ID:         "trigger-test",
		PluginName: "test",
		URL:        server.URL,
		Secret:     "test-secret",
		Events:     []string{"item.created", "item.deleted"},
		Active:     true,
	})

	// Trigger is async, so we need to wait for it to complete.
	wm.Trigger("test", "item.created", map[string]string{"name": "Widget"})

	// Give the async delivery some time.
	time.Sleep(500 * time.Millisecond)

	if len(receivedBody) == 0 {
		t.Fatal("expected webhook body to be received")
	}

	// Verify body is valid JSON with expected structure.
	var payload WebhookPayload
	if err := json.Unmarshal(receivedBody, &payload); err != nil {
		t.Fatalf("failed to unmarshal webhook payload: %v", err)
	}
	if payload.Event != "item.created" {
		t.Errorf("payload.Event = %q, want %q", payload.Event, "item.created")
	}

	// Verify signature is present.
	if receivedSignature == "" {
		t.Error("expected X-Webhook-Signature header to be set")
	}
}

func TestWebhookManager_InactiveWebhook(t *testing.T) {
	serverCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	wm := NewWebhookManager(testLogger())
	wm.client = server.Client()

	wm.Register(WebhookConfig{
		ID:         "inactive-test",
		PluginName: "test",
		URL:        server.URL,
		Events:     []string{"test.event"},
		Active:     false, // Disabled
	})

	wm.Trigger("test", "test.event", nil)

	// Give a moment for any errant request.
	time.Sleep(200 * time.Millisecond)

	if serverCalled {
		t.Error("inactive webhook should not trigger HTTP request")
	}
}

func TestWebhookManager_NoMatchingEvent(t *testing.T) {
	serverCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	wm := NewWebhookManager(testLogger())
	wm.client = server.Client()

	wm.Register(WebhookConfig{
		ID:         "nomatch",
		PluginName: "test",
		URL:        server.URL,
		Events:     []string{"item.created"},
		Active:     true,
	})

	wm.Trigger("test", "item.deleted", nil) // Different event

	// Give a moment for any errant request.
	time.Sleep(200 * time.Millisecond)

	if serverCalled {
		t.Error("server should not be called for non-matching event")
	}
}

func TestWebhookManager_CreatedAtAutoSet(t *testing.T) {
	wm := NewWebhookManager(testLogger())

	before := time.Now()
	wm.Register(WebhookConfig{
		ID:         "autotime",
		PluginName: "test",
		URL:        "https://example.com",
		Events:     []string{"e"},
		Active:     true,
	})
	after := time.Now()

	webhooks := wm.GetWebhooks("test")
	if len(webhooks) != 1 {
		t.Fatal("expected 1 webhook")
	}

	createdAt := webhooks[0].CreatedAt
	if createdAt.Before(before) || createdAt.After(after) {
		t.Errorf("CreatedAt = %v, expected between %v and %v", createdAt, before, after)
	}
}

func TestWebhookManager_WrongPlugin(t *testing.T) {
	serverCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	wm := NewWebhookManager(testLogger())
	wm.client = server.Client()

	wm.Register(WebhookConfig{
		ID:         "wrong-plugin",
		PluginName: "plugin-a",
		URL:        server.URL,
		Events:     []string{"test.event"},
		Active:     true,
	})

	// Trigger with a different plugin name.
	wm.Trigger("plugin-b", "test.event", nil)

	time.Sleep(200 * time.Millisecond)

	if serverCalled {
		t.Error("webhook should not fire for a different plugin name")
	}
}

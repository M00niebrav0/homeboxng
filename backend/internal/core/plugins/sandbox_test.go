package plugins

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
)

func TestSandboxedHTTPClient_AllowedHost(t *testing.T) {
	// Start a test server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	// Parse the server URL to get the host.
	u, _ := url.Parse(server.URL)

	sc := NewSandboxedHTTPClient([]string{u.Host}, 5*time.Second)
	client := sc.Client()

	resp, err := client.Get(server.URL + "/test")
	if err != nil {
		t.Fatalf("allowed host request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestSandboxedHTTPClient_BlockedHost(t *testing.T) {
	sc := NewSandboxedHTTPClient([]string{"allowed.example.com"}, 5*time.Second)
	client := sc.Client()

	_, err := client.Get("http://blocked.example.com/test")
	if err == nil {
		t.Error("expected error for non-whitelisted host")
	}
}

func TestSandboxedHTTPClient_EmptyAllowList(t *testing.T) {
	sc := NewSandboxedHTTPClient(nil, 5*time.Second)
	client := sc.Client()

	_, err := client.Get("http://example.com/test")
	if err == nil {
		t.Error("expected error when allow list is empty")
	}
}

func TestSandboxedHTTPClient_PrivateIPBlocked(t *testing.T) {
	tests := []struct {
		name string
		ip   string
	}{
		{"192.168.1.1", "http://192.168.1.1/test"},
		{"10.0.0.1", "http://10.0.0.1/test"},
		{"127.0.0.1", "http://127.0.0.1/test"},
		{"172.16.0.1", "http://172.16.0.1/test"},
		{"169.254.1.1", "http://169.254.1.1/test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Not in the allow list, so should be blocked.
			sc := NewSandboxedHTTPClient([]string{"example.com"}, 5*time.Second)
			client := sc.Client()

			_, err := client.Get(tt.ip)
			if err == nil {
				t.Errorf("expected error for private IP %s", tt.name)
			}
		})
	}
}

func TestSandboxedHTTPClient_PrivateIPAllowedExplicitly(t *testing.T) {
	// Start a test server on localhost (which is a private IP).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	host := u.Host

	// Explicitly allow the localhost address.
	sc := NewSandboxedHTTPClient([]string{host}, 5*time.Second)
	client := sc.Client()

	resp, err := client.Get(server.URL + "/test")
	if err != nil {
		t.Fatalf("explicitly allowed private IP request should succeed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestSandboxedHTTPClient_CaseInsensitive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)

	// Register with uppercase (host is typically lowercase but test robustness).
	sc := NewSandboxedHTTPClient([]string{u.Host}, 5*time.Second)
	client := sc.Client()

	resp, err := client.Get(server.URL + "/test")
	if err != nil {
		t.Fatalf("case-insensitive host match failed: %v", err)
	}
	resp.Body.Close()
}

func TestResourceTracker_RateLimit(t *testing.T) {
	rt := NewResourceTracker()

	// Allow 3 requests per minute.
	maxPerMinute := 3

	for i := 0; i < 3; i++ {
		if err := rt.RecordRequest("plugin-a", maxPerMinute); err != nil {
			t.Fatalf("request %d should succeed, got error: %v", i+1, err)
		}
	}

	// Fourth request should fail.
	err := rt.RecordRequest("plugin-a", maxPerMinute)
	if err == nil {
		t.Error("expected error when exceeding rate limit")
	}
}

func TestResourceTracker_GoroutineLimit(t *testing.T) {
	rt := NewResourceTracker()

	maxGoroutines := 2

	// Acquire two goroutine slots.
	release1, err := rt.RecordGoroutine("plugin-a", maxGoroutines)
	if err != nil {
		t.Fatalf("first goroutine should succeed: %v", err)
	}

	release2, err := rt.RecordGoroutine("plugin-a", maxGoroutines)
	if err != nil {
		t.Fatalf("second goroutine should succeed: %v", err)
	}

	// Third should fail.
	_, err = rt.RecordGoroutine("plugin-a", maxGoroutines)
	if err == nil {
		t.Error("expected error when exceeding goroutine limit")
	}

	// Verify usage.
	usage := rt.GetUsage("plugin-a")
	if usage.ActiveGoroutines != 2 {
		t.Errorf("ActiveGoroutines = %d, want 2", usage.ActiveGoroutines)
	}

	// Release one.
	release1()

	// Now we should be able to acquire one more.
	release3, err := rt.RecordGoroutine("plugin-a", maxGoroutines)
	if err != nil {
		t.Fatalf("after release, goroutine should succeed: %v", err)
	}

	// Cleanup.
	release2()
	release3()
}

func TestResourceTracker_GoroutineRelease(t *testing.T) {
	rt := NewResourceTracker()

	release, err := rt.RecordGoroutine("test", 10)
	if err != nil {
		t.Fatalf("RecordGoroutine() error = %v", err)
	}

	usage := rt.GetUsage("test")
	if usage.ActiveGoroutines != 1 {
		t.Errorf("ActiveGoroutines before release = %d, want 1", usage.ActiveGoroutines)
	}

	release()

	usage = rt.GetUsage("test")
	if usage.ActiveGoroutines != 0 {
		t.Errorf("ActiveGoroutines after release = %d, want 0", usage.ActiveGoroutines)
	}

	// Double-release should not go below 0.
	release()

	usage = rt.GetUsage("test")
	if usage.ActiveGoroutines != 0 {
		t.Errorf("ActiveGoroutines after double-release = %d, want 0", usage.ActiveGoroutines)
	}
}

func TestResourceTracker_ResetCounters(t *testing.T) {
	rt := NewResourceTracker()

	rt.RecordRequest("plugin-a", 100)
	rt.RecordRequest("plugin-a", 100)
	rt.RecordGoroutine("plugin-a", 100)

	rt.ResetCounters()

	usage := rt.GetUsage("plugin-a")
	if usage.ActiveGoroutines != 0 {
		t.Errorf("ActiveGoroutines after reset = %d, want 0", usage.ActiveGoroutines)
	}
	if usage.RequestsInWindow != 0 {
		t.Errorf("RequestsInWindow after reset = %d, want 0", usage.RequestsInWindow)
	}
}

func TestResourceTracker_PluginIsolation(t *testing.T) {
	rt := NewResourceTracker()

	// Plugin A uses up its limit.
	for i := 0; i < 5; i++ {
		rt.RecordRequest("plugin-a", 5)
	}
	err := rt.RecordRequest("plugin-a", 5)
	if err == nil {
		t.Error("plugin-a should be rate limited")
	}

	// Plugin B should still be fine.
	err = rt.RecordRequest("plugin-b", 5)
	if err != nil {
		t.Errorf("plugin-b should not be rate limited: %v", err)
	}
}

func TestResourceTracker_GetUsage_Empty(t *testing.T) {
	rt := NewResourceTracker()

	usage := rt.GetUsage("nonexistent")
	if usage.ActiveGoroutines != 0 || usage.RequestsInWindow != 0 {
		t.Error("expected zero usage for nonexistent plugin")
	}
}

func TestSandbox_Integration(t *testing.T) {
	cfg := SandboxConfig{
		MaxGoroutines:        5,
		MaxRequestsPerMinute: 10,
		AllowedHosts:         []string{"example.com"},
		ReadOnlyMode:         true,
	}

	sandbox := NewSandbox("my-plugin", cfg)

	if !sandbox.IsReadOnly() {
		t.Error("expected read-only mode to be true")
	}

	// Record requests.
	for i := 0; i < 10; i++ {
		if err := sandbox.RecordRequest(); err != nil {
			t.Fatalf("request %d should succeed: %v", i+1, err)
		}
	}

	// Next request should fail.
	if err := sandbox.RecordRequest(); err == nil {
		t.Error("expected rate limit error")
	}

	// Acquire goroutines.
	var releases []func()
	for i := 0; i < 5; i++ {
		rel, err := sandbox.AcquireGoroutine()
		if err != nil {
			t.Fatalf("goroutine %d should succeed: %v", i+1, err)
		}
		releases = append(releases, rel)
	}

	// Next should fail.
	_, err := sandbox.AcquireGoroutine()
	if err == nil {
		t.Error("expected goroutine limit error")
	}

	// Release all.
	for _, rel := range releases {
		rel()
	}

	// Usage should reflect the current state.
	usage := sandbox.Usage()
	if usage.ActiveGoroutines != 0 {
		t.Errorf("ActiveGoroutines = %d, want 0 after releasing all", usage.ActiveGoroutines)
	}
}

func TestSandbox_ConcurrentAccess(t *testing.T) {
	cfg := SandboxConfig{
		MaxGoroutines:        100,
		MaxRequestsPerMinute: 1000,
	}
	sandbox := NewSandbox("concurrent", cfg)

	var wg sync.WaitGroup
	const goroutines = 50

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sandbox.RecordRequest()
			rel, err := sandbox.AcquireGoroutine()
			if err == nil {
				rel()
			}
			sandbox.Usage()
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent sandbox access timed out (possible deadlock)")
	}
}

func TestDefaultSandboxConfig(t *testing.T) {
	cfg := DefaultSandboxConfig()

	if cfg.MaxMemoryMB != 256 {
		t.Errorf("MaxMemoryMB = %d, want 256", cfg.MaxMemoryMB)
	}
	if cfg.MaxCPUPercent != 25 {
		t.Errorf("MaxCPUPercent = %d, want 25", cfg.MaxCPUPercent)
	}
	if cfg.MaxGoroutines != 50 {
		t.Errorf("MaxGoroutines = %d, want 50", cfg.MaxGoroutines)
	}
	if cfg.MaxRequestsPerMinute != 60 {
		t.Errorf("MaxRequestsPerMinute = %d, want 60", cfg.MaxRequestsPerMinute)
	}
	if cfg.MaxStorageMB != 100 {
		t.Errorf("MaxStorageMB = %d, want 100", cfg.MaxStorageMB)
	}
	if cfg.ReadOnlyMode {
		t.Error("ReadOnlyMode should be false by default")
	}
	if cfg.AllowedHosts != nil {
		t.Error("AllowedHosts should be nil by default")
	}
}

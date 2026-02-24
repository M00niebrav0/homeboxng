package plugins

import (
	"context"
	"sync"
	"testing"
	"time"
)

// testHealthChecker implements HealthChecker for testing.
type testHealthChecker struct {
	mu     sync.Mutex
	status HealthStatus
	msg    string
	checks map[string]CheckResult
}

func (c *testHealthChecker) HealthCheck(_ context.Context) HealthReport {
	c.mu.Lock()
	defer c.mu.Unlock()
	return HealthReport{
		Status:    c.status,
		Message:   c.msg,
		LastCheck: time.Now(),
		Checks:    c.checks,
	}
}

func (c *testHealthChecker) setStatus(s HealthStatus, msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status = s
	c.msg = msg
}

func newTestChecker(status HealthStatus, msg string) *testHealthChecker {
	return &testHealthChecker{
		status: status,
		msg:    msg,
		checks: make(map[string]CheckResult),
	}
}

func TestHealthMonitor_RegisterAndGetReport(t *testing.T) {
	hm := NewHealthMonitor(time.Minute, nil)

	checker := newTestChecker(HealthStatusHealthy, "all systems go")
	hm.Register("test-plugin", checker)

	// Before any check cycle, the report should show unknown status.
	report := hm.GetReport("test-plugin")
	if report.PluginName != "test-plugin" {
		t.Errorf("PluginName = %q, want %q", report.PluginName, "test-plugin")
	}
	if report.Status != HealthStatusUnknown {
		t.Errorf("Status = %q, want %q (before check)", report.Status, HealthStatusUnknown)
	}
}

func TestHealthMonitor_GetReportUnregistered(t *testing.T) {
	hm := NewHealthMonitor(time.Minute, nil)

	report := hm.GetReport("nonexistent")
	if report.Status != HealthStatusUnknown {
		t.Errorf("Status = %q, want %q for unregistered plugin", report.Status, HealthStatusUnknown)
	}
	if report.PluginName != "nonexistent" {
		t.Errorf("PluginName = %q, want %q", report.PluginName, "nonexistent")
	}
}

func TestHealthMonitor_StartAndStop(t *testing.T) {
	var transitions []StatusTransition
	var mu sync.Mutex

	callback := func(tr StatusTransition) {
		mu.Lock()
		transitions = append(transitions, tr)
		mu.Unlock()
	}

	hm := NewHealthMonitor(50*time.Millisecond, callback)

	checker := newTestChecker(HealthStatusHealthy, "ok")
	hm.Register("test-plugin", checker)

	ctx := context.Background()
	hm.Start(ctx)

	// Wait for at least one check cycle.
	time.Sleep(200 * time.Millisecond)

	// The initial report should now be populated.
	report := hm.GetReport("test-plugin")
	if report.Status != HealthStatusHealthy {
		t.Errorf("Status = %q, want %q after check cycle", report.Status, HealthStatusHealthy)
	}
	if report.Message != "ok" {
		t.Errorf("Message = %q, want %q", report.Message, "ok")
	}

	hm.Stop()
}

func TestHealthMonitor_StatusTransition(t *testing.T) {
	var transitions []StatusTransition
	var mu sync.Mutex

	callback := func(tr StatusTransition) {
		mu.Lock()
		transitions = append(transitions, tr)
		mu.Unlock()
	}

	hm := NewHealthMonitor(50*time.Millisecond, callback)

	checker := newTestChecker(HealthStatusHealthy, "ok")
	hm.Register("transitioner", checker)

	ctx := context.Background()
	hm.Start(ctx)

	// Wait for the initial check.
	time.Sleep(150 * time.Millisecond)

	// Change status to degraded.
	checker.setStatus(HealthStatusDegraded, "high latency")

	// Wait for the next check cycle.
	time.Sleep(150 * time.Millisecond)

	hm.Stop()

	mu.Lock()
	defer mu.Unlock()

	// We expect at least one transition: Unknown -> Healthy, then Healthy -> Degraded.
	if len(transitions) < 1 {
		t.Fatalf("expected at least 1 transition, got %d", len(transitions))
	}

	// Find the degraded transition.
	foundDegraded := false
	for _, tr := range transitions {
		if tr.To == HealthStatusDegraded {
			foundDegraded = true
			if tr.From != HealthStatusHealthy {
				t.Errorf("expected transition from healthy to degraded, got from %q", tr.From)
			}
		}
	}
	if !foundDegraded {
		t.Error("expected a transition to degraded status")
	}
}

func TestHealthMonitor_GetAllReports(t *testing.T) {
	hm := NewHealthMonitor(time.Minute, nil)

	checkerA := newTestChecker(HealthStatusHealthy, "")
	checkerB := newTestChecker(HealthStatusDegraded, "")
	checkerC := newTestChecker(HealthStatusUnhealthy, "")

	hm.Register("plugin-a", checkerA)
	hm.Register("plugin-b", checkerB)
	hm.Register("plugin-c", checkerC)

	reports := hm.GetAllReports()
	if len(reports) != 3 {
		t.Fatalf("expected 3 reports, got %d", len(reports))
	}

	// All should be in unknown state since no check has run yet.
	for name, report := range reports {
		if report.Status != HealthStatusUnknown {
			t.Errorf("plugin %q: Status = %q, want %q (before check)", name, report.Status, HealthStatusUnknown)
		}
	}
}

func TestHealthMonitor_GetSummary(t *testing.T) {
	hm := NewHealthMonitor(time.Minute, nil)

	hm.Register("a", newTestChecker(HealthStatusHealthy, ""))
	hm.Register("b", newTestChecker(HealthStatusHealthy, ""))
	hm.Register("c", newTestChecker(HealthStatusHealthy, ""))

	summary := hm.GetSummary()
	if summary.TotalPlugins != 3 {
		t.Errorf("TotalPlugins = %d, want 3", summary.TotalPlugins)
	}
	// Before checks, all are unknown.
	if summary.Unknown != 3 {
		t.Errorf("Unknown = %d, want 3", summary.Unknown)
	}
}

func TestHealthMonitor_Unregister(t *testing.T) {
	hm := NewHealthMonitor(time.Minute, nil)

	hm.Register("removeme", newTestChecker(HealthStatusHealthy, ""))
	hm.Unregister("removeme")

	report := hm.GetReport("removeme")
	if report.Status != HealthStatusUnknown {
		t.Errorf("Status = %q, want %q after unregister", report.Status, HealthStatusUnknown)
	}

	reports := hm.GetAllReports()
	if len(reports) != 0 {
		t.Errorf("expected 0 reports after unregister, got %d", len(reports))
	}
}

func TestHealthMonitor_ConcurrentAccess(t *testing.T) {
	hm := NewHealthMonitor(time.Minute, nil)

	// Register several plugins.
	for i := 0; i < 10; i++ {
		name := "plugin-" + string(rune('a'+i))
		hm.Register(name, newTestChecker(HealthStatusHealthy, "ok"))
	}

	var wg sync.WaitGroup
	const goroutines = 50

	// Readers: get individual reports.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := "plugin-" + string(rune('a'+idx%10))
			hm.GetReport(name)
		}(i)
	}

	// Additional readers: get all reports.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hm.GetAllReports()
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
		t.Fatal("concurrent access test timed out (possible deadlock)")
	}
}

func TestHealthStatus_Values(t *testing.T) {
	// Verify the string values of health status constants.
	tests := []struct {
		status HealthStatus
		want   string
	}{
		{HealthStatusHealthy, "healthy"},
		{HealthStatusDegraded, "degraded"},
		{HealthStatusUnhealthy, "unhealthy"},
		{HealthStatusUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := string(tt.status)
			if got != tt.want {
				t.Errorf("string(status) = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHealthMonitor_InitialStatusUnknown(t *testing.T) {
	hm := NewHealthMonitor(time.Minute, nil)

	hm.Register("fresh", newTestChecker(HealthStatusHealthy, ""))

	// Before any check, the report should show unknown status.
	report := hm.GetReport("fresh")
	if report.Status != HealthStatusUnknown {
		t.Errorf("initial Status = %q, want %q", report.Status, HealthStatusUnknown)
	}
}

func TestHealthMonitor_DefaultInterval(t *testing.T) {
	// Zero interval should use the default of 60 seconds (constructor handles this).
	// Just verify we can create one without panic.
	hm := NewHealthMonitor(0, nil)
	if hm == nil {
		t.Fatal("NewHealthMonitor(0, nil) returned nil")
	}
}

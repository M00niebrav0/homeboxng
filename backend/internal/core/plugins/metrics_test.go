package plugins

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestMetricsCollector_RecordRequest(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordRequest("test-plugin", 100*time.Millisecond, nil)
	mc.RecordRequest("test-plugin", 200*time.Millisecond, nil)
	mc.RecordRequest("test-plugin", 150*time.Millisecond, nil)

	m := mc.GetMetrics("test-plugin")

	if m.RequestCount != 3 {
		t.Errorf("RequestCount = %d, want 3", m.RequestCount)
	}

	// Average should be (100+200+150)/3 = 150ms.
	expectedAvg := 150 * time.Millisecond
	if m.AverageLatency != expectedAvg {
		t.Errorf("AverageLatency = %v, want %v", m.AverageLatency, expectedAvg)
	}
}

func TestMetricsCollector_ErrorTracking(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordRequest("test-plugin", 10*time.Millisecond, errors.New("connection timeout"))
	mc.RecordRequest("test-plugin", 20*time.Millisecond, errors.New("database unavailable"))

	m := mc.GetMetrics("test-plugin")

	if m.ErrorCount != 2 {
		t.Errorf("ErrorCount = %d, want 2", m.ErrorCount)
	}
	if m.LastError != "database unavailable" {
		t.Errorf("LastError = %q, want %q", m.LastError, "database unavailable")
	}
	if m.LastErrorTime.IsZero() {
		t.Error("LastErrorTime should not be zero")
	}
}

func TestMetricsCollector_LatencyCalculation(t *testing.T) {
	mc := NewMetricsCollector()

	latencies := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
	}

	for _, l := range latencies {
		mc.RecordRequest("calc-plugin", l, nil)
	}

	m := mc.GetMetrics("calc-plugin")

	// Avg = (10+20+30+40+50)/5 = 30ms.
	expectedAvg := 30 * time.Millisecond
	if m.AverageLatency != expectedAvg {
		t.Errorf("AverageLatency = %v, want %v", m.AverageLatency, expectedAvg)
	}
}

func TestMetricsCollector_P95Latency(t *testing.T) {
	mc := NewMetricsCollector()

	// Record 100 requests with known latencies: 1ms, 2ms, ..., 100ms.
	for i := 1; i <= 100; i++ {
		mc.RecordRequest("p95-plugin", time.Duration(i)*time.Millisecond, nil)
	}

	m := mc.GetMetrics("p95-plugin")

	// P95 of [1..100] should be at index ceil(100*0.95)-1 = 94, which is 95ms.
	expectedP95 := 95 * time.Millisecond
	if m.P95Latency != expectedP95 {
		t.Errorf("P95Latency = %v, want %v", m.P95Latency, expectedP95)
	}
}

func TestMetricsCollector_P95Latency_SmallSample(t *testing.T) {
	mc := NewMetricsCollector()

	// Single request: P95 should be that request's latency.
	mc.RecordRequest("single", 42*time.Millisecond, nil)

	m := mc.GetMetrics("single")
	if m.P95Latency != 42*time.Millisecond {
		t.Errorf("P95Latency for single request = %v, want 42ms", m.P95Latency)
	}
}

func TestMetricsCollector_P95Latency_UnsortedInput(t *testing.T) {
	mc := NewMetricsCollector()

	// Record in non-sorted order.
	latencies := []time.Duration{
		500 * time.Millisecond,
		100 * time.Millisecond,
		300 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
	}

	for _, l := range latencies {
		mc.RecordRequest("unsorted", l, nil)
	}

	m := mc.GetMetrics("unsorted")

	// Sorted: [100, 200, 300, 400, 500]
	// P95 index: ceil(5*0.95)-1 = ceil(4.75)-1 = 5-1 = 4, value = 500ms.
	if m.P95Latency != 500*time.Millisecond {
		t.Errorf("P95Latency = %v, want 500ms", m.P95Latency)
	}
}

func TestMetricsCollector_ResetMetrics(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordRequest("reset-plugin", 100*time.Millisecond, nil)
	mc.RecordRequest("reset-plugin", 50*time.Millisecond, errors.New("oops"))

	mc.ResetMetrics("reset-plugin")

	m := mc.GetMetrics("reset-plugin")
	if m.RequestCount != 0 {
		t.Errorf("RequestCount after reset = %d, want 0", m.RequestCount)
	}
	if m.ErrorCount != 0 {
		t.Errorf("ErrorCount after reset = %d, want 0", m.ErrorCount)
	}
	if m.AverageLatency != 0 {
		t.Errorf("AverageLatency after reset = %v, want 0", m.AverageLatency)
	}
	if m.LastError != "" {
		t.Errorf("LastError after reset = %q, want empty", m.LastError)
	}
}

func TestMetricsCollector_MultiplePlugins(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordRequest("plugin-a", 100*time.Millisecond, nil)
	mc.RecordRequest("plugin-a", 200*time.Millisecond, nil)
	mc.RecordRequest("plugin-b", 50*time.Millisecond, nil)
	mc.RecordRequest("plugin-b", 30*time.Millisecond, errors.New("fail"))

	mA := mc.GetMetrics("plugin-a")
	mB := mc.GetMetrics("plugin-b")

	if mA.RequestCount != 2 {
		t.Errorf("plugin-a RequestCount = %d, want 2", mA.RequestCount)
	}
	if mA.ErrorCount != 0 {
		t.Errorf("plugin-a ErrorCount = %d, want 0", mA.ErrorCount)
	}
	if mB.RequestCount != 2 {
		t.Errorf("plugin-b RequestCount = %d, want 2", mB.RequestCount)
	}
	if mB.ErrorCount != 1 {
		t.Errorf("plugin-b ErrorCount = %d, want 1", mB.ErrorCount)
	}

	// Reset one plugin should not affect the other.
	mc.ResetMetrics("plugin-a")
	mA = mc.GetMetrics("plugin-a")
	mB = mc.GetMetrics("plugin-b")

	if mA.RequestCount != 0 {
		t.Errorf("plugin-a RequestCount after reset = %d, want 0", mA.RequestCount)
	}
	if mB.RequestCount != 2 {
		t.Errorf("plugin-b RequestCount should be unchanged = %d, want 2", mB.RequestCount)
	}
}

func TestMetricsCollector_GetAllMetrics(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordRequest("a", 10*time.Millisecond, nil)
	mc.RecordRequest("b", 20*time.Millisecond, nil)
	mc.RecordRequest("c", 30*time.Millisecond, nil)

	all := mc.GetAllMetrics()
	if len(all) != 3 {
		t.Fatalf("expected 3 plugins, got %d", len(all))
	}

	if all["a"].RequestCount != 1 {
		t.Errorf("plugin a RequestCount = %d, want 1", all["a"].RequestCount)
	}
	if all["b"].RequestCount != 1 {
		t.Errorf("plugin b RequestCount = %d, want 1", all["b"].RequestCount)
	}
	if all["c"].RequestCount != 1 {
		t.Errorf("plugin c RequestCount = %d, want 1", all["c"].RequestCount)
	}
}

func TestMetricsCollector_GetMetrics_Nonexistent(t *testing.T) {
	mc := NewMetricsCollector()

	m := mc.GetMetrics("nonexistent")
	if m.RequestCount != 0 || m.ErrorCount != 0 {
		t.Error("expected zero metrics for nonexistent plugin")
	}
}

func TestMetricsCollector_Uptime(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordRequest("uptime-test", 10*time.Millisecond, nil)

	time.Sleep(50 * time.Millisecond)

	m := mc.GetMetrics("uptime-test")
	if m.Uptime < 50*time.Millisecond {
		t.Errorf("Uptime = %v, expected at least 50ms", m.Uptime)
	}
	if m.StartTime.IsZero() {
		t.Error("StartTime should not be zero")
	}
}

func TestMetricsCollector_ConcurrentAccess(t *testing.T) {
	mc := NewMetricsCollector()

	var wg sync.WaitGroup
	const goroutines = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := "plugin"
			if idx%2 == 0 {
				mc.RecordRequest(name, time.Duration(idx)*time.Millisecond, nil)
			} else {
				mc.RecordRequest(name, time.Duration(idx)*time.Millisecond, errors.New("error"))
			}
		}(i)
	}

	// Also read concurrently.
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mc.GetMetrics("plugin")
			mc.GetAllMetrics()
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

	// Verify data consistency.
	m := mc.GetMetrics("plugin")
	if m.RequestCount != int64(goroutines) {
		t.Errorf("RequestCount = %d, want %d", m.RequestCount, goroutines)
	}
}

func TestMetricsCollector_SlidingWindow(t *testing.T) {
	// Create a collector with a small window.
	mc := NewMetricsCollectorWithWindow(5)

	// Record more than the window size.
	for i := 1; i <= 10; i++ {
		mc.RecordRequest("window-test", time.Duration(i)*time.Millisecond, nil)
	}

	m := mc.GetMetrics("window-test")
	if m.RequestCount != 10 {
		t.Errorf("RequestCount = %d, want 10", m.RequestCount)
	}

	// Average should be based on the last 5 entries (6,7,8,9,10 ms) = 8ms.
	expectedAvg := 8 * time.Millisecond
	if m.AverageLatency != expectedAvg {
		t.Errorf("AverageLatency = %v, want %v (sliding window of last 5)", m.AverageLatency, expectedAvg)
	}
}

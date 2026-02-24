package plugins

import (
	"math"
	"sort"
	"sync"
	"time"
)

const (
	// defaultSlidingWindowSize is the number of requests stored for latency percentile calculation.
	defaultSlidingWindowSize = 1000
)

// PluginMetrics contains runtime statistics for a single plugin.
type PluginMetrics struct {
	// RequestCount is the total number of requests processed.
	RequestCount int64 `json:"requestCount"`
	// ErrorCount is the total number of requests that resulted in errors.
	ErrorCount int64 `json:"errorCount"`
	// LastError is the message of the most recent error, if any.
	LastError string `json:"lastError,omitempty"`
	// LastErrorTime is when the most recent error occurred.
	LastErrorTime time.Time `json:"lastErrorTime,omitempty"`
	// AverageLatency is the mean request duration over the sliding window.
	AverageLatency time.Duration `json:"averageLatency"`
	// P95Latency is the 95th percentile request duration over the sliding window.
	P95Latency time.Duration `json:"p95Latency"`
	// StartTime is when the plugin started collecting metrics.
	StartTime time.Time `json:"startTime"`
	// Uptime is the duration since StartTime (computed at retrieval time).
	Uptime time.Duration `json:"uptime"`
}

// pluginMetricsState holds the mutable state for a single plugin's metrics.
type pluginMetricsState struct {
	requestCount int64
	errorCount   int64
	lastError    string
	lastErrorAt  time.Time
	startTime    time.Time
	latencies    []time.Duration // circular buffer
	latencyIdx   int             // next write index in the circular buffer
	latencyCount int             // total entries written (may exceed window size)
}

// MetricsCollector provides thread-safe per-plugin metrics collection
// using a sliding window for latency percentile calculations.
type MetricsCollector struct {
	mu      sync.RWMutex
	plugins map[string]*pluginMetricsState
	window  int
}

// NewMetricsCollector creates a MetricsCollector with the default sliding window size.
func NewMetricsCollector() *MetricsCollector {
	return NewMetricsCollectorWithWindow(defaultSlidingWindowSize)
}

// NewMetricsCollectorWithWindow creates a MetricsCollector with a custom sliding window size.
// If windowSize is zero or negative, defaults to 1000.
func NewMetricsCollectorWithWindow(windowSize int) *MetricsCollector {
	if windowSize <= 0 {
		windowSize = defaultSlidingWindowSize
	}
	return &MetricsCollector{
		plugins: make(map[string]*pluginMetricsState),
		window:  windowSize,
	}
}

// ensureState returns the state for a plugin, creating it if necessary.
// Must be called while holding the write lock.
func (mc *MetricsCollector) ensureState(pluginName string) *pluginMetricsState {
	state, ok := mc.plugins[pluginName]
	if !ok {
		state = &pluginMetricsState{
			startTime: time.Now(),
			latencies: make([]time.Duration, mc.window),
		}
		mc.plugins[pluginName] = state
	}
	return state
}

// RecordRequest records a single request for the named plugin.
// The duration is the time the request took. If err is non-nil, it is recorded as an error.
func (mc *MetricsCollector) RecordRequest(pluginName string, duration time.Duration, err error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	state := mc.ensureState(pluginName)
	state.requestCount++

	// Record latency in the circular buffer.
	state.latencies[state.latencyIdx] = duration
	state.latencyIdx = (state.latencyIdx + 1) % mc.window
	state.latencyCount++

	if err != nil {
		state.errorCount++
		state.lastError = err.Error()
		state.lastErrorAt = time.Now()
	}
}

// GetMetrics returns the current metrics snapshot for the named plugin.
// If the plugin has no recorded metrics, a zero-value PluginMetrics is returned
// with Uptime of zero.
func (mc *MetricsCollector) GetMetrics(pluginName string) PluginMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	state, ok := mc.plugins[pluginName]
	if !ok {
		return PluginMetrics{}
	}

	return mc.buildMetrics(state)
}

// GetAllMetrics returns a snapshot of metrics for every tracked plugin.
func (mc *MetricsCollector) GetAllMetrics() map[string]PluginMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make(map[string]PluginMetrics, len(mc.plugins))
	for name, state := range mc.plugins {
		result[name] = mc.buildMetrics(state)
	}
	return result
}

// ResetMetrics clears all recorded metrics for the named plugin and resets its start time.
func (mc *MetricsCollector) ResetMetrics(pluginName string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	delete(mc.plugins, pluginName)
}

// buildMetrics constructs a PluginMetrics snapshot from a pluginMetricsState.
// Must be called while holding at least a read lock.
func (mc *MetricsCollector) buildMetrics(state *pluginMetricsState) PluginMetrics {
	avg, p95 := mc.computeLatencies(state)

	return PluginMetrics{
		RequestCount:   state.requestCount,
		ErrorCount:     state.errorCount,
		LastError:      state.lastError,
		LastErrorTime:  state.lastErrorAt,
		AverageLatency: avg,
		P95Latency:     p95,
		StartTime:      state.startTime,
		Uptime:         time.Since(state.startTime),
	}
}

// computeLatencies calculates average and P95 latency from the sliding window.
// Must be called while holding at least a read lock.
func (mc *MetricsCollector) computeLatencies(state *pluginMetricsState) (avg, p95 time.Duration) {
	count := state.latencyCount
	if count == 0 {
		return 0, 0
	}

	// Determine how many entries in the buffer are valid.
	n := count
	if n > mc.window {
		n = mc.window
	}

	// Collect the valid latencies.
	values := make([]time.Duration, n)
	if count <= mc.window {
		// Buffer hasn't wrapped yet; entries are at indices [0, count).
		copy(values, state.latencies[:n])
	} else {
		// Buffer has wrapped; read in order starting from the oldest entry.
		start := state.latencyIdx // oldest entry
		for i := 0; i < n; i++ {
			values[i] = state.latencies[(start+i)%mc.window]
		}
	}

	// Calculate average.
	var total time.Duration
	for _, v := range values {
		total += v
	}
	avg = total / time.Duration(n)

	// Calculate P95 using sorted values.
	sort.Slice(values, func(i, j int) bool {
		return values[i] < values[j]
	})

	p95Idx := int(math.Ceil(float64(n)*0.95)) - 1
	if p95Idx < 0 {
		p95Idx = 0
	}
	if p95Idx >= n {
		p95Idx = n - 1
	}
	p95 = values[p95Idx]

	return avg, p95
}

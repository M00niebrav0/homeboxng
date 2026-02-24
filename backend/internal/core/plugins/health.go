package plugins

import (
	"context"
	"sync"
	"time"
)

// HealthStatus represents the health state of a plugin.
type HealthStatus string

const (
	// HealthStatusHealthy indicates the plugin is operating normally.
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusDegraded indicates the plugin is operational but experiencing issues.
	HealthStatusDegraded HealthStatus = "degraded"
	// HealthStatusUnhealthy indicates the plugin is not functioning correctly.
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	// HealthStatusUnknown indicates the health status has not been determined.
	HealthStatusUnknown HealthStatus = "unknown"
)

// CheckResult represents the outcome of a single health check within a plugin.
type CheckResult struct {
	// Name identifies this specific check (e.g., "database", "api_connection").
	Name string `json:"name"`
	// Status is the health status for this check.
	Status HealthStatus `json:"status"`
	// Message provides human-readable detail about this check.
	Message string `json:"message"`
	// Duration is how long this check took to execute.
	Duration time.Duration `json:"duration"`
}

// HealthReport contains the full health status of a single plugin.
type HealthReport struct {
	// PluginName is the name of the plugin this report belongs to.
	PluginName string `json:"pluginName"`
	// Status is the overall health status of the plugin.
	Status HealthStatus `json:"status"`
	// Message provides a human-readable summary of the plugin's health.
	Message string `json:"message"`
	// LastCheck is the time when this report was generated.
	LastCheck time.Time `json:"lastCheck"`
	// Uptime is how long the plugin has been running.
	Uptime time.Duration `json:"uptime"`
	// Checks contains individual check results keyed by check name.
	Checks map[string]CheckResult `json:"checks"`
}

// HealthSummary provides an aggregate view of all plugin health for API responses.
type HealthSummary struct {
	// TotalPlugins is the total number of registered plugins.
	TotalPlugins int `json:"totalPlugins"`
	// Healthy is the count of plugins with healthy status.
	Healthy int `json:"healthy"`
	// Degraded is the count of plugins with degraded status.
	Degraded int `json:"degraded"`
	// Unhealthy is the count of plugins with unhealthy status.
	Unhealthy int `json:"unhealthy"`
	// Unknown is the count of plugins with unknown status.
	Unknown int `json:"unknown"`
	// Reports contains the individual health reports for all plugins.
	Reports []HealthReport `json:"reports"`
}

// HealthChecker is implemented by plugins that support health monitoring.
type HealthChecker interface {
	// HealthCheck performs a health check and returns the plugin's current health report.
	HealthCheck(ctx context.Context) HealthReport
}

// StatusTransition represents a change in a plugin's health status.
type StatusTransition struct {
	PluginName string
	From       HealthStatus
	To         HealthStatus
	Report     HealthReport
}

// StatusTransitionCallback is invoked when a plugin's health status changes.
type StatusTransitionCallback func(transition StatusTransition)

// HealthMonitor periodically checks the health of all registered plugins
// and tracks status transitions.
type HealthMonitor struct {
	mu       sync.RWMutex
	plugins  map[string]HealthChecker
	reports  map[string]HealthReport
	interval time.Duration
	callback StatusTransitionCallback

	cancel context.CancelFunc
	done   chan struct{}
}

// NewHealthMonitor creates a new HealthMonitor with the specified check interval.
// If interval is zero or negative, defaults to 60 seconds.
func NewHealthMonitor(interval time.Duration, callback StatusTransitionCallback) *HealthMonitor {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	return &HealthMonitor{
		plugins:  make(map[string]HealthChecker),
		reports:  make(map[string]HealthReport),
		interval: interval,
		callback: callback,
	}
}

// Register adds a plugin to be monitored. The plugin must implement HealthChecker.
func (hm *HealthMonitor) Register(name string, checker HealthChecker) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.plugins[name] = checker
	hm.reports[name] = HealthReport{
		PluginName: name,
		Status:     HealthStatusUnknown,
		Message:    "Health check not yet performed",
		LastCheck:  time.Time{},
		Checks:     make(map[string]CheckResult),
	}
}

// Unregister removes a plugin from health monitoring.
func (hm *HealthMonitor) Unregister(name string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	delete(hm.plugins, name)
	delete(hm.reports, name)
}

// Start begins periodic health checking. The context controls the lifetime
// of the monitoring loop.
func (hm *HealthMonitor) Start(ctx context.Context) {
	monitorCtx, cancel := context.WithCancel(ctx)
	hm.mu.Lock()
	hm.cancel = cancel
	hm.done = make(chan struct{})
	hm.mu.Unlock()

	go hm.run(monitorCtx)
}

// Stop terminates the periodic health checking loop and waits for it to finish.
func (hm *HealthMonitor) Stop() {
	hm.mu.RLock()
	cancel := hm.cancel
	done := hm.done
	hm.mu.RUnlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

// GetReport returns the latest health report for a specific plugin.
// If the plugin is not registered, a report with Unknown status is returned.
func (hm *HealthMonitor) GetReport(pluginName string) HealthReport {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	report, ok := hm.reports[pluginName]
	if !ok {
		return HealthReport{
			PluginName: pluginName,
			Status:     HealthStatusUnknown,
			Message:    "Plugin not registered for health monitoring",
			LastCheck:  time.Time{},
			Checks:     make(map[string]CheckResult),
		}
	}
	return report
}

// GetAllReports returns the latest health reports for all monitored plugins.
func (hm *HealthMonitor) GetAllReports() map[string]HealthReport {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	result := make(map[string]HealthReport, len(hm.reports))
	for name, report := range hm.reports {
		result[name] = report
	}
	return result
}

// GetSummary returns an aggregate health summary across all monitored plugins.
func (hm *HealthMonitor) GetSummary() HealthSummary {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	summary := HealthSummary{
		TotalPlugins: len(hm.reports),
		Reports:      make([]HealthReport, 0, len(hm.reports)),
	}

	for _, report := range hm.reports {
		summary.Reports = append(summary.Reports, report)
		switch report.Status {
		case HealthStatusHealthy:
			summary.Healthy++
		case HealthStatusDegraded:
			summary.Degraded++
		case HealthStatusUnhealthy:
			summary.Unhealthy++
		default:
			summary.Unknown++
		}
	}

	return summary
}

// run is the main monitoring loop that periodically checks all plugins.
func (hm *HealthMonitor) run(ctx context.Context) {
	defer close(hm.done)

	// Run an initial check immediately.
	hm.checkAll(ctx)

	ticker := time.NewTicker(hm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hm.checkAll(ctx)
		}
	}
}

// checkAll performs a health check on every registered plugin.
func (hm *HealthMonitor) checkAll(ctx context.Context) {
	hm.mu.RLock()
	plugins := make(map[string]HealthChecker, len(hm.plugins))
	for name, checker := range hm.plugins {
		plugins[name] = checker
	}
	hm.mu.RUnlock()

	for name, checker := range plugins {
		checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		report := checker.HealthCheck(checkCtx)
		cancel()

		report.PluginName = name

		hm.mu.Lock()
		previous, exists := hm.reports[name]
		hm.reports[name] = report
		hm.mu.Unlock()

		// Notify on status transitions.
		if hm.callback != nil && exists && previous.Status != report.Status {
			hm.callback(StatusTransition{
				PluginName: name,
				From:       previous.Status,
				To:         report.Status,
				Report:     report,
			})
		}
	}
}

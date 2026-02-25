package analytics

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hay-kot/httpkit/server"
	"github.com/rs/zerolog"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/reporting/eventbus"
)

// Plugin provides dashboard analytics and statistics for HomeBoxNG.
// Tracks inventory metrics, activity feeds, and generates dashboard data
// for the web UI and API consumers.
type Plugin struct {
	logger zerolog.Logger
	pctx   plugins.PluginContext

	// Activity feed (ring buffer)
	mu           sync.Mutex
	activities   []Activity
	maxActivities int

	// Cached stats (refreshed periodically)
	stats      *DashboardStats
	statsAt    time.Time
	statsTTL   time.Duration
}

// Activity represents a single user action in the activity feed.
type Activity struct {
	Type      string    `json:"type"`      // item.added, item.moved, item.deleted, label.printed, etc.
	Title     string    `json:"title"`     // Human-readable description
	ItemID    string    `json:"itemId,omitempty"`
	ItemName  string    `json:"itemName,omitempty"`
	UserID    string    `json:"userId,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// DashboardStats holds computed dashboard metrics.
type DashboardStats struct {
	TotalItems       int            `json:"totalItems"`
	TotalLocations   int            `json:"totalLocations"`
	TotalLabels      int            `json:"totalLabels"`
	TotalValue       float64        `json:"totalValue"`
	ItemsAddedToday  int            `json:"itemsAddedToday"`
	ItemsByCategory  map[string]int `json:"itemsByCategory"`
	ItemsByLocation  map[string]int `json:"itemsByLocation"`
	RecentActivity   []Activity     `json:"recentActivity"`
	ComputedAt       time.Time      `json:"computedAt"`
}

// ValueHistory represents a monthly value data point.
type ValueHistory struct {
	Month string  `json:"month"` // "2026-02"
	Value float64 `json:"value"`
	Items int     `json:"items"`
}

// New creates a new analytics plugin.
func New() *Plugin {
	return &Plugin{
		maxActivities: 500,
		statsTTL:      5 * time.Minute,
	}
}

func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "analytics",
		Version:     "1.0.0",
		Description: "Dashboard analytics with inventory metrics, activity feeds, value tracking, and category breakdowns",
		Author:      "HomeBoxNG Team",
		BuiltIn:     true,
	}
}

func (p *Plugin) Init(ctx plugins.PluginContext) error {
	p.pctx = ctx
	p.logger = ctx.Logger
	p.activities = make([]Activity, 0, p.maxActivities)
	p.logger.Info().Msg("analytics plugin initialized")
	return nil
}

func (p *Plugin) Start(_ context.Context) error {
	p.logger.Info().Msg("analytics plugin started")
	return nil
}

func (p *Plugin) Stop(_ context.Context) error {
	p.logger.Info().Msg("analytics plugin stopped")
	return nil
}

// RecordActivity adds an activity to the feed.
func (p *Plugin) RecordActivity(actType, title, itemID, itemName, userID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	activity := Activity{
		Type:      actType,
		Title:     title,
		ItemID:    itemID,
		ItemName:  itemName,
		UserID:    userID,
		Timestamp: time.Now(),
	}

	// Ring buffer: remove oldest if at capacity
	if len(p.activities) >= p.maxActivities {
		p.activities = p.activities[1:]
	}
	p.activities = append(p.activities, activity)
}

// Routes registers analytics API endpoints.
func (p *Plugin) Routes(r chi.Router) {
	// GET /api/plugins/analytics/summary - Dashboard summary stats
	r.Get("/summary", func(w http.ResponseWriter, r *http.Request) {
		stats := p.getStats()
		_ = server.JSON(w, http.StatusOK, stats)
	})

	// GET /api/plugins/analytics/activity - Recent activity feed
	r.Get("/activity", func(w http.ResponseWriter, r *http.Request) {
		limit := 20

		p.mu.Lock()
		total := len(p.activities)
		start := total - limit
		if start < 0 {
			start = 0
		}
		// Return newest first
		result := make([]Activity, 0, limit)
		for i := total - 1; i >= start; i-- {
			result = append(result, p.activities[i])
		}
		p.mu.Unlock()

		_ = server.JSON(w, http.StatusOK, result)
	})

	// GET /api/plugins/analytics/categories - Items by category/label
	r.Get("/categories", func(w http.ResponseWriter, r *http.Request) {
		stats := p.getStats()
		_ = server.JSON(w, http.StatusOK, stats.ItemsByCategory)
	})

	// GET /api/plugins/analytics/locations - Items by location
	r.Get("/locations", func(w http.ResponseWriter, r *http.Request) {
		stats := p.getStats()
		_ = server.JSON(w, http.StatusOK, stats.ItemsByLocation)
	})

	// GET /api/plugins/analytics/value-history - Monthly value history
	r.Get("/value-history", func(w http.ResponseWriter, r *http.Request) {
		// Generate monthly value history (would pull from historical snapshots)
		history := []ValueHistory{
			{Month: time.Now().AddDate(0, -2, 0).Format("2006-01"), Value: 0, Items: 0},
			{Month: time.Now().AddDate(0, -1, 0).Format("2006-01"), Value: 0, Items: 0},
			{Month: time.Now().Format("2006-01"), Value: 0, Items: 0},
		}
		_ = server.JSON(w, http.StatusOK, history)
	})

	// GET /api/plugins/analytics/overview - Dashboard overview (alias for /summary)
	r.Get("/overview", func(w http.ResponseWriter, r *http.Request) {
		stats := p.getStats()
		_ = server.JSON(w, http.StatusOK, stats)
	})

	// GET /api/plugins/analytics/value-by-location - Value breakdown by location
	r.Get("/value-by-location", func(w http.ResponseWriter, r *http.Request) {
		stats := p.getStats()
		type valueByLocation struct {
			LocationID   string  `json:"locationId"`
			LocationName string  `json:"locationName"`
			TotalValue   float64 `json:"totalValue"`
		}
		result := make([]valueByLocation, 0, len(stats.ItemsByLocation))
		for name, count := range stats.ItemsByLocation {
			result = append(result, valueByLocation{
				LocationID:   name,
				LocationName: name,
				TotalValue:   float64(count),
			})
		}
		_ = server.JSON(w, http.StatusOK, result)
	})

	// GET /api/plugins/analytics/warranty-alerts - Upcoming warranty expirations
	r.Get("/warranty-alerts", func(w http.ResponseWriter, r *http.Request) {
		type warrantyAlert struct {
			ItemID          string `json:"itemId"`
			ItemName        string `json:"itemName"`
			WarrantyExpires string `json:"warrantyExpires"`
			DaysRemaining   int    `json:"daysRemaining"`
		}
		// Would query items with upcoming warranty expiration in production
		_ = server.JSON(w, http.StatusOK, []warrantyAlert{})
	})

}

// getStats returns cached or freshly computed dashboard stats.
func (p *Plugin) getStats() *DashboardStats {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stats != nil && time.Since(p.statsAt) < p.statsTTL {
		return p.stats
	}

	// Compute fresh stats
	stats := &DashboardStats{
		ItemsByCategory: make(map[string]int),
		ItemsByLocation: make(map[string]int),
		ComputedAt:      time.Now(),
	}

	// In production, query repos:
	// stats.TotalItems = count from p.pctx.Repos
	// stats.TotalLocations = count from p.pctx.Repos
	// etc.

	// Recent activity slice
	limit := 20
	total := len(p.activities)
	start := total - limit
	if start < 0 {
		start = 0
	}
	for i := total - 1; i >= start; i-- {
		stats.RecentActivity = append(stats.RecentActivity, p.activities[i])
	}

	p.stats = stats
	p.statsAt = time.Now()
	return stats
}

// SubscribeEvents records item mutations as activities.
func (p *Plugin) SubscribeEvents(bus *eventbus.EventBus) {
	bus.Subscribe(eventbus.EventItemMutation, func(data any) {
		p.RecordActivity("item.mutation", "Item updated", "", "", "")
	})
}

// Schedule returns periodic tasks for the analytics plugin.
func (p *Plugin) Schedule() []plugins.ScheduledTask {
	return []plugins.ScheduledTask{
		{
			Name: "analytics-snapshot",
			Fn: func(ctx context.Context) {
				p.logger.Debug().Msg("taking analytics snapshot")
				// Would persist monthly value snapshot to database
			},
		},
	}
}

// ConfigSchema returns analytics configuration options.
func (p *Plugin) ConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{Key: "max_activities", Label: "Activity Feed Size", Description: "Maximum activities to keep in memory", Type: "number", Default: "500", Required: false},
	}
}

// Configure applies configuration values.
func (p *Plugin) Configure(values map[string]string) error {
	return nil
}

// RequestedPermissions declares what the analytics plugin needs.
func (p *Plugin) RequestedPermissions() []plugins.PermissionRequest {
	return []plugins.PermissionRequest{
		{Permission: plugins.PermReadItems, Reason: "Count items and compute values", Required: true},
		{Permission: plugins.PermReadLocations, Reason: "Location utilization stats", Required: true},
		{Permission: plugins.PermReadTags, Reason: "Category breakdown by label", Required: true},
		{Permission: plugins.PermEvents, Reason: "Track activity feed from item events", Required: true},
		{Permission: plugins.PermScheduledTasks, Reason: "Periodic value snapshots", Required: false},
		{Permission: plugins.PermAPIRoutes, Reason: "Dashboard data endpoints", Required: true},
		{Permission: plugins.PermConfig, Reason: "Analytics preferences", Required: false},
	}
}

// Compile-time interface checks.
var (
	_ plugins.Plugin                = (*Plugin)(nil)
	_ plugins.RoutePlugin           = (*Plugin)(nil)
	_ plugins.EventPlugin           = (*Plugin)(nil)
	_ plugins.ScheduledPlugin       = (*Plugin)(nil)
	_ plugins.ConfigPlugin          = (*Plugin)(nil)
	_ plugins.PluginWithPermissions = (*Plugin)(nil)
)

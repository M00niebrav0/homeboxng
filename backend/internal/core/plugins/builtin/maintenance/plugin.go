package maintenance

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hay-kot/httpkit/server"
	"github.com/rs/zerolog"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/reporting/eventbus"
)

// Plugin provides enhanced maintenance scheduling, repair logging,
// and service provider management for HomeBoxNG items.
type Plugin struct {
	logger zerolog.Logger
	pctx   plugins.PluginContext

	mu        sync.Mutex
	schedules []Schedule
	repairs   []Repair
	providers []ServiceProvider
}

// Schedule defines a recurring maintenance task for an item.
type Schedule struct {
	ID           string     `json:"id"`
	ItemID       string     `json:"itemId"`
	ItemName     string     `json:"itemName"`
	Description  string     `json:"description"`
	IntervalDays int        `json:"intervalDays"`
	LastDoneAt   *time.Time `json:"lastDoneAt,omitempty"`
	NextDueAt    time.Time  `json:"nextDueAt"`
	SnoozedUntil *time.Time `json:"snoozedUntil,omitempty"`
	ProviderID   string     `json:"providerId,omitempty"`
}

// Repair records a single repair event.
type Repair struct {
	ID          string    `json:"id"`
	ItemID      string    `json:"itemId"`
	ItemName    string    `json:"itemName"`
	Cost        float64   `json:"cost"`
	Description string    `json:"description"`
	ProviderID  string    `json:"providerId,omitempty"`
	Provider    string    `json:"provider,omitempty"`
	Date        time.Time `json:"date"`
	ReceiptURL  string    `json:"receiptUrl,omitempty"`
}

// ServiceProvider stores technician/vendor contact info.
type ServiceProvider struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Phone     string   `json:"phone,omitempty"`
	Email     string   `json:"email,omitempty"`
	Specialty []string `json:"specialty,omitempty"` // hvac, plumbing, electrical, etc.
}

// New creates a new maintenance plugin.
func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "maintenance",
		Version:     "1.0.0",
		Description: "Enhanced maintenance scheduling with repair logging, cost tracking, and service provider management",
		Author:      "HomeBoxNG Team",
		BuiltIn:     true,
	}
}

func (p *Plugin) Init(ctx plugins.PluginContext) error {
	p.pctx = ctx
	p.logger = ctx.Logger
	p.logger.Info().Msg("maintenance plugin initialized")
	return nil
}

func (p *Plugin) Start(_ context.Context) error  { return nil }
func (p *Plugin) Stop(_ context.Context) error   { return nil }

// Routes registers maintenance API endpoints.
func (p *Plugin) Routes(r chi.Router) {
	// GET /api/plugins/maintenance/upcoming - Upcoming maintenance
	r.Get("/upcoming", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		ahead := now.AddDate(0, 0, 30) // 30 days ahead

		p.mu.Lock()
		var upcoming []Schedule
		for _, s := range p.schedules {
			if s.NextDueAt.Before(ahead) {
				if s.SnoozedUntil != nil && s.SnoozedUntil.After(now) {
					continue
				}
				upcoming = append(upcoming, s)
			}
		}
		p.mu.Unlock()

		_ = server.JSON(w, http.StatusOK, upcoming)
	})

	// GET /api/plugins/maintenance/overdue - Overdue maintenance
	r.Get("/overdue", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()

		p.mu.Lock()
		var overdue []Schedule
		for _, s := range p.schedules {
			if s.NextDueAt.Before(now) {
				overdue = append(overdue, s)
			}
		}
		p.mu.Unlock()

		_ = server.JSON(w, http.StatusOK, overdue)
	})

	// POST /api/plugins/maintenance/schedules - Create maintenance schedule
	r.Post("/schedules", func(w http.ResponseWriter, r *http.Request) {
		var s Schedule
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		s.ID = time.Now().Format("20060102150405")
		if s.NextDueAt.IsZero() {
			s.NextDueAt = time.Now().AddDate(0, 0, s.IntervalDays)
		}

		p.mu.Lock()
		p.schedules = append(p.schedules, s)
		p.mu.Unlock()

		_ = server.JSON(w, http.StatusCreated, s)
	})

	// POST /api/plugins/maintenance/schedules/{id}/complete - Mark done
	r.Post("/schedules/{id}/complete", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		now := time.Now()

		p.mu.Lock()
		for i := range p.schedules {
			if p.schedules[i].ID == id {
				p.schedules[i].LastDoneAt = &now
				p.schedules[i].NextDueAt = now.AddDate(0, 0, p.schedules[i].IntervalDays)
				p.schedules[i].SnoozedUntil = nil
				p.mu.Unlock()
				_ = server.JSON(w, http.StatusOK, p.schedules[i])
				return
			}
		}
		p.mu.Unlock()
		_ = server.JSON(w, http.StatusNotFound, map[string]string{"error": "schedule not found"})
	})

	// POST /api/plugins/maintenance/schedules/{id}/snooze - Snooze reminder
	r.Post("/schedules/{id}/snooze", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var body struct {
			Days int `json:"days"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.Days <= 0 {
			body.Days = 7
		}
		snoozed := time.Now().AddDate(0, 0, body.Days)

		p.mu.Lock()
		for i := range p.schedules {
			if p.schedules[i].ID == id {
				p.schedules[i].SnoozedUntil = &snoozed
				p.mu.Unlock()
				_ = server.JSON(w, http.StatusOK, p.schedules[i])
				return
			}
		}
		p.mu.Unlock()
		_ = server.JSON(w, http.StatusNotFound, map[string]string{"error": "schedule not found"})
	})

	// --- Repairs ---
	r.Get("/repairs", func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		repairs := make([]Repair, len(p.repairs))
		copy(repairs, p.repairs)
		p.mu.Unlock()
		_ = server.JSON(w, http.StatusOK, repairs)
	})

	r.Post("/repairs", func(w http.ResponseWriter, r *http.Request) {
		var repair Repair
		if err := json.NewDecoder(r.Body).Decode(&repair); err != nil {
			_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		repair.ID = time.Now().Format("20060102150405")
		if repair.Date.IsZero() {
			repair.Date = time.Now()
		}

		p.mu.Lock()
		p.repairs = append(p.repairs, repair)
		p.mu.Unlock()

		_ = server.JSON(w, http.StatusCreated, repair)
	})

	r.Get("/repairs/cost/{itemId}", func(w http.ResponseWriter, r *http.Request) {
		itemID := chi.URLParam(r, "itemId")
		var total float64

		p.mu.Lock()
		for _, repair := range p.repairs {
			if repair.ItemID == itemID {
				total += repair.Cost
			}
		}
		p.mu.Unlock()

		_ = server.JSON(w, http.StatusOK, map[string]any{"itemId": itemID, "totalCost": total})
	})

	// --- Service Providers ---
	r.Get("/providers", func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		providers := make([]ServiceProvider, len(p.providers))
		copy(providers, p.providers)
		p.mu.Unlock()
		_ = server.JSON(w, http.StatusOK, providers)
	})

	r.Post("/providers", func(w http.ResponseWriter, r *http.Request) {
		var sp ServiceProvider
		if err := json.NewDecoder(r.Body).Decode(&sp); err != nil {
			_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		sp.ID = time.Now().Format("20060102150405")

		p.mu.Lock()
		p.providers = append(p.providers, sp)
		p.mu.Unlock()

		_ = server.JSON(w, http.StatusCreated, sp)
	})
}

// SubscribeEvents listens for item events.
func (p *Plugin) SubscribeEvents(bus *eventbus.EventBus) {}

// Schedule returns periodic tasks for due date checking.
func (p *Plugin) Schedule() []plugins.ScheduledTask {
	return []plugins.ScheduledTask{
		{
			Name: "maintenance-due-check",
			Fn: func(ctx context.Context) {
				now := time.Now()
				p.mu.Lock()
				for _, s := range p.schedules {
					if s.NextDueAt.Before(now) {
						p.logger.Warn().Str("item", s.ItemName).Str("desc", s.Description).Msg("maintenance overdue")
					}
				}
				p.mu.Unlock()
			},
		},
	}
}

func (p *Plugin) ConfigSchema() []plugins.ConfigField { return nil }
func (p *Plugin) Configure(_ map[string]string) error  { return nil }

func (p *Plugin) RequestedPermissions() []plugins.PermissionRequest {
	return []plugins.PermissionRequest{
		{Permission: plugins.PermMaintenance, Reason: "Create and manage maintenance schedules", Required: true},
		{Permission: plugins.PermReadItems, Reason: "Read items for maintenance tracking", Required: true},
		{Permission: plugins.PermAPIRoutes, Reason: "Maintenance management endpoints", Required: true},
		{Permission: plugins.PermScheduledTasks, Reason: "Periodic due date checking", Required: false},
	}
}

var (
	_ plugins.Plugin                = (*Plugin)(nil)
	_ plugins.RoutePlugin           = (*Plugin)(nil)
	_ plugins.EventPlugin           = (*Plugin)(nil)
	_ plugins.ScheduledPlugin       = (*Plugin)(nil)
	_ plugins.ConfigPlugin          = (*Plugin)(nil)
	_ plugins.PluginWithPermissions = (*Plugin)(nil)
)

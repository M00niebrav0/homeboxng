package lending

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

// Plugin tracks items lent to friends, family, and neighbors
// with return reminders and borrower contact management.
type Plugin struct {
	logger zerolog.Logger
	pctx   plugins.PluginContext

	mu        sync.Mutex
	loans     []Loan
	borrowers []Borrower
}

// Loan represents an item checked out to a borrower.
type Loan struct {
	ID         string     `json:"id"`
	ItemID     string     `json:"itemId"`
	ItemName   string     `json:"itemName"`
	BorrowerID string     `json:"borrowerId"`
	Borrower   string     `json:"borrower"`
	LentAt     time.Time  `json:"lentAt"`
	DueAt      *time.Time `json:"dueAt,omitempty"`
	ReturnedAt *time.Time `json:"returnedAt,omitempty"`
	Notes      string     `json:"notes,omitempty"`
	Condition  string     `json:"condition,omitempty"` // good, fair, damaged
}

// Borrower stores contact info for people who borrow items.
type Borrower struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Phone string `json:"phone,omitempty"`
	Email string `json:"email,omitempty"`
}

// New creates a new lending plugin.
func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "lending",
		Version:     "1.0.0",
		Description: "Item lending and checkout tracking with return reminders and borrower management",
		Author:      "HomeBoxNG Team",
		BuiltIn:     true,
	}
}

func (p *Plugin) Init(ctx plugins.PluginContext) error {
	p.pctx = ctx
	p.logger = ctx.Logger
	p.logger.Info().Msg("lending plugin initialized")
	return nil
}

func (p *Plugin) Start(_ context.Context) error  { return nil }
func (p *Plugin) Stop(_ context.Context) error   { return nil }

// Routes registers lending API endpoints.
func (p *Plugin) Routes(r chi.Router) {
	// GET /api/plugins/lending/loans - Active loans
	r.Get("/loans", func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		var active []Loan
		for _, l := range p.loans {
			if l.ReturnedAt == nil {
				active = append(active, l)
			}
		}
		p.mu.Unlock()
		_ = server.JSON(w, http.StatusOK, active)
	})

	// GET /api/plugins/lending/overdue - Overdue loans
	r.Get("/overdue", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		p.mu.Lock()
		var overdue []Loan
		for _, l := range p.loans {
			if l.ReturnedAt == nil && l.DueAt != nil && l.DueAt.Before(now) {
				overdue = append(overdue, l)
			}
		}
		p.mu.Unlock()
		_ = server.JSON(w, http.StatusOK, overdue)
	})

	// POST /api/plugins/lending/lend - Lend an item
	r.Post("/lend", func(w http.ResponseWriter, r *http.Request) {
		var loan Loan
		if err := json.NewDecoder(r.Body).Decode(&loan); err != nil {
			_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		loan.LentAt = time.Now()
		loan.ID = time.Now().Format("20060102150405")

		p.mu.Lock()
		p.loans = append(p.loans, loan)
		p.mu.Unlock()

		p.logger.Info().Str("item", loan.ItemName).Str("to", loan.Borrower).Msg("item lent")
		_ = server.JSON(w, http.StatusCreated, loan)
	})

	// POST /api/plugins/lending/return/{id} - Return an item
	r.Post("/return/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		now := time.Now()

		p.mu.Lock()
		for i := range p.loans {
			if p.loans[i].ID == id {
				p.loans[i].ReturnedAt = &now
				p.mu.Unlock()
				_ = server.JSON(w, http.StatusOK, p.loans[i])
				return
			}
		}
		p.mu.Unlock()
		_ = server.JSON(w, http.StatusNotFound, map[string]string{"error": "loan not found"})
	})

	// GET /api/plugins/lending/history - All loans (including returned)
	r.Get("/history", func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		loans := make([]Loan, len(p.loans))
		copy(loans, p.loans)
		p.mu.Unlock()
		_ = server.JSON(w, http.StatusOK, loans)
	})

	// --- Borrower Management ---
	r.Get("/borrowers", func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		b := make([]Borrower, len(p.borrowers))
		copy(b, p.borrowers)
		p.mu.Unlock()
		_ = server.JSON(w, http.StatusOK, b)
	})

	r.Post("/borrowers", func(w http.ResponseWriter, r *http.Request) {
		var b Borrower
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		b.ID = time.Now().Format("20060102150405")

		p.mu.Lock()
		p.borrowers = append(p.borrowers, b)
		p.mu.Unlock()
		_ = server.JSON(w, http.StatusCreated, b)
	})
}

// SubscribeEvents listens for item deletion to warn about lent items.
func (p *Plugin) SubscribeEvents(bus *eventbus.EventBus) {
	bus.Subscribe(eventbus.EventItemMutation, func(data any) {
		p.logger.Debug().Msg("lending: item mutation received")
	})
}

// Schedule returns periodic tasks for overdue reminders.
func (p *Plugin) Schedule() []plugins.ScheduledTask {
	return []plugins.ScheduledTask{
		{
			Name: "lending-overdue-check",
			Fn: func(ctx context.Context) {
				now := time.Now()
				p.mu.Lock()
				for _, l := range p.loans {
					if l.ReturnedAt == nil && l.DueAt != nil && l.DueAt.Before(now) {
						p.logger.Warn().Str("item", l.ItemName).Str("borrower", l.Borrower).Msg("item overdue")
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
		{Permission: plugins.PermReadItems, Reason: "Read items for lending checkout", Required: true},
		{Permission: plugins.PermWriteItems, Reason: "Update item notes with lending status", Required: false},
		{Permission: plugins.PermAPIRoutes, Reason: "Lending management endpoints", Required: true},
		{Permission: plugins.PermEvents, Reason: "Warn when lent items are modified", Required: false},
		{Permission: plugins.PermScheduledTasks, Reason: "Overdue return reminders", Required: false},
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

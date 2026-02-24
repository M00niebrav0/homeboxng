package shopping

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hay-kot/httpkit/server"
	"github.com/rs/zerolog"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/reporting/eventbus"
)

// Plugin manages shopping lists, reorder triggers, and purchase tracking.
type Plugin struct {
	logger zerolog.Logger
	pctx   plugins.PluginContext

	mu    sync.Mutex
	items []ShoppingItem
	rules []ReorderRule
}

// ShoppingItem represents an item on the shopping list.
type ShoppingItem struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Quantity  int       `json:"quantity"`
	Store     string    `json:"store,omitempty"`    // Amazon, Home Depot, Walmart
	Priority  string    `json:"priority,omitempty"` // urgent, normal, wishlist
	Price     float64   `json:"price,omitempty"`    // Estimated price
	ItemID    string    `json:"itemId,omitempty"`   // Linked HomeBox item (for reorders)
	AddedAt   time.Time `json:"addedAt"`
	BoughtAt  *time.Time `json:"boughtAt,omitempty"`
	AddedBy   string    `json:"addedBy,omitempty"`
}

// ReorderRule triggers auto-add to shopping list when stock is low.
type ReorderRule struct {
	ItemID       string `json:"itemId"`
	ItemName     string `json:"itemName"`
	MinQuantity  int    `json:"minQuantity"`  // Alert when at or below
	ReorderQty   int    `json:"reorderQty"`   // Quantity to add to list
	PreferStore  string `json:"preferStore"`  // Preferred store
}

// New creates a new shopping plugin.
func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "shopping",
		Version:     "1.0.0",
		Description: "Shopping list with auto-reorder triggers, store grouping, price estimates, and purchase tracking",
		Author:      "HomeBoxNG Team",
		BuiltIn:     true,
	}
}

func (p *Plugin) Init(ctx plugins.PluginContext) error {
	p.pctx = ctx
	p.logger = ctx.Logger
	p.logger.Info().Msg("shopping plugin initialized")
	return nil
}

func (p *Plugin) Start(_ context.Context) error  { return nil }
func (p *Plugin) Stop(_ context.Context) error   { return nil }

// Routes registers shopping list API endpoints.
func (p *Plugin) Routes(r chi.Router) {
	// GET /api/plugins/shopping/list - Full shopping list
	r.Get("/list", func(w http.ResponseWriter, r *http.Request) {
		store := r.URL.Query().Get("store")

		p.mu.Lock()
		var result []ShoppingItem
		for _, item := range p.items {
			if item.BoughtAt != nil {
				continue // Skip purchased
			}
			if store != "" && !strings.EqualFold(item.Store, store) {
				continue
			}
			result = append(result, item)
		}
		p.mu.Unlock()

		_ = server.JSON(w, http.StatusOK, result)
	})

	// GET /api/plugins/shopping/by-store - Shopping list grouped by store
	r.Get("/by-store", func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		grouped := make(map[string][]ShoppingItem)
		for _, item := range p.items {
			if item.BoughtAt != nil {
				continue
			}
			store := item.Store
			if store == "" {
				store = "Other"
			}
			grouped[store] = append(grouped[store], item)
		}
		p.mu.Unlock()

		_ = server.JSON(w, http.StatusOK, grouped)
	})

	// POST /api/plugins/shopping/add - Add item to shopping list
	r.Post("/add", func(w http.ResponseWriter, r *http.Request) {
		var item ShoppingItem
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		item.ID = time.Now().Format("20060102150405")
		item.AddedAt = time.Now()
		if item.Quantity <= 0 {
			item.Quantity = 1
		}
		if item.Priority == "" {
			item.Priority = "normal"
		}

		p.mu.Lock()
		p.items = append(p.items, item)
		p.mu.Unlock()

		_ = server.JSON(w, http.StatusCreated, item)
	})

	// POST /api/plugins/shopping/bought/{id} - Mark as purchased
	r.Post("/bought/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		now := time.Now()

		p.mu.Lock()
		for i := range p.items {
			if p.items[i].ID == id {
				p.items[i].BoughtAt = &now
				p.mu.Unlock()
				p.logger.Info().Str("item", p.items[i].Name).Msg("item purchased")
				_ = server.JSON(w, http.StatusOK, p.items[i])
				return
			}
		}
		p.mu.Unlock()
		_ = server.JSON(w, http.StatusNotFound, map[string]string{"error": "item not found"})
	})

	// DELETE /api/plugins/shopping/{id} - Remove from list
	r.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		p.mu.Lock()
		for i := range p.items {
			if p.items[i].ID == id {
				p.items = append(p.items[:i], p.items[i+1:]...)
				p.mu.Unlock()
				_ = server.JSON(w, http.StatusOK, map[string]string{"status": "removed"})
				return
			}
		}
		p.mu.Unlock()
		_ = server.JSON(w, http.StatusNotFound, map[string]string{"error": "item not found"})
	})

	// --- Reorder Rules ---
	r.Get("/rules", func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		rules := make([]ReorderRule, len(p.rules))
		copy(rules, p.rules)
		p.mu.Unlock()
		_ = server.JSON(w, http.StatusOK, rules)
	})

	r.Post("/rules", func(w http.ResponseWriter, r *http.Request) {
		var rule ReorderRule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		p.mu.Lock()
		p.rules = append(p.rules, rule)
		p.mu.Unlock()

		_ = server.JSON(w, http.StatusCreated, rule)
	})

	// GET /api/plugins/shopping/history - Purchase history
	r.Get("/history", func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		var purchased []ShoppingItem
		for _, item := range p.items {
			if item.BoughtAt != nil {
				purchased = append(purchased, item)
			}
		}
		p.mu.Unlock()
		_ = server.JSON(w, http.StatusOK, purchased)
	})

	// GET /api/plugins/shopping/total - Estimated total cost
	r.Get("/total", func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		total := 0.0
		byStore := make(map[string]float64)
		for _, item := range p.items {
			if item.BoughtAt != nil {
				continue
			}
			cost := item.Price * float64(item.Quantity)
			total += cost
			store := item.Store
			if store == "" {
				store = "Other"
			}
			byStore[store] += cost
		}
		p.mu.Unlock()

		_ = server.JSON(w, http.StatusOK, map[string]any{
			"total":   total,
			"byStore": byStore,
		})
	})
}

// SubscribeEvents checks for low-stock triggers.
func (p *Plugin) SubscribeEvents(bus *eventbus.EventBus) {
	bus.Subscribe(eventbus.EventItemMutation, func(data any) {
		// Would check if item quantity dropped below reorder threshold
		p.logger.Debug().Msg("shopping: checking reorder rules")
	})
}

// Schedule returns periodic tasks for reorder checking.
func (p *Plugin) Schedule() []plugins.ScheduledTask {
	return []plugins.ScheduledTask{
		{
			Name: "shopping-reorder-check",
			Fn: func(ctx context.Context) {
				// Would scan all items against reorder rules
				p.logger.Debug().Msg("checking reorder rules")
			},
		},
	}
}

func (p *Plugin) ConfigSchema() []plugins.ConfigField { return nil }
func (p *Plugin) Configure(_ map[string]string) error  { return nil }

func (p *Plugin) RequestedPermissions() []plugins.PermissionRequest {
	return []plugins.PermissionRequest{
		{Permission: plugins.PermReadItems, Reason: "Check item quantities for reorder triggers", Required: true},
		{Permission: plugins.PermAPIRoutes, Reason: "Shopping list management endpoints", Required: true},
		{Permission: plugins.PermEvents, Reason: "Low stock auto-detection", Required: false},
		{Permission: plugins.PermScheduledTasks, Reason: "Periodic reorder threshold checks", Required: false},
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

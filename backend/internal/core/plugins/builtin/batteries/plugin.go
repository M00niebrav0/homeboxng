package batteries

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/hay-kot/httpkit/server"
	"github.com/rs/zerolog"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins"
)

// Plugin provides power tool battery and charger tracking for HomeBoxNG.
// Supports Milwaukee M12/M18, Ryobi ONE+/40V, DeWalt 20V/60V, Makita 18V/40V,
// Bosch 12V/18V, Rigid 18V, Festool 18V, and generic lithium-ion batteries.
type Plugin struct {
	logger zerolog.Logger
	pctx   plugins.PluginContext
}

// BatteryPlatform represents a power tool battery ecosystem.
type BatteryPlatform struct {
	Slug         string           `json:"slug"`
	Brand        string           `json:"brand"`
	Name         string           `json:"name"`
	NominalVolts float64          `json:"nominalVolts"`
	Description  string           `json:"description"`
	Batteries    []BatteryModel   `json:"batteries"`
	Chargers     []ChargerModel   `json:"chargers"`
	Color        string           `json:"color"` // Brand color for UI
}

// BatteryModel is a specific battery SKU within a platform.
type BatteryModel struct {
	SKU       string  `json:"sku"`
	Label     string  `json:"label"`
	Ah        float64 `json:"ah"`
	HighOut   bool    `json:"highOutput"`
	Cells     string  `json:"cells,omitempty"`
	WeightLbs float64 `json:"weightLbs,omitempty"`
	Notes     string  `json:"notes,omitempty"`
}

// ChargerModel is a specific charger SKU within a platform.
type ChargerModel struct {
	SKU         string   `json:"sku"`
	Label       string   `json:"label"`
	Ports       int      `json:"ports"`
	Rapid       bool     `json:"rapid"`
	Platforms   []string `json:"platforms,omitempty"` // Cross-platform chargers
	Notes       string   `json:"notes,omitempty"`
}

// ============================================================================
// Static seed data — battery platforms, models, and chargers
// ============================================================================

var batteryPlatforms = []BatteryPlatform{
	{
		Slug: "milwaukee-m18", Brand: "Milwaukee", Name: "M18",
		NominalVolts: 18, Color: "#DB0032",
		Description: "Milwaukee M18 18V lithium-ion platform — 250+ tools",
		Batteries: []BatteryModel{
			{SKU: "48-11-1815", Label: "M18 CP 1.5Ah", Ah: 1.5},
			{SKU: "48-11-1820", Label: "M18 CP 2.0Ah", Ah: 2.0},
			{SKU: "48-11-1830", Label: "M18 CP 3.0Ah", Ah: 3.0},
			{SKU: "48-11-1840", Label: "M18 XC 4.0Ah", Ah: 4.0},
			{SKU: "48-11-1850", Label: "M18 XC 5.0Ah", Ah: 5.0},
			{SKU: "48-11-1860", Label: "M18 XC 6.0Ah", Ah: 6.0},
			{SKU: "48-11-1865", Label: "M18 XC 6.0Ah (2-Pack)", Ah: 6.0},
			{SKU: "48-11-1880", Label: "M18 HO 8.0Ah", Ah: 8.0, HighOut: true},
			{SKU: "48-11-1812", Label: "M18 HO 12.0Ah", Ah: 12.0, HighOut: true},
		},
		Chargers: []ChargerModel{
			{SKU: "48-59-1812", Label: "M18 Standard Charger", Ports: 1},
			{SKU: "48-59-1802", Label: "M18/M12 Rapid Charger", Ports: 1, Rapid: true, Platforms: []string{"milwaukee-m18", "milwaukee-m12"}},
			{SKU: "48-59-1807", Label: "M18 Dual Bay Rapid Charger", Ports: 2, Rapid: true},
			{SKU: "48-59-1808", Label: "M18 6-Bay Sequential Charger", Ports: 6},
		},
	},
	{
		Slug: "milwaukee-m12", Brand: "Milwaukee", Name: "M12",
		NominalVolts: 12, Color: "#DB0032",
		Description: "Milwaukee M12 12V compact lithium-ion platform — 100+ tools",
		Batteries: []BatteryModel{
			{SKU: "48-11-2401", Label: "M12 CP 1.5Ah", Ah: 1.5},
			{SKU: "48-11-2420", Label: "M12 CP 2.0Ah", Ah: 2.0},
			{SKU: "48-11-2430", Label: "M12 XC 3.0Ah", Ah: 3.0},
			{SKU: "48-11-2440", Label: "M12 XC 4.0Ah", Ah: 4.0},
			{SKU: "48-11-2460", Label: "M12 XC 6.0Ah", Ah: 6.0},
			{SKU: "48-11-2425", Label: "M12 HO 2.5Ah", Ah: 2.5, HighOut: true},
		},
		Chargers: []ChargerModel{
			{SKU: "48-59-2401", Label: "M12 Standard Charger", Ports: 1},
			{SKU: "48-59-2440", Label: "M12 Lithium-ion Charger", Ports: 1},
		},
	},
	{
		Slug: "milwaukee-mx-fuel", Brand: "Milwaukee", Name: "MX FUEL",
		NominalVolts: 72, Color: "#DB0032",
		Description: "Milwaukee MX FUEL — equipment-grade battery platform for heavy-duty tools",
		Batteries: []BatteryModel{
			{SKU: "MXFCP203", Label: "MX FUEL CP 3.0Ah", Ah: 3.0},
			{SKU: "MXFXC406", Label: "MX FUEL XC 6.0Ah", Ah: 6.0},
			{SKU: "MXF361-2XC", Label: "MX FUEL REDLITHIUM Battery Pack", Ah: 6.0},
		},
		Chargers: []ChargerModel{
			{SKU: "MXFC", Label: "MX FUEL Charger", Ports: 1},
			{SKU: "MXFSC", Label: "MX FUEL Super Charger", Ports: 1, Rapid: true},
		},
	},
	{
		Slug: "ryobi-one-plus", Brand: "Ryobi", Name: "ONE+ 18V",
		NominalVolts: 18, Color: "#78BE20",
		Description: "Ryobi ONE+ 18V platform — 300+ tools, most affordable ecosystem",
		Batteries: []BatteryModel{
			{SKU: "PBP003", Label: "ONE+ 1.5Ah", Ah: 1.5},
			{SKU: "PBP2002", Label: "ONE+ 2.0Ah (2-Pack)", Ah: 2.0},
			{SKU: "PBP004", Label: "ONE+ 4.0Ah", Ah: 4.0},
			{SKU: "PBP005", Label: "ONE+ HP 4.0Ah", Ah: 4.0, HighOut: true},
			{SKU: "PBP007", Label: "ONE+ 6.0Ah", Ah: 6.0},
			{SKU: "PBP010", Label: "ONE+ HP 6.0Ah", Ah: 6.0, HighOut: true},
		},
		Chargers: []ChargerModel{
			{SKU: "PCG002", Label: "ONE+ 18V Charger", Ports: 1},
			{SKU: "PCG004", Label: "ONE+ 18V Dual Charger", Ports: 2},
			{SKU: "PCG008", Label: "ONE+ 18V 6-Port Supercharger", Ports: 6, Rapid: true},
		},
	},
	{
		Slug: "ryobi-40v", Brand: "Ryobi", Name: "40V",
		NominalVolts: 40, Color: "#78BE20",
		Description: "Ryobi 40V outdoor power equipment platform — mowers, blowers, chainsaws",
		Batteries: []BatteryModel{
			{SKU: "OP4020", Label: "40V 2.0Ah", Ah: 2.0},
			{SKU: "OP4030", Label: "40V 3.0Ah", Ah: 3.0},
			{SKU: "OP4040", Label: "40V 4.0Ah", Ah: 4.0},
			{SKU: "OP40602", Label: "40V 6.0Ah", Ah: 6.0},
			{SKU: "OP40804", Label: "40V HP 8.0Ah", Ah: 8.0, HighOut: true},
		},
		Chargers: []ChargerModel{
			{SKU: "OP403A", Label: "40V Standard Charger", Ports: 1},
			{SKU: "OP406A", Label: "40V Rapid Charger", Ports: 1, Rapid: true},
		},
	},
	{
		Slug: "dewalt-20v-max", Brand: "DeWalt", Name: "20V MAX",
		NominalVolts: 20, Color: "#FEBD17",
		Description: "DeWalt 20V MAX platform — 200+ tools, FLEXVOLT compatible",
		Batteries: []BatteryModel{
			{SKU: "DCB201", Label: "20V MAX 1.5Ah", Ah: 1.5},
			{SKU: "DCB203", Label: "20V MAX 2.0Ah", Ah: 2.0},
			{SKU: "DCB230", Label: "20V MAX 3.0Ah", Ah: 3.0},
			{SKU: "DCB204", Label: "20V MAX XR 4.0Ah", Ah: 4.0},
			{SKU: "DCB205", Label: "20V MAX XR 5.0Ah", Ah: 5.0},
			{SKU: "DCB206", Label: "20V MAX XR 6.0Ah", Ah: 6.0},
			{SKU: "DCB208", Label: "20V MAX XR 8.0Ah", Ah: 8.0},
			{SKU: "DCB210", Label: "20V MAX XR 10.0Ah", Ah: 10.0},
			{SKU: "DCB612", Label: "FLEXVOLT 20V/60V 6.0/2.0Ah", Ah: 6.0, HighOut: true, Notes: "Dual-voltage FLEXVOLT"},
			{SKU: "DCB609", Label: "FLEXVOLT 20V/60V 9.0/3.0Ah", Ah: 9.0, HighOut: true, Notes: "Dual-voltage FLEXVOLT"},
			{SKU: "DCB615", Label: "FLEXVOLT 20V/60V 15.0/5.0Ah", Ah: 15.0, HighOut: true, Notes: "Dual-voltage FLEXVOLT"},
		},
		Chargers: []ChargerModel{
			{SKU: "DCB107", Label: "20V MAX Charger", Ports: 1},
			{SKU: "DCB112", Label: "20V MAX Charger", Ports: 1},
			{SKU: "DCB102", Label: "20V MAX Dual Port Charger", Ports: 2},
			{SKU: "DCB104", Label: "20V MAX 4-Port Charger", Ports: 4},
			{SKU: "DCB118", Label: "FLEXVOLT Fan-Cooled Fast Charger", Ports: 1, Rapid: true},
		},
	},
	{
		Slug: "makita-lxt", Brand: "Makita", Name: "LXT 18V",
		NominalVolts: 18, Color: "#00A3A1",
		Description: "Makita LXT 18V lithium-ion platform — 300+ tools, industry standard",
		Batteries: []BatteryModel{
			{SKU: "BL1815N", Label: "LXT 1.5Ah", Ah: 1.5},
			{SKU: "BL1820B", Label: "LXT 2.0Ah", Ah: 2.0},
			{SKU: "BL1830B", Label: "LXT 3.0Ah", Ah: 3.0},
			{SKU: "BL1840B", Label: "LXT 4.0Ah", Ah: 4.0},
			{SKU: "BL1850B", Label: "LXT 5.0Ah", Ah: 5.0},
			{SKU: "BL1860B", Label: "LXT 6.0Ah", Ah: 6.0},
		},
		Chargers: []ChargerModel{
			{SKU: "DC18RC", Label: "LXT Rapid Optimum Charger", Ports: 1, Rapid: true},
			{SKU: "DC18RD", Label: "LXT Dual Port Rapid Charger", Ports: 2, Rapid: true},
			{SKU: "DC18SF", Label: "LXT 4-Port Sequential Charger", Ports: 4},
		},
	},
	{
		Slug: "makita-xgt", Brand: "Makita", Name: "XGT 40V",
		NominalVolts: 40, Color: "#00A3A1",
		Description: "Makita XGT 40V MAX platform — next-gen high power tools",
		Batteries: []BatteryModel{
			{SKU: "BL4025", Label: "XGT 2.5Ah", Ah: 2.5},
			{SKU: "BL4040", Label: "XGT 4.0Ah", Ah: 4.0},
			{SKU: "BL4050F", Label: "XGT 5.0Ah", Ah: 5.0},
			{SKU: "BL4080F", Label: "XGT 8.0Ah", Ah: 8.0},
		},
		Chargers: []ChargerModel{
			{SKU: "DC40RA", Label: "XGT Rapid Optimum Charger", Ports: 1, Rapid: true},
			{SKU: "DC40RB", Label: "XGT Dual Port Rapid Charger", Ports: 2, Rapid: true},
		},
	},
	{
		Slug: "bosch-18v", Brand: "Bosch", Name: "CORE 18V",
		NominalVolts: 18, Color: "#005691",
		Description: "Bosch 18V CORE platform — professional and DIY power tools",
		Batteries: []BatteryModel{
			{SKU: "GBA18V20", Label: "CORE18V 2.0Ah", Ah: 2.0},
			{SKU: "GBA18V40", Label: "CORE18V 4.0Ah", Ah: 4.0},
			{SKU: "GBA18V60", Label: "CORE18V 6.0Ah", Ah: 6.0},
			{SKU: "GBA18V80", Label: "CORE18V 8.0Ah", Ah: 8.0, HighOut: true},
			{SKU: "GBA18V120", Label: "CORE18V 12.0Ah", Ah: 12.0, HighOut: true},
		},
		Chargers: []ChargerModel{
			{SKU: "GAL18V-40", Label: "18V Lithium-ion Charger", Ports: 1},
			{SKU: "GAL18V-160C", Label: "18V Turbo Charger", Ports: 1, Rapid: true},
		},
	},
	{
		Slug: "ridgid-18v", Brand: "Ridgid", Name: "18V",
		NominalVolts: 18, Color: "#FF6600",
		Description: "Ridgid 18V platform — lifetime service agreement, Home Depot exclusive",
		Batteries: []BatteryModel{
			{SKU: "R87002", Label: "18V 2.0Ah", Ah: 2.0},
			{SKU: "R87004", Label: "18V 4.0Ah", Ah: 4.0},
			{SKU: "R87006", Label: "18V MAX Output 6.0Ah", Ah: 6.0, HighOut: true},
			{SKU: "R87008", Label: "18V MAX Output 8.0Ah", Ah: 8.0, HighOut: true},
		},
		Chargers: []ChargerModel{
			{SKU: "R86092", Label: "18V Charger", Ports: 1},
			{SKU: "R86098", Label: "18V Dual Port Sequential Charger", Ports: 2},
		},
	},
	{
		Slug: "festool-18v", Brand: "Festool", Name: "18V",
		NominalVolts: 18, Color: "#1B1B1B",
		Description: "Festool 18V battery platform — premium woodworking and construction tools",
		Batteries: []BatteryModel{
			{SKU: "577400", Label: "BP 18 Li 4.0 Ah HPC-ASI", Ah: 4.0},
			{SKU: "577703", Label: "BP 18 Li 5.0 Ah ASI", Ah: 5.0},
			{SKU: "577702", Label: "BP 18 Li 8.0 Ah HPC-ASI", Ah: 8.0, HighOut: true},
		},
		Chargers: []ChargerModel{
			{SKU: "577018", Label: "TCL 6 Rapid Charger", Ports: 1, Rapid: true},
			{SKU: "577020", Label: "TCL 6 DUO Dual Charger", Ports: 2, Rapid: true},
		},
	},
}

// StorageSystem represents a modular tool storage system (e.g., Milwaukee PACKOUT, Ryobi LINK).
type StorageSystem struct {
	Slug        string         `json:"slug"`
	Brand       string         `json:"brand"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Color       string         `json:"color"`
	Items       []StorageItem  `json:"items"`
}

// StorageItem represents a specific storage product in a system.
type StorageItem struct {
	SKU         string `json:"sku"`
	Label       string `json:"label"`
	Category    string `json:"category"` // "toolbox", "organizer", "bag", "wall-mount", "accessory"
	Description string `json:"description,omitempty"`
}

var storageSystems = []StorageSystem{
	{
		Slug: "milwaukee-packout", Brand: "Milwaukee", Name: "PACKOUT",
		Color: "#DB0032",
		Description: "Milwaukee PACKOUT modular storage system — interlocking boxes, organizers, and wall mounts",
		Items: []StorageItem{
			{SKU: "48-22-8424", Label: "PACKOUT Tool Box", Category: "toolbox"},
			{SKU: "48-22-8425", Label: "PACKOUT Large Tool Box", Category: "toolbox"},
			{SKU: "48-22-8426", Label: "PACKOUT Rolling Tool Box", Category: "toolbox"},
			{SKU: "48-22-8422", Label: "PACKOUT Compact Tool Box", Category: "toolbox"},
			{SKU: "48-22-8431", Label: "PACKOUT Low-Profile Organizer", Category: "organizer"},
			{SKU: "48-22-8432", Label: "PACKOUT Deep Organizer", Category: "organizer"},
			{SKU: "48-22-8435", Label: "PACKOUT Compact Organizer", Category: "organizer"},
			{SKU: "48-22-8310", Label: "PACKOUT Tote", Category: "bag"},
			{SKU: "48-22-8311", Label: "PACKOUT Large Tote", Category: "bag"},
			{SKU: "48-22-8315", Label: "PACKOUT Backpack", Category: "bag"},
			{SKU: "48-22-8348", Label: "PACKOUT Wall-Mount Plate", Category: "wall-mount"},
			{SKU: "48-22-8340", Label: "PACKOUT Shop Storage Cabinet", Category: "wall-mount"},
			{SKU: "48-22-8480", Label: "PACKOUT Racking Kit", Category: "wall-mount"},
			{SKU: "48-22-8349", Label: "PACKOUT Mounting Plate", Category: "wall-mount"},
		},
	},
	{
		Slug: "ryobi-link", Brand: "Ryobi", Name: "LINK",
		Color: "#78BE20",
		Description: "Ryobi LINK modular wall storage system — wall rails, hooks, bins, and shelves",
		Items: []StorageItem{
			{SKU: "STM401", Label: "LINK Standard Wall Rail (32\")", Category: "wall-mount"},
			{SKU: "STM501", Label: "LINK Wall Cabinet", Category: "wall-mount"},
			{SKU: "STM504", Label: "LINK Tool Organizer", Category: "organizer"},
			{SKU: "STM505", Label: "LINK Wall Storage Bin", Category: "wall-mount"},
			{SKU: "STM301", Label: "LINK Power Tool Hook", Category: "accessory"},
			{SKU: "STM302", Label: "LINK Standard Hook", Category: "accessory"},
			{SKU: "STM303", Label: "LINK Compact Power Tool Hook", Category: "accessory"},
			{SKU: "STM201", Label: "LINK Tool Box (Medium)", Category: "toolbox"},
			{SKU: "STM202", Label: "LINK Tool Box (Large)", Category: "toolbox"},
		},
	},
	{
		Slug: "dewalt-toughsystem", Brand: "DeWalt", Name: "TOUGHSYSTEM 2.0",
		Color: "#FEBD17",
		Description: "DeWalt TOUGHSYSTEM 2.0 modular storage — heavy-duty interlocking boxes and organizers",
		Items: []StorageItem{
			{SKU: "DWST08165", Label: "TOUGHSYSTEM 2.0 Small Tool Box", Category: "toolbox"},
			{SKU: "DWST08300", Label: "TOUGHSYSTEM 2.0 Large Tool Box", Category: "toolbox"},
			{SKU: "DWST08400", Label: "TOUGHSYSTEM 2.0 XL Tool Box", Category: "toolbox"},
			{SKU: "DWST08202", Label: "TOUGHSYSTEM 2.0 Shallow Tool Tray", Category: "organizer"},
			{SKU: "DWST08017", Label: "TOUGHSYSTEM 2.0 Compact Organizer", Category: "organizer"},
			{SKU: "DWST08035", Label: "TOUGHSYSTEM 2.0 Deep Compact Organizer", Category: "organizer"},
			{SKU: "DWST08250", Label: "TOUGHSYSTEM 2.0 Half Width Tool Box", Category: "toolbox"},
			{SKU: "DWST08450", Label: "TOUGHSYSTEM 2.0 Rolling Tool Box", Category: "toolbox"},
		},
	},
	{
		Slug: "makita-makpac", Brand: "Makita", Name: "MAKPAC",
		Color: "#00A3A1",
		Description: "Makita MAKPAC interlocking modular case system",
		Items: []StorageItem{
			{SKU: "197210-9", Label: "MAKPAC Type 1 (Compact)", Category: "toolbox"},
			{SKU: "197211-7", Label: "MAKPAC Type 2 (Standard)", Category: "toolbox"},
			{SKU: "197212-5", Label: "MAKPAC Type 3 (Deep)", Category: "toolbox"},
			{SKU: "197213-3", Label: "MAKPAC Type 4 (Extra Deep)", Category: "toolbox"},
		},
	},
	{
		Slug: "festool-systainer", Brand: "Festool", Name: "Systainer",
		Color: "#1B1B1B",
		Description: "Festool Systainer 3 modular storage system — premium interlocking cases",
		Items: []StorageItem{
			{SKU: "204840", Label: "Systainer3 SYS3 S 76", Category: "toolbox"},
			{SKU: "204841", Label: "Systainer3 SYS3 M 112", Category: "toolbox"},
			{SKU: "204842", Label: "Systainer3 SYS3 M 137", Category: "toolbox"},
			{SKU: "204843", Label: "Systainer3 SYS3 M 187", Category: "toolbox"},
			{SKU: "204844", Label: "Systainer3 SYS3 M 237", Category: "toolbox"},
			{SKU: "204845", Label: "Systainer3 SYS3 M 337", Category: "toolbox"},
			{SKU: "204852", Label: "Systainer3 Organizer SYS3 ORG", Category: "organizer"},
		},
	},
}

// New creates a new batteries & storage plugin.
func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "batteries",
		Version:     "1.0.0",
		Description: "Power tool battery, charger, and modular storage system tracking",
		Author:      "HomeBoxNG Team",
		BuiltIn:     true,
	}
}

func (p *Plugin) Init(ctx plugins.PluginContext) error {
	p.pctx = ctx
	p.logger = ctx.Logger
	p.logger.Info().Msg("batteries & storage plugin initialized")
	return nil
}

func (p *Plugin) Start(_ context.Context) error {
	p.logger.Info().Msg("batteries & storage plugin started")
	return nil
}

func (p *Plugin) Stop(_ context.Context) error {
	p.logger.Info().Msg("batteries & storage plugin stopped")
	return nil
}

// Routes registers battery and storage management API endpoints.
func (p *Plugin) Routes(r chi.Router) {
	// ---- Battery Platforms ----

	// GET /api/plugins/batteries/platforms — List all battery platforms
	r.Get("/platforms", func(w http.ResponseWriter, r *http.Request) {
		_ = server.JSON(w, http.StatusOK, batteryPlatforms)
	})

	// GET /api/plugins/batteries/platforms/{slug} — Get a specific platform with batteries and chargers
	r.Get("/platforms/{slug}", func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		for _, bp := range batteryPlatforms {
			if bp.Slug == slug {
				_ = server.JSON(w, http.StatusOK, bp)
				return
			}
		}
		_ = server.JSON(w, http.StatusNotFound, map[string]string{"error": "platform not found"})
	})

	// GET /api/plugins/batteries/brands — List unique brands
	r.Get("/brands", func(w http.ResponseWriter, r *http.Request) {
		seen := make(map[string]bool)
		brands := []map[string]string{}
		for _, bp := range batteryPlatforms {
			if !seen[bp.Brand] {
				seen[bp.Brand] = true
				brands = append(brands, map[string]string{
					"brand": bp.Brand,
					"color": bp.Color,
				})
			}
		}
		_ = server.JSON(w, http.StatusOK, brands)
	})

	// GET /api/plugins/batteries/search?q=... — Search batteries and chargers
	r.Get("/search", func(w http.ResponseWriter, r *http.Request) {
		query := strings.ToLower(r.URL.Query().Get("q"))
		if query == "" {
			_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "query parameter 'q' is required"})
			return
		}

		type result struct {
			Type     string `json:"type"` // "battery" or "charger"
			Platform string `json:"platform"`
			Brand    string `json:"brand"`
			SKU      string `json:"sku"`
			Label    string `json:"label"`
		}

		var results []result
		for _, bp := range batteryPlatforms {
			for _, bat := range bp.Batteries {
				if strings.Contains(strings.ToLower(bat.SKU), query) ||
					strings.Contains(strings.ToLower(bat.Label), query) {
					results = append(results, result{
						Type: "battery", Platform: bp.Slug, Brand: bp.Brand,
						SKU: bat.SKU, Label: bat.Label,
					})
				}
			}
			for _, chg := range bp.Chargers {
				if strings.Contains(strings.ToLower(chg.SKU), query) ||
					strings.Contains(strings.ToLower(chg.Label), query) {
					results = append(results, result{
						Type: "charger", Platform: bp.Slug, Brand: bp.Brand,
						SKU: chg.SKU, Label: chg.Label,
					})
				}
			}
		}

		_ = server.JSON(w, http.StatusOK, map[string]any{
			"query":   query,
			"results": results,
			"total":   len(results),
		})
	})

	// ---- Storage Systems ----

	// GET /api/plugins/batteries/storage — List all storage systems
	r.Get("/storage", func(w http.ResponseWriter, r *http.Request) {
		_ = server.JSON(w, http.StatusOK, storageSystems)
	})

	// GET /api/plugins/batteries/storage/{slug} — Get a specific storage system
	r.Get("/storage/{slug}", func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		for _, ss := range storageSystems {
			if ss.Slug == slug {
				_ = server.JSON(w, http.StatusOK, ss)
				return
			}
		}
		_ = server.JSON(w, http.StatusNotFound, map[string]string{"error": "storage system not found"})
	})
}

// ConfigSchema returns plugin configuration options.
func (p *Plugin) ConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{
			Key:         "default_platform",
			Label:       "Default Platform",
			Description: "Default battery platform to show when creating new battery items",
			Type:        "select",
			Default:     "milwaukee-m18",
			Options:     []string{"milwaukee-m18", "milwaukee-m12", "ryobi-one-plus", "ryobi-40v", "dewalt-20v-max", "makita-lxt", "makita-xgt", "bosch-18v", "ridgid-18v", "festool-18v"},
			Required:    false,
		},
	}
}

// Configure applies configuration values.
func (p *Plugin) Configure(_ map[string]string) error {
	return nil
}

// RequestedPermissions declares what the batteries plugin needs.
func (p *Plugin) RequestedPermissions() []plugins.PermissionRequest {
	return []plugins.PermissionRequest{
		{Permission: plugins.PermReadItems, Reason: "Read items to identify batteries and chargers", Required: true},
		{Permission: plugins.PermAPIRoutes, Reason: "Battery platform and storage system API endpoints", Required: true},
		{Permission: plugins.PermConfig, Reason: "Default platform preference", Required: false},
	}
}

// Compile-time interface checks.
var (
	_ plugins.Plugin                = (*Plugin)(nil)
	_ plugins.RoutePlugin           = (*Plugin)(nil)
	_ plugins.ConfigPlugin          = (*Plugin)(nil)
	_ plugins.PluginWithPermissions = (*Plugin)(nil)
)

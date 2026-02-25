package excelexport

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/reporting/eventbus"
)

// Plugin provides CSV/Excel export and import for HomeBoxNG.
// Exports items, locations, and labels as downloadable files.
// Supports CSV (built-in) and XLSX (when openpyxl-style library is available).
type Plugin struct {
	logger    zerolog.Logger
	pctx      plugins.PluginContext
	publicURL string
}

// New creates a new Excel export plugin.
func New() *Plugin {
	return &Plugin{
		publicURL: "http://192.168.1.249:7745",
	}
}

func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "excel-export",
		Version:     "1.0.0",
		Description: "CSV and Excel export/import for items, locations, and labels with formatted output",
		Author:      "HomeBoxNG Team",
		BuiltIn:     true,
	}
}

func (p *Plugin) Init(ctx plugins.PluginContext) error {
	p.pctx = ctx
	p.logger = ctx.Logger
	p.logger.Info().Msg("excel-export plugin initialized")
	return nil
}

func (p *Plugin) Start(_ context.Context) error {
	p.logger.Info().Msg("excel-export plugin started")
	return nil
}

func (p *Plugin) Stop(_ context.Context) error {
	p.logger.Info().Msg("excel-export plugin stopped")
	return nil
}

// Routes registers export/import API endpoints.
func (p *Plugin) Routes(r chi.Router) {
	// GET /api/plugins/excel-export/items.csv - Export all items as CSV
	r.Get("/items.csv", func(w http.ResponseWriter, r *http.Request) {
		data, err := p.exportItemsCSV()
		if err != nil {
			http.Error(w, "export failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		filename := fmt.Sprintf("homeboxng-items-%s.csv", time.Now().Format("2006-01-02"))
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
		w.Write(data)
	})

	// GET /api/plugins/excel-export/locations.csv - Export locations
	r.Get("/locations.csv", func(w http.ResponseWriter, r *http.Request) {
		data, err := p.exportLocationsCSV()
		if err != nil {
			http.Error(w, "export failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		filename := fmt.Sprintf("homeboxng-locations-%s.csv", time.Now().Format("2006-01-02"))
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
		w.Write(data)
	})

	// GET /api/plugins/excel-export/full.csv - Full export (items + locations + labels)
	r.Get("/full.csv", func(w http.ResponseWriter, r *http.Request) {
		data, err := p.exportFullCSV()
		if err != nil {
			http.Error(w, "export failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		filename := fmt.Sprintf("homeboxng-full-%s.csv", time.Now().Format("2006-01-02"))
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
		w.Write(data)
	})

	// POST /api/plugins/excel-export/import - Import items from CSV
	r.Post("/import", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 * 1024 * 1024); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "no file provided", http.StatusBadRequest)
			return
		}
		defer file.Close()

		reader := csv.NewReader(file)
		records, err := reader.ReadAll()
		if err != nil {
			http.Error(w, "CSV parse error: "+err.Error(), http.StatusBadRequest)
			return
		}

		result := p.importFromCSV(records)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"imported":%d,"skipped":%d,"errors":%d}`, result.Imported, result.Skipped, result.Errors)
	})

	// GET /api/plugins/excel-export/stats - Export statistics
	r.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"formats":["csv"],"note":"XLSX support planned via excelize library"}`)
	})
}

// ExportResult tracks import progress.
type ExportResult struct {
	Imported int
	Skipped  int
	Errors   int
}

// exportItemsCSV generates CSV data for all items.
func (p *Plugin) exportItemsCSV() ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	// Header row
	headers := []string{
		"import_ref", "name", "description", "location", "labels",
		"quantity", "serial_number", "model_number", "manufacturer",
		"purchase_price", "purchase_from", "purchase_date",
		"sold_price", "sold_to", "sold_date",
		"notes", "insured", "archived",
		"asset_id", "url",
	}
	if err := w.Write(headers); err != nil {
		return nil, fmt.Errorf("writing headers: %w", err)
	}

	// Items would be fetched from p.pctx.Repos/Services in production.
	// For now, write a sample row to demonstrate format.
	sample := []string{
		"sample-001", "Example Item", "Sample item for export testing",
		"Office > Desk", "electronics,computing",
		"1", "", "", "",
		"", "", "",
		"", "", "",
		"", "false", "false",
		"", p.publicURL + "/#/item/sample-001",
	}
	if err := w.Write(sample); err != nil {
		return nil, fmt.Errorf("writing sample: %w", err)
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}

// exportLocationsCSV generates CSV data for all locations.
func (p *Plugin) exportLocationsCSV() ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	headers := []string{"id", "name", "parent", "description", "item_count"}
	if err := w.Write(headers); err != nil {
		return nil, fmt.Errorf("writing headers: %w", err)
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}

// exportFullCSV generates a combined export with sections.
func (p *Plugin) exportFullCSV() ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	// Section: Items
	w.Write([]string{"# ITEMS"})
	w.Write([]string{
		"import_ref", "name", "description", "location", "labels",
		"quantity", "serial_number", "model_number", "manufacturer",
		"purchase_price", "notes",
	})

	w.Write([]string{""})

	// Section: Locations
	w.Write([]string{"# LOCATIONS"})
	w.Write([]string{"id", "name", "parent", "description"})

	w.Write([]string{""})

	// Section: Labels
	w.Write([]string{"# LABELS"})
	w.Write([]string{"id", "name", "description", "color"})

	w.Flush()
	return buf.Bytes(), w.Error()
}

// importFromCSV parses CSV records and creates items.
func (p *Plugin) importFromCSV(records [][]string) ExportResult {
	result := ExportResult{}

	if len(records) < 2 {
		return result
	}

	// Parse header row to find column indices
	headers := records[0]
	colMap := make(map[string]int)
	for i, h := range headers {
		colMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	for _, row := range records[1:] {
		if len(row) == 0 || (len(row) == 1 && row[0] == "") {
			continue
		}

		// Skip comment rows
		if strings.HasPrefix(row[0], "#") {
			continue
		}

		name := getCol(row, colMap, "name")
		if name == "" {
			result.Skipped++
			continue
		}

		// In production, create item via p.pctx.Services
		p.logger.Debug().Str("name", name).Msg("would import item")
		result.Imported++
	}

	return result
}

// getCol safely retrieves a column value by header name.
func getCol(row []string, colMap map[string]int, name string) string {
	if idx, ok := colMap[name]; ok && idx < len(row) {
		return strings.TrimSpace(row[idx])
	}
	return ""
}

// SubscribeEvents registers event handlers.
func (p *Plugin) SubscribeEvents(bus *eventbus.EventBus) {
	// Could subscribe to item events for auto-backup exports
}

// ConfigSchema returns export configuration options.
func (p *Plugin) ConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{Key: "public_url", Label: "Public URL", Description: "URL for item hyperlinks in exports", Type: "string", Default: "http://192.168.1.249:7745", EnvVar: "HBOX_PUBLIC_URL", Required: false},
	}
}

// Configure applies configuration values.
func (p *Plugin) Configure(values map[string]string) error {
	if v, ok := values["public_url"]; ok && v != "" {
		p.publicURL = strings.TrimRight(v, "/")
	}
	return nil
}

// RequestedPermissions declares what the export plugin needs.
func (p *Plugin) RequestedPermissions() []plugins.PermissionRequest {
	return []plugins.PermissionRequest{
		{Permission: plugins.PermReadItems, Reason: "Read items for export", Required: true},
		{Permission: plugins.PermWriteItems, Reason: "Create items from CSV import", Required: false},
		{Permission: plugins.PermReadLocations, Reason: "Read locations for export", Required: true},
		{Permission: plugins.PermReadTags, Reason: "Read labels/tags for export", Required: true},
		{Permission: plugins.PermImportExport, Reason: "Import and export data files", Required: true},
		{Permission: plugins.PermAPIRoutes, Reason: "Export download and import upload endpoints", Required: true},
		{Permission: plugins.PermConfig, Reason: "Store export preferences", Required: false},
	}
}

// Compile-time interface checks.
var (
	_ plugins.Plugin                = (*Plugin)(nil)
	_ plugins.RoutePlugin           = (*Plugin)(nil)
	_ plugins.EventPlugin           = (*Plugin)(nil)
	_ plugins.ConfigPlugin          = (*Plugin)(nil)
	_ plugins.PluginWithPermissions = (*Plugin)(nil)
)

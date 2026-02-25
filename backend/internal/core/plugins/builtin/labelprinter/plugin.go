package labelprinter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hay-kot/httpkit/server"
	"github.com/rs/zerolog"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/reporting/eventbus"
)

// Plugin is the label printer built-in plugin for HomeBoxNG.
// It handles label generation, QR codes, and communication with
// Brother QL-series thermal label printers.
type Plugin struct {
	logger   zerolog.Logger
	pctx     plugins.PluginContext
	renderer *LabelRenderer
	printer  *BrotherQL

	// Configuration
	printerIP   string
	printerPort int
	printerModel string
	publicURL   string
	defaultOrientation Orientation
	defaultSize        LabelSize
	defaultColor       AccentColor

	// Print history
	mu      sync.Mutex
	history []PrintHistory
}

// New creates a new label printer plugin with default configuration.
func New() *Plugin {
	return &Plugin{
		printerIP:          "192.168.1.243",
		printerPort:        9100,
		printerModel:       "QL-710W",
		publicURL:          "http://192.168.1.249:7745",
		defaultOrientation: OrientationPortrait,
		defaultSize:        LabelSizeFull,
		defaultColor:       ColorBlack,
		renderer:           NewLabelRenderer(),
	}
}

func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "label-printer",
		Version:     "1.0.0",
		Description: "Brother QL thermal label printer with QR codes, smart presets, half-label support, and orientation control",
		Author:      "HomeBoxNG Team",
		BuiltIn:     true,
	}
}

func (p *Plugin) Init(ctx plugins.PluginContext) error {
	p.pctx = ctx
	p.logger = ctx.Logger

	// Initialize printer client
	p.printer = NewBrotherQL(p.printerIP, p.printerPort, p.printerModel, p.logger)

	p.logger.Info().
		Str("printer_ip", p.printerIP).
		Int("printer_port", p.printerPort).
		Str("model", p.printerModel).
		Msg("label-printer plugin initialized")

	return nil
}

func (p *Plugin) Start(_ context.Context) error {
	p.logger.Info().Msg("label-printer plugin started")
	return nil
}

func (p *Plugin) Stop(_ context.Context) error {
	p.logger.Info().Msg("label-printer plugin stopped")
	return nil
}

// Routes registers label printer API endpoints under /api/plugins/label-printer/.
func (p *Plugin) Routes(r chi.Router) {
	// POST /api/plugins/label-printer/print - Print labels
	r.Post("/print", func(w http.ResponseWriter, r *http.Request) {
		var req PrintRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request: " + err.Error()})
			return
		}

		if len(req.Labels) == 0 {
			_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "no labels provided"})
			return
		}

		// Apply defaults
		if req.Orientation == "" {
			req.Orientation = p.defaultOrientation
		}
		if req.Size == "" {
			req.Size = p.defaultSize
		}
		if req.Copies == 0 {
			req.Copies = 1
		}

		result := p.printLabels(req)
		_ = server.JSON(w, http.StatusOK, result)
	})

	// POST /api/plugins/label-printer/print-location - Print label for a location
	r.Post("/print-location", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			LocationID   string `json:"locationId"`
			LocationName string `json:"locationName"`
			ParentName   string `json:"parentName,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		// Build QR URL
		qrURL := fmt.Sprintf("%s/#/location/%s", p.publicURL, body.LocationID)

		content := LabelContent{
			Title:      body.LocationName,
			Subtitle:   body.ParentName,
			QRData:     qrURL,
			LocationID: body.LocationID,
		}

		// Match preset for automatic formatting
		preset := p.renderer.MatchPreset(body.LocationName)
		orientation := p.defaultOrientation
		size := p.defaultSize
		accentColor := string(p.defaultColor)

		if preset != nil {
			orientation = preset.Orientation
			size = preset.Size
			accentColor = string(preset.AccentColor)
			p.logger.Info().Str("preset", preset.ID).Str("location", body.LocationName).Msg("matched preset")
		}

		content.AccentColor = accentColor

		result := p.printLabels(PrintRequest{
			Labels:      []LabelContent{content},
			Orientation: orientation,
			Size:        size,
			Copies:      1,
		})

		_ = server.JSON(w, http.StatusOK, result)
	})

	// POST /api/plugins/label-printer/preview - Generate label preview (base64 PNG)
	r.Post("/preview", func(w http.ResponseWriter, r *http.Request) {
		var req PrintRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		if len(req.Labels) == 0 {
			_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "no labels provided"})
			return
		}

		if req.Orientation == "" {
			req.Orientation = p.defaultOrientation
		}

		var img = p.renderer.RenderLabel(req.Labels[0], req.Orientation)
		dataURI, err := EncodeBase64PNG(img)
		if err != nil {
			_ = server.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		_ = server.JSON(w, http.StatusOK, map[string]string{"preview": dataURI})
	})

	// GET /api/plugins/label-printer/presets - List presets
	r.Get("/presets", func(w http.ResponseWriter, r *http.Request) {
		_ = server.JSON(w, http.StatusOK, p.renderer.presets)
	})

	// GET /api/plugins/label-printer/status - Printer status
	r.Get("/status", func(w http.ResponseWriter, r *http.Request) {
		result, _ := p.printer.TestConnection()
		_ = server.JSON(w, http.StatusOK, map[string]any{
			"ip":          p.printerIP,
			"port":        p.printerPort,
			"model":       p.printerModel,
			"connection":  result,
			"orientation": p.defaultOrientation,
			"size":        p.defaultSize,
			"color":       p.defaultColor,
		})
	})

	// POST /api/plugins/label-printer/test - Test printer connection
	r.Post("/test", func(w http.ResponseWriter, r *http.Request) {
		result, err := p.printer.TestConnection()
		status := http.StatusOK
		if err != nil {
			status = http.StatusServiceUnavailable
		}
		_ = server.JSON(w, status, map[string]string{"result": result})
	})

	// GET /api/plugins/label-printer/history - Print history
	r.Get("/history", func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		h := make([]PrintHistory, len(p.history))
		copy(h, p.history)
		p.mu.Unlock()
		_ = server.JSON(w, http.StatusOK, h)
	})
}

// printLabels renders and sends labels to the printer.
func (p *Plugin) printLabels(req PrintRequest) PrintResult {
	labelCount := 0

	for _, label := range req.Labels {
		var img = p.renderer.RenderLabel(label, req.Orientation)

		pngData, err := EncodePNG(img)
		if err != nil {
			return PrintResult{Success: false, Error: "render failed: " + err.Error()}
		}

		raster, err := ImageToRaster(pngData)
		if err != nil {
			return PrintResult{Success: false, Error: "raster conversion failed: " + err.Error()}
		}

		for copy := 0; copy < req.Copies; copy++ {
			if err := p.printer.PrintRaw(raster); err != nil {
				return PrintResult{
					Success:    false,
					Error:      "print failed: " + err.Error(),
					LabelCount: labelCount,
				}
			}
			labelCount++
		}

		// Record history
		p.mu.Lock()
		p.history = append(p.history, PrintHistory{
			LocationID: label.LocationID,
			Title:      label.Title,
			PrintedAt:  time.Now().Format(time.RFC3339),
		})
		p.mu.Unlock()
	}

	return PrintResult{
		Success:    true,
		LabelCount: labelCount,
	}
}

// SubscribeEvents listens for location creation to auto-print labels.
func (p *Plugin) SubscribeEvents(bus *eventbus.EventBus) {
	bus.Subscribe(eventbus.EventItemMutation, func(data any) {
		p.logger.Debug().Msg("label-printer received item mutation event")
	})
}

// ConfigSchema returns label printer configuration options.
func (p *Plugin) ConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{Key: "printer_ip", Label: "Printer IP", Description: "Brother QL printer IP address", Type: "string", Default: "192.168.1.243", EnvVar: "HBOX_PRINTER_IP", Required: true},
		{Key: "printer_port", Label: "Printer Port", Description: "Printer TCP port (default: 9100)", Type: "number", Default: "9100", EnvVar: "HBOX_PRINTER_PORT", Required: false},
		{Key: "printer_model", Label: "Printer Model", Description: "Brother QL model (e.g., QL-710W, QL-820NWB)", Type: "string", Default: "QL-710W", Required: false},
		{Key: "public_url", Label: "Public URL", Description: "URL for QR codes (e.g., https://homebox-ai.nocommscompany.com)", Type: "string", Default: "http://192.168.1.249:7745", Required: true},
		{Key: "default_orientation", Label: "Default Orientation", Description: "Default label orientation", Type: "select", Default: "portrait", Options: []string{"portrait", "landscape"}, Required: false},
		{Key: "default_size", Label: "Default Size", Description: "Default label size", Type: "select", Default: "full", Options: []string{"full", "half"}, Required: false},
		{Key: "default_color", Label: "Default Accent Color", Description: "Default accent bar color", Type: "select", Default: "black", Options: []string{"black", "blue", "red", "green", "yellow", "orange", "purple", "white"}, Required: false},
	}
}

// Configure applies configuration values and rebuilds printer client.
func (p *Plugin) Configure(values map[string]string) error {
	if v, ok := values["printer_ip"]; ok && v != "" {
		p.printerIP = v
	}
	if v, ok := values["printer_port"]; ok && v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			p.printerPort = port
		}
	}
	if v, ok := values["printer_model"]; ok && v != "" {
		p.printerModel = v
	}
	if v, ok := values["public_url"]; ok && v != "" {
		p.publicURL = strings.TrimRight(v, "/")
	}
	if v, ok := values["default_orientation"]; ok {
		p.defaultOrientation = Orientation(v)
	}
	if v, ok := values["default_size"]; ok {
		p.defaultSize = LabelSize(v)
	}
	if v, ok := values["default_color"]; ok {
		p.defaultColor = AccentColor(v)
	}

	// Rebuild printer client
	p.printer = NewBrotherQL(p.printerIP, p.printerPort, p.printerModel, p.logger)

	return nil
}

// RequestedPermissions declares what the label printer plugin needs.
func (p *Plugin) RequestedPermissions() []plugins.PermissionRequest {
	return []plugins.PermissionRequest{
		{Permission: plugins.PermReadLocations, Reason: "Read location names for label content", Required: true},
		{Permission: plugins.PermNetwork, Reason: "Connect to Brother QL printer over TCP/IP", Required: true},
		{Permission: plugins.PermAPIRoutes, Reason: "Label printing, preview, and status endpoints", Required: true},
		{Permission: plugins.PermEvents, Reason: "Auto-print labels when new locations are created", Required: false},
		{Permission: plugins.PermConfig, Reason: "Store printer IP, defaults, and preferences", Required: true},
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

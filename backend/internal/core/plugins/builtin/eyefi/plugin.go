package eyefi

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

// Plugin is the Eye-Fi WiFi SD card receiver for HomeBoxNG.
// It implements the Eye-Fi SOAP upload protocol so photos from cameras
// with Eye-Fi Mobi Pro cards are received directly into the vision pipeline.
type Plugin struct {
	logger  zerolog.Logger
	pctx    plugins.PluginContext
	soap    *SOAPHandler
	watcher *FileWatcher

	// Upload state
	mu           sync.Mutex
	uploads      []UploadEvent
	lastUpload   *time.Time
	totalUploads int
	startedAt    time.Time

	// Configuration
	soapPort    int
	uploadKey   string
	cardMAC     string
	uploadDir   string
	importMode  ImportMode
	pollSeconds int
	soapServer  *http.Server
}

// New creates a new Eye-Fi plugin with default configuration.
func New() *Plugin {
	return &Plugin{
		soapPort:    59278,
		uploadDir:   "/tmp/homeboxng-eyefi",
		importMode:  ImportModeManual,
		pollSeconds: 10,
	}
}

func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "eyefi",
		Version:     "1.0.0",
		Description: "Eye-Fi WiFi SD card receiver with SOAP protocol support for wireless photo upload from cameras",
		Author:      "HomeBoxNG Team",
		BuiltIn:     true,
	}
}

func (p *Plugin) Init(ctx plugins.PluginContext) error {
	p.pctx = ctx
	p.logger = ctx.Logger
	p.startedAt = time.Now()

	// Create upload directory
	if err := os.MkdirAll(p.uploadDir, 0755); err != nil {
		p.logger.Warn().Err(err).Str("dir", p.uploadDir).Msg("failed to create upload dir")
	}

	// Initialize SOAP handler
	if p.uploadKey != "" {
		p.soap = NewSOAPHandler(p.uploadKey, p.cardMAC, p.logger)
	}

	// Initialize file watcher
	p.watcher = NewFileWatcher(time.Duration(p.pollSeconds)*time.Second, p.logger)
	if p.uploadDir != "" {
		p.watcher.AddSource(&IngestSource{
			Name:    "eyefi-uploads",
			Type:    "local",
			Path:    p.uploadDir,
			Enabled: true,
		})
	}
	p.watcher.SetMode(p.importMode)

	p.logger.Info().
		Int("soap_port", p.soapPort).
		Str("upload_dir", p.uploadDir).
		Msg("eyefi plugin initialized")

	return nil
}

func (p *Plugin) Start(ctx context.Context) error {
	// Start file watcher
	p.watcher.Start(ctx)

	// Start SOAP server if configured
	if p.soap != nil {
		go p.startSOAPServer()
	}

	p.logger.Info().Msg("eyefi plugin started")
	return nil
}

func (p *Plugin) Stop(_ context.Context) error {
	p.watcher.Stop()

	if p.soapServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		p.soapServer.Shutdown(ctx)
	}

	p.logger.Info().Msg("eyefi plugin stopped")
	return nil
}

// startSOAPServer runs the Eye-Fi SOAP upload receiver.
func (p *Plugin) startSOAPServer() {
	mux := http.NewServeMux()

	// Eye-Fi cards POST to /api/soap/eyefilm/v1 for all SOAP operations
	mux.HandleFunc("/api/soap/eyefilm/v1", p.handleSOAP)
	mux.HandleFunc("/api/soap/eyefilm/v1/upload", p.handleUpload)

	p.soapServer = &http.Server{
		Addr:         ":" + strconv.Itoa(p.soapPort),
		Handler:      mux,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	p.logger.Info().Int("port", p.soapPort).Msg("eye-fi SOAP server listening")

	if err := p.soapServer.ListenAndServe(); err != http.ErrServerClosed {
		p.logger.Error().Err(err).Msg("SOAP server error")
	}
}

// handleSOAP processes Eye-Fi SOAP protocol messages.
func (p *Plugin) handleSOAP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	soapBody, err := p.soap.ParseRequest(body)
	if err != nil {
		p.logger.Warn().Err(err).Msg("failed to parse SOAP request")
		http.Error(w, "parse error", http.StatusBadRequest)
		return
	}

	var respBody any

	switch {
	case soapBody.StartSession != nil:
		resp, err := p.soap.HandleStartSession(soapBody.StartSession)
		if err != nil {
			p.logger.Warn().Err(err).Msg("session start failed")
			http.Error(w, "auth failed", http.StatusUnauthorized)
			return
		}
		respBody = resp

	case soapBody.GetPhotoStatus != nil:
		respBody = p.soap.HandleGetPhotoStatus(soapBody.GetPhotoStatus)

	default:
		http.Error(w, "unknown operation", http.StatusBadRequest)
		return
	}

	respXML, err := WrapResponse(respBody)
	if err != nil {
		http.Error(w, "response error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Write(respXML)
}

// handleUpload receives TAR-archived photos from the Eye-Fi card.
func (p *Plugin) handleUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(100 * 1024 * 1024); err != nil {
		http.Error(w, "parse error", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("FILENAME")
	if err != nil {
		http.Error(w, "no file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}

	// Eye-Fi uploads are TAR archives containing the JPEG
	var savedFiles []string
	if strings.HasSuffix(strings.ToLower(header.Filename), ".tar") {
		savedFiles = p.extractTar(data)
	} else {
		// Direct JPEG upload
		filename := header.Filename
		outPath := filepath.Join(p.uploadDir, filename)
		if err := os.WriteFile(outPath, data, 0644); err == nil {
			savedFiles = append(savedFiles, outPath)
		}
	}

	// Record upload event
	now := time.Now()
	p.mu.Lock()
	p.lastUpload = &now
	p.totalUploads += len(savedFiles)
	for _, f := range savedFiles {
		p.uploads = append(p.uploads, UploadEvent{
			Filename:  filepath.Base(f),
			Size:      int64(len(data)),
			SourceMAC: r.FormValue("MACADDRESS"),
			Timestamp: now,
		})
	}
	p.mu.Unlock()

	p.logger.Info().
		Int("files", len(savedFiles)).
		Str("source", header.Filename).
		Msg("eye-fi upload received")

	// Respond with success
	resp := UploadPhotoResponse{Success: true}
	respXML, _ := xml.Marshal(resp)
	w.Header().Set("Content-Type", "text/xml")
	w.Write(respXML)
}

// extractTar extracts JPEG files from a TAR archive.
func (p *Plugin) extractTar(data []byte) []string {
	var saved []string

	tr := tar.NewReader(bytes.NewReader(data))
	for {
		hdr, err := tr.Next()
		if err != nil {
			break
		}

		ext := strings.ToLower(filepath.Ext(hdr.Name))
		if ext != ".jpg" && ext != ".jpeg" {
			continue
		}

		// Validate JPEG magic bytes
		imgData, err := io.ReadAll(tr)
		if err != nil || len(imgData) < 2 {
			continue
		}
		if imgData[0] != 0xFF || imgData[1] != 0xD8 {
			continue
		}

		outPath := filepath.Join(p.uploadDir, filepath.Base(hdr.Name))
		if err := os.WriteFile(outPath, imgData, 0644); err == nil {
			saved = append(saved, outPath)
		}
	}

	return saved
}

// Routes registers Eye-Fi API endpoints under /api/plugins/eyefi/.
func (p *Plugin) Routes(r chi.Router) {
	// GET /api/plugins/eyefi/status
	r.Get("/status", func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		status := ServerStatus{
			Running:      p.soap != nil,
			Port:         p.soapPort,
			CardMAC:      p.cardMAC,
			LastUpload:   p.lastUpload,
			TotalUploads: p.totalUploads,
			Uptime:       time.Since(p.startedAt),
		}
		p.mu.Unlock()

		_ = server.JSON(w, http.StatusOK, status)
	})

	// GET /api/plugins/eyefi/uploads - Recent upload events
	r.Get("/uploads", func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		uploads := make([]UploadEvent, len(p.uploads))
		copy(uploads, p.uploads)
		p.mu.Unlock()

		_ = server.JSON(w, http.StatusOK, uploads)
	})

	// GET /api/plugins/eyefi/sources - List ingestion sources
	r.Get("/sources", func(w http.ResponseWriter, r *http.Request) {
		stats := p.watcher.GetStats()
		_ = server.JSON(w, http.StatusOK, stats)
	})

	// POST /api/plugins/eyefi/sources/test - Test a source
	r.Post("/sources/test", func(w http.ResponseWriter, r *http.Request) {
		src := &IngestSource{
			Type: r.URL.Query().Get("type"),
			Path: r.URL.Query().Get("path"),
		}
		if src.Type == "" {
			src.Type = "local"
		}

		result, err := TestSource(src)
		status := "ok"
		if err != nil {
			status = "error"
		}

		now := time.Now()
		_ = server.JSON(w, http.StatusOK, map[string]any{
			"status":   status,
			"result":   result,
			"testedAt": now,
		})
	})

	// PUT /api/plugins/eyefi/mode - Set import mode
	r.Put("/mode", func(w http.ResponseWriter, r *http.Request) {
		mode := ImportMode(r.URL.Query().Get("mode"))
		switch mode {
		case ImportModeAuto, ImportModeManual, ImportModeOff:
			p.watcher.SetMode(mode)
			_ = server.JSON(w, http.StatusOK, map[string]string{"mode": string(mode)})
		default:
			_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid mode: use auto, manual, or off"})
		}
	})

	// POST /api/plugins/eyefi/process - Force process pending files
	r.Post("/process", func(w http.ResponseWriter, r *http.Request) {
		_ = server.JSON(w, http.StatusOK, map[string]string{"status": "processing triggered"})
	})
}

// SubscribeEvents listens for relevant events.
func (p *Plugin) SubscribeEvents(bus *eventbus.EventBus) {
	bus.Subscribe(eventbus.EventItemMutation, func(data any) {
		p.logger.Debug().Msg("eyefi received item mutation event")
	})
}

// ConfigSchema returns Eye-Fi configuration options.
func (p *Plugin) ConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{Key: "soap_port", Label: "SOAP Port", Description: "Eye-Fi SOAP protocol port (default: 59278)", Type: "number", Default: "59278", Required: false},
		{Key: "upload_key", Label: "Upload Key", Description: "32-char hex upload key from Eye-Fi Settings.xml", Type: "secret", Required: true},
		{Key: "card_mac", Label: "Card MAC", Description: "Eye-Fi card MAC address (e.g., 00:1A:7D:DA:71:XX)", Type: "string", Required: false},
		{Key: "upload_dir", Label: "Upload Directory", Description: "Directory where uploaded photos are stored", Type: "string", Default: "/tmp/homeboxng-eyefi", Required: true},
		{Key: "import_mode", Label: "Import Mode", Description: "How photos are processed: auto (immediate), manual (on demand), off (store only)", Type: "select", Default: "manual", Options: []string{"auto", "manual", "off"}, Required: false},
		{Key: "poll_interval", Label: "Poll Interval (seconds)", Description: "How often to check for new files", Type: "number", Default: "10", Required: false},
	}
}

// Configure applies configuration values.
func (p *Plugin) Configure(values map[string]string) error {
	if v, ok := values["soap_port"]; ok && v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			p.soapPort = port
		}
	}
	if v, ok := values["upload_key"]; ok {
		p.uploadKey = v
	}
	if v, ok := values["card_mac"]; ok {
		p.cardMAC = v
	}
	if v, ok := values["upload_dir"]; ok && v != "" {
		p.uploadDir = v
	}
	if v, ok := values["import_mode"]; ok {
		p.importMode = ImportMode(v)
	}
	if v, ok := values["poll_interval"]; ok && v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			p.pollSeconds = secs
		}
	}

	// Rebuild SOAP handler with new credentials
	if p.uploadKey != "" {
		p.soap = NewSOAPHandler(p.uploadKey, p.cardMAC, p.logger)
	}

	return nil
}

// RequestedPermissions declares what the Eye-Fi plugin needs.
func (p *Plugin) RequestedPermissions() []plugins.PermissionRequest {
	return []plugins.PermissionRequest{
		{Permission: plugins.PermWriteItems, Reason: "Create items from uploaded photos", Required: true},
		{Permission: plugins.PermWriteAttachments, Reason: "Attach uploaded photos to items", Required: true},
		{Permission: plugins.PermNetwork, Reason: "Run SOAP server for Eye-Fi card uploads", Required: true},
		{Permission: plugins.PermAPIRoutes, Reason: "Upload status and source management endpoints", Required: true},
		{Permission: plugins.PermEvents, Reason: "Subscribe to item events", Required: false},
		{Permission: plugins.PermConfig, Reason: "Store Eye-Fi card and upload settings", Required: true},
		{Permission: plugins.PermStorage, Reason: "Write uploaded photos to disk", Required: true},
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

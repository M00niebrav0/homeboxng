package aivision

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hay-kot/httpkit/server"
	"github.com/rs/zerolog"
	"github.com/sysadminsmedia/homebox/backend/internal/core/plugins"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/reporting/eventbus"
)

// Plugin is the AI Vision built-in plugin for HomeBoxNG.
// It provides AI-powered photo identification using a two-step pipeline:
// Step 1: Local vision model (Qwen3-VL via Ollama) identifies items from photos
// Step 2: Cloud text model (Gemini via LiteLLM) verifies and enhances results
type Plugin struct {
	logger   zerolog.Logger
	pctx     plugins.PluginContext
	pipeline *Pipeline
	grouper  *ImageGrouper

	// Configuration
	ollamaURL     string
	ollamaModel   string
	litellmURL    string
	litellmKey    string
	litellmModel  string
	autoConfidence int
}

// New creates a new AI Vision plugin with default configuration.
func New() *Plugin {
	return &Plugin{
		ollamaURL:      "http://192.168.1.193:11434",
		ollamaModel:    "qwen3-vl:8b",
		litellmURL:     "http://192.168.1.249:4000",
		litellmModel:   "gemini-flash",
		autoConfidence: 80,
		grouper:        NewImageGrouper(),
	}
}

func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "ai-vision",
		Version:     "1.0.0",
		Description: "AI-powered photo identification using local LLMs (Ollama/Qwen3-VL) and cloud verification (Gemini)",
		Author:      "HomeBoxNG Team",
		BuiltIn:     true,
	}
}

func (p *Plugin) Init(ctx plugins.PluginContext) error {
	p.pctx = ctx
	p.logger = ctx.Logger

	// Build pipeline with configured clients
	vision := NewOllamaClient(p.ollamaURL, p.ollamaModel, p.logger)
	llm := NewLiteLLMClient(p.litellmURL, p.litellmKey, p.litellmModel, p.logger)
	p.pipeline = NewPipeline(vision, llm, p.logger)

	p.logger.Info().
		Str("vision_url", p.ollamaURL).
		Str("vision_model", p.ollamaModel).
		Str("verify_url", p.litellmURL).
		Str("verify_model", p.litellmModel).
		Msg("ai-vision plugin initialized")

	return nil
}

func (p *Plugin) Start(_ context.Context) error {
	p.logger.Info().Msg("ai-vision plugin started")
	return nil
}

func (p *Plugin) Stop(_ context.Context) error {
	p.logger.Info().Msg("ai-vision plugin stopped")
	return nil
}

// Routes registers AI vision API endpoints under /api/plugins/ai-vision/.
func (p *Plugin) Routes(r chi.Router) {
	// POST /api/plugins/ai-vision/analyze - Analyze uploaded photos
	r.Post("/analyze", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(50 * 1024 * 1024); err != nil { // 50MB max
			_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart form: " + err.Error()})
			return
		}

		files := r.MultipartForm.File["images"]
		if len(files) == 0 {
			_ = server.JSON(w, http.StatusBadRequest, map[string]string{"error": "no images provided"})
			return
		}

		var imageBytes [][]byte
		for _, fh := range files {
			f, err := fh.Open()
			if err != nil {
				continue
			}
			buf := make([]byte, fh.Size)
			_, _ = f.Read(buf)
			f.Close()
			imageBytes = append(imageBytes, buf)
		}

		userContext := r.FormValue("context")
		locationsStr := r.FormValue("locations")
		labelsStr := r.FormValue("labels")

		var locations, labels []string
		if locationsStr != "" {
			_ = json.Unmarshal([]byte(locationsStr), &locations)
		}
		if labelsStr != "" {
			_ = json.Unmarshal([]byte(labelsStr), &labels)
		}

		result, err := p.pipeline.Run(r.Context(), imageBytes, locations, labels, userContext)
		if err != nil {
			_ = server.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		_ = server.JSON(w, http.StatusOK, result)
	})

	// GET /api/plugins/ai-vision/status - Check vision pipeline status
	r.Get("/status", func(w http.ResponseWriter, r *http.Request) {
		_ = server.JSON(w, http.StatusOK, map[string]any{
			"visionModel":       p.ollamaModel,
			"visionURL":         p.ollamaURL,
			"verificationModel": p.litellmModel,
			"verificationURL":   p.litellmURL,
			"autoConfidence":    p.autoConfidence,
		})
	})

	// POST /api/plugins/ai-vision/warmup - Force warmup of the vision model
	r.Post("/warmup", func(w http.ResponseWriter, r *http.Request) {
		if p.pipeline == nil || p.pipeline.vision == nil {
			_ = server.JSON(w, http.StatusServiceUnavailable, map[string]string{"error": "pipeline not initialized"})
			return
		}

		if err := p.pipeline.vision.EnsureWarm(r.Context()); err != nil {
			_ = server.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		_ = server.JSON(w, http.StatusOK, map[string]string{"status": "warm"})
	})
}

// SubscribeEvents listens for item mutations for auto-identification workflows.
func (p *Plugin) SubscribeEvents(bus *eventbus.EventBus) {
	bus.Subscribe(eventbus.EventItemMutation, func(data any) {
		p.logger.Debug().Msg("ai-vision received item mutation event")
	})
}

// ConfigSchema returns the AI vision configuration options.
func (p *Plugin) ConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{Key: "ollama_url", Label: "Ollama Vision URL", Description: "URL of Ollama instance with vision model (e.g., http://192.168.1.193:11434)", Type: "string", Default: "http://192.168.1.193:11434", Required: true},
		{Key: "ollama_model", Label: "Vision Model", Description: "Ollama model for photo identification (e.g., qwen3-vl:8b)", Type: "string", Default: "qwen3-vl:8b", Required: true},
		{Key: "litellm_url", Label: "LiteLLM URL", Description: "LiteLLM proxy URL for text verification (e.g., http://192.168.1.249:4000)", Type: "string", Default: "http://192.168.1.249:4000", Required: true},
		{Key: "litellm_key", Label: "LiteLLM API Key", Description: "API key for LiteLLM proxy", Type: "secret", Required: false},
		{Key: "litellm_model", Label: "Verification Model", Description: "Text model for step 2 verification (e.g., gemini-flash)", Type: "string", Default: "gemini-flash", Required: true},
		{Key: "auto_confidence", Label: "Auto-Add Confidence", Description: "Items above this confidence % are auto-added without asking (0-100)", Type: "number", Default: "80", Required: false},
	}
}

// Configure applies configuration values and rebuilds the pipeline.
func (p *Plugin) Configure(values map[string]string) error {
	if v, ok := values["ollama_url"]; ok && v != "" {
		p.ollamaURL = v
	}
	if v, ok := values["ollama_model"]; ok && v != "" {
		p.ollamaModel = v
	}
	if v, ok := values["litellm_url"]; ok && v != "" {
		p.litellmURL = v
	}
	if v, ok := values["litellm_key"]; ok {
		p.litellmKey = v
	}
	if v, ok := values["litellm_model"]; ok && v != "" {
		p.litellmModel = v
	}

	// Rebuild pipeline with new settings
	vision := NewOllamaClient(p.ollamaURL, p.ollamaModel, p.logger)
	llm := NewLiteLLMClient(p.litellmURL, p.litellmKey, p.litellmModel, p.logger)
	p.pipeline = NewPipeline(vision, llm, p.logger)

	return nil
}

// RequestedPermissions declares what the AI vision plugin needs.
func (p *Plugin) RequestedPermissions() []plugins.PermissionRequest {
	return []plugins.PermissionRequest{
		{Permission: plugins.PermReadItems, Reason: "Read items to avoid duplicates during identification", Required: false},
		{Permission: plugins.PermWriteItems, Reason: "Create new items from identified photos", Required: true},
		{Permission: plugins.PermReadLocations, Reason: "Suggest storage locations for identified items", Required: false},
		{Permission: plugins.PermReadTags, Reason: "Suggest labels/tags for identified items", Required: false},
		{Permission: plugins.PermWriteAttachments, Reason: "Attach original photos to created items", Required: true},
		{Permission: plugins.PermNetwork, Reason: "Connect to Ollama and LiteLLM for AI inference", Required: true},
		{Permission: plugins.PermAPIRoutes, Reason: "Photo upload and analysis API endpoints", Required: true},
		{Permission: plugins.PermEvents, Reason: "Subscribe to item events for auto-identification", Required: false},
		{Permission: plugins.PermConfig, Reason: "Store LLM connection settings", Required: true},
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

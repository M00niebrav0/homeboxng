package aivision

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

// Pipeline orchestrates the two-step vision identification process:
// Step 1: Vision model (Ollama/Qwen3-VL) identifies items from photos
// Step 2: Text model (Gemini via LiteLLM) verifies and enhances the results
type Pipeline struct {
	vision  *OllamaClient
	llm     *LiteLLMClient
	logger  zerolog.Logger
}

// NewPipeline creates a vision pipeline with the given clients.
func NewPipeline(vision *OllamaClient, llm *LiteLLMClient, logger zerolog.Logger) *Pipeline {
	return &Pipeline{
		vision: vision,
		llm:    llm,
		logger: logger.With().Str("component", "vision-pipeline").Logger(),
	}
}

// Run processes a list of images through the full pipeline.
// It processes images one at a time (8GB VRAM limit) with context carry-forward.
func (p *Pipeline) Run(ctx context.Context, images [][]byte, locations, labels []string, userContext string) (*PipelineResult, error) {
	result := &PipelineResult{
		VisionModel:       p.vision.model,
		VerificationModel: p.llm.model,
		ImageCount:        len(images),
		ProcessedAt:       time.Now(),
	}

	if len(images) == 0 {
		return result, nil
	}

	// Step 0: Ensure vision model is warm
	if err := p.vision.EnsureWarm(ctx); err != nil {
		p.logger.Warn().Err(err).Msg("warmup failed, proceeding anyway")
	}

	// Step 1: Analyze each image with the vision model
	var allRawItems []IdentifiedItem
	var contextItems []string // Names of previously identified items

	for i, imgBytes := range images {
		p.logger.Info().Int("image", i+1).Int("total", len(images)).Msg("analyzing image")

		analysis := p.analyzeImage(ctx, imgBytes, i, len(images), contextItems, userContext)
		if !analysis.Success {
			p.logger.Warn().Int("image", i+1).Str("error", analysis.Error).Msg("image analysis failed")
			continue
		}

		allRawItems = append(allRawItems, analysis.RawItems...)

		// Build context for next image
		for _, item := range analysis.RawItems {
			contextItems = append(contextItems, item.Name)
		}

		// Small delay between images to avoid overloading VRAM
		if i < len(images)-1 {
			time.Sleep(2 * time.Second)
		}
	}

	result.RawItems = allRawItems

	if len(allRawItems) == 0 {
		result.Error = "no items identified from any image"
		return result, nil
	}

	// Step 2: Verify with text model (Gemini)
	verified, err := p.verify(ctx, allRawItems, locations, labels, userContext)
	if err != nil {
		p.logger.Warn().Err(err).Msg("verification failed, using raw items")
		result.VerifiedItems = allRawItems
		return result, nil
	}

	result.VerifiedItems = verified
	p.logger.Info().
		Int("raw", len(allRawItems)).
		Int("verified", len(verified)).
		Msg("pipeline complete")

	return result, nil
}

// analyzeImage processes a single image with retry logic.
func (p *Pipeline) analyzeImage(ctx context.Context, imageBytes []byte, index, total int, contextItems []string, userContext string) ImageAnalysis {
	prompt := BuildStep1Prompt(index, total, contextItems, userContext)

	var lastErr string

	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			p.logger.Info().Int("attempt", attempt+1).Msg("retrying image analysis")
			time.Sleep(3 * time.Second)
		}

		text, err := p.vision.AnalyzeImage(ctx, imageBytes, prompt, 180*time.Second)
		if err != nil {
			lastErr = err.Error()
			continue
		}

		if text == "" {
			lastErr = "empty response from vision model"
			continue
		}

		items := ExtractJSONFromText(text)
		if items == nil {
			items = []IdentifiedItem{} // Ensure non-nil
		}

		return ImageAnalysis{
			ImageIndex:     index,
			RawItems:       items,
			Success:        true,
			ResponseLength: len(text),
		}
	}

	return ImageAnalysis{
		ImageIndex: index,
		Success:    false,
		Error:      lastErr,
	}
}

// verify sends raw items to the text model for verification and enhancement.
func (p *Pipeline) verify(ctx context.Context, rawItems []IdentifiedItem, locations, labels []string, userContext string) ([]IdentifiedItem, error) {
	rawJSON, err := json.MarshalIndent(rawItems, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling raw items: %w", err)
	}

	prompt := BuildStep2Prompt(string(rawJSON), locations, labels, userContext)

	text, err := p.llm.Complete(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("verification call failed: %w", err)
	}

	verified := ExtractJSONFromText(text)
	if verified == nil || len(verified) == 0 {
		return nil, fmt.Errorf("verification returned no items")
	}

	return verified, nil
}

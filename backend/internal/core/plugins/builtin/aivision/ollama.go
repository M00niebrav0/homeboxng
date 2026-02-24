package aivision

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// OllamaClient handles direct communication with an Ollama instance for vision.
// LiteLLM cannot forward images to Ollama, so vision calls go direct.
type OllamaClient struct {
	baseURL string
	model   string
	logger  zerolog.Logger

	// Warmup state - first call after model load returns empty
	warmupMu       sync.Mutex
	warmupVerified time.Time
	warmupTTL      time.Duration
}

// NewOllamaClient creates a client for Ollama vision calls.
func NewOllamaClient(baseURL, model string, logger zerolog.Logger) *OllamaClient {
	return &OllamaClient{
		baseURL:   baseURL,
		model:     model,
		logger:    logger.With().Str("client", "ollama-vision").Logger(),
		warmupTTL: 5 * time.Minute,
	}
}

// ollamaRequest is the Ollama /api/chat request format.
type ollamaRequest struct {
	Model    string           `json:"model"`
	Messages []ollamaMessage  `json:"messages"`
	Stream   bool             `json:"stream"`
	Options  *ollamaOptions   `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"images,omitempty"` // base64-encoded
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
}

// ollamaResponse is the Ollama /api/chat response format.
type ollamaResponse struct {
	Message struct {
		Content  string `json:"content"`
		Thinking string `json:"thinking"`
	} `json:"message"`
}

// EnsureWarm verifies the vision model is loaded and ready.
// Qwen3-VL returns empty on the first call after model load.
func (c *OllamaClient) EnsureWarm(ctx context.Context) error {
	c.warmupMu.Lock()
	defer c.warmupMu.Unlock()

	if time.Since(c.warmupVerified) < c.warmupTTL {
		return nil // Still warm
	}

	c.logger.Info().Msg("warming up vision model")

	resp, err := c.chatRaw(ctx, "/no_think Say OK", nil, 30*time.Second)
	if err != nil {
		return fmt.Errorf("warmup failed: %w", err)
	}

	if resp == "" {
		// Retry once after delay
		time.Sleep(5 * time.Second)
		resp, err = c.chatRaw(ctx, "/no_think Say OK", nil, 30*time.Second)
		if err != nil {
			return fmt.Errorf("warmup retry failed: %w", err)
		}
	}

	if resp != "" {
		c.warmupVerified = time.Now()
		c.logger.Info().Msg("vision model is warm")
	} else {
		c.logger.Warn().Msg("warmup returned empty response")
	}

	return nil
}

// AnalyzeImage sends a single image to the vision model with a prompt.
// Returns the raw text response from the model.
func (c *OllamaClient) AnalyzeImage(ctx context.Context, imageBytes []byte, prompt string, timeout time.Duration) (string, error) {
	b64 := base64.StdEncoding.EncodeToString(imageBytes)

	// Prepend /no_think to disable Qwen3's thinking mode
	fullPrompt := "/no_think\n\n" + prompt

	return c.chatRaw(ctx, fullPrompt, []string{b64}, timeout)
}

// chatRaw sends a chat request to Ollama and returns the content string.
func (c *OllamaClient) chatRaw(ctx context.Context, content string, images []string, timeout time.Duration) (string, error) {
	msg := ollamaMessage{
		Role:    "user",
		Content: content,
		Images:  images,
	}

	reqBody := ollamaRequest{
		Model:    c.model,
		Messages: []ollamaMessage{msg},
		Stream:   false,
		Options:  &ollamaOptions{Temperature: 0.1},
	}

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	url := c.baseURL + "/api/chat"

	// Fresh HTTP client per call to avoid stale connection pools after timeouts
	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyJSON))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB limit
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	// Return content, falling back to thinking field
	result := ollamaResp.Message.Content
	if result == "" {
		result = ollamaResp.Message.Thinking
	}

	return result, nil
}

package aivision

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// LiteLLMClient handles text-only LLM calls via the LiteLLM proxy.
// Used for Step 2 verification (Gemini Flash, etc.) where no images are needed.
type LiteLLMClient struct {
	baseURL string
	apiKey  string
	model   string
	logger  zerolog.Logger
	client  *http.Client
}

// NewLiteLLMClient creates a client for LiteLLM text completions.
func NewLiteLLMClient(baseURL, apiKey, model string, logger zerolog.Logger) *LiteLLMClient {
	return &LiteLLMClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		logger:  logger.With().Str("client", "litellm").Logger(),
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

// openAIRequest is the OpenAI-compatible request format that LiteLLM accepts.
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAIResponse is the OpenAI-compatible response format.
type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Complete sends a text-only completion request via LiteLLM.
func (c *LiteLLMClient) Complete(ctx context.Context, prompt string) (string, error) {
	reqBody := openAIRequest{
		Model: c.model,
		Messages: []openAIMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1,
		MaxTokens:   4096,
	}

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	url := c.baseURL + "/v1/chat/completions"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyJSON))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("litellm request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("litellm returned %d: %s", resp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("litellm returned no choices")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

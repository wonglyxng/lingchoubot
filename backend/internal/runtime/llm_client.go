package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LLMClientConfig holds the configuration for an OpenAI-compatible LLM API.
type LLMClientConfig struct {
	BaseURL     string
	APIKey      string
	Model       string
	Temperature float64
	MaxTokens   int
}

// LLMClient calls an OpenAI-compatible chat completions endpoint.
type LLMClient struct {
	cfg    LLMClientConfig
	client *http.Client
}

func NewLLMClient(cfg LLMClientConfig) *LLMClient {
	if cfg.Temperature == 0 {
		cfg.Temperature = 0.3
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 4096
	}
	return &LLMClient{
		cfg:    cfg,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	Temperature    float64         `json:"temperature"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage *chatUsage `json:"usage,omitempty"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type chatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LLMCallMeta holds metadata about a single LLM API call for audit purposes.
type LLMCallMeta struct {
	Model            string        `json:"model"`
	DurationMs       int64         `json:"duration_ms"`
	PromptTokens     int           `json:"prompt_tokens"`
	CompletionTokens int           `json:"completion_tokens"`
	TotalTokens      int           `json:"total_tokens"`
	StatusCode       int           `json:"status_code"`
	Error            string        `json:"error,omitempty"`
	PromptVersion    string        `json:"prompt_version,omitempty"`
}

// ChatJSON sends a system+user prompt to the LLM and returns the raw content string.
// response_format is set to json_object to force valid JSON output.
func (c *LLMClient) ChatJSON(ctx context.Context, system, user string) (string, error) {
	content, _, err := c.ChatJSONWithMeta(ctx, system, user)
	return content, err
}

// ChatJSONWithMeta sends a system+user prompt and returns both content and call metadata.
func (c *LLMClient) ChatJSONWithMeta(ctx context.Context, system, user string) (string, *LLMCallMeta, error) {
	meta := &LLMCallMeta{Model: c.cfg.Model}
	start := time.Now()

	reqBody := chatRequest{
		Model: c.cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature:    c.cfg.Temperature,
		MaxTokens:      c.cfg.MaxTokens,
		ResponseFormat: &responseFormat{Type: "json_object"},
	}

	body, _ := json.Marshal(reqBody)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		meta.DurationMs = time.Since(start).Milliseconds()
		meta.Error = err.Error()
		return "", meta, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		meta.DurationMs = time.Since(start).Milliseconds()
		meta.Error = err.Error()
		return "", meta, fmt.Errorf("http call: %w", err)
	}
	defer resp.Body.Close()
	meta.StatusCode = resp.StatusCode

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		meta.DurationMs = time.Since(start).Milliseconds()
		meta.Error = err.Error()
		return "", meta, fmt.Errorf("read response: %w", err)
	}
	meta.DurationMs = time.Since(start).Milliseconds()

	if resp.StatusCode != http.StatusOK {
		meta.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return "", meta, fmt.Errorf("LLM API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		meta.Error = "response parse error"
		return "", meta, fmt.Errorf("parse response: %w", err)
	}
	if chatResp.Error != nil {
		meta.Error = chatResp.Error.Message
		return "", meta, fmt.Errorf("LLM error: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		meta.Error = "no choices"
		return "", meta, fmt.Errorf("LLM returned no choices")
	}

	// Extract token usage if provided by the API
	if chatResp.Usage != nil {
		meta.PromptTokens = chatResp.Usage.PromptTokens
		meta.CompletionTokens = chatResp.Usage.CompletionTokens
		meta.TotalTokens = chatResp.Usage.TotalTokens
	}

	return chatResp.Choices[0].Message.Content, meta, nil
}

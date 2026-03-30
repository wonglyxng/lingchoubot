package runtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLLMClient_ChatJSON_Success(t *testing.T) {
	expected := `{"status":"success","summary":"test"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected /chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer test-key, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content-type")
		}

		// Verify request body structure
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "test-model" {
			t.Errorf("expected model test-model, got %s", req.Model)
		}
		if len(req.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(req.Messages))
		}
		if req.Messages[0].Role != "system" || req.Messages[0].Content != "sys" {
			t.Errorf("unexpected system message: %+v", req.Messages[0])
		}
		if req.Messages[1].Role != "user" || req.Messages[1].Content != "usr" {
			t.Errorf("unexpected user message: %+v", req.Messages[1])
		}
		if req.ResponseFormat == nil || req.ResponseFormat.Type != "json_object" {
			t.Errorf("expected json_object response format")
		}

		resp := chatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: expected}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewLLMClient(LLMClientConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "test-model",
	})

	result, err := client.ChatJSON(context.Background(), "sys", "usr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestLLMClient_ChatJSON_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"rate limit exceeded"}}`))
	}))
	defer server.Close()

	client := NewLLMClient(LLMClientConfig{
		BaseURL: server.URL,
		APIKey:  "key",
		Model:   "model",
	})

	_, err := client.ChatJSON(context.Background(), "sys", "usr")
	if err == nil {
		t.Fatal("expected error for HTTP 429")
	}
	if got := err.Error(); !contains(got, "429") {
		t.Errorf("error should mention 429: %s", got)
	}
}

func TestLLMClient_ChatJSON_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatResponse{}
		resp.Error = &struct {
			Message string `json:"message"`
		}{Message: "invalid model"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewLLMClient(LLMClientConfig{
		BaseURL: server.URL,
		APIKey:  "key",
		Model:   "model",
	})

	_, err := client.ChatJSON(context.Background(), "sys", "usr")
	if err == nil {
		t.Fatal("expected error for API error response")
	}
	if got := err.Error(); !contains(got, "invalid model") {
		t.Errorf("error should mention 'invalid model': %s", got)
	}
}

func TestLLMClient_ChatJSON_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatResponse{Choices: nil}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewLLMClient(LLMClientConfig{
		BaseURL: server.URL,
		APIKey:  "key",
		Model:   "model",
	})

	_, err := client.ChatJSON(context.Background(), "sys", "usr")
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
	if got := err.Error(); !contains(got, "no choices") {
		t.Errorf("error should mention 'no choices': %s", got)
	}
}

func TestLLMClient_ChatJSON_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	client := NewLLMClient(LLMClientConfig{
		BaseURL: server.URL,
		APIKey:  "key",
		Model:   "model",
	})

	_, err := client.ChatJSON(context.Background(), "sys", "usr")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if got := err.Error(); !contains(got, "parse response") {
		t.Errorf("error should mention 'parse response': %s", got)
	}
}

func TestLLMClient_ChatJSON_ServerDown(t *testing.T) {
	client := NewLLMClient(LLMClientConfig{
		BaseURL: "http://127.0.0.1:1", // nothing listening
		APIKey:  "key",
		Model:   "model",
	})

	_, err := client.ChatJSON(context.Background(), "sys", "usr")
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
	if got := err.Error(); !contains(got, "http call") {
		t.Errorf("error should mention 'http call': %s", got)
	}
}

func TestLLMClient_DefaultConfig(t *testing.T) {
	client := NewLLMClient(LLMClientConfig{
		BaseURL: "http://example.com",
		APIKey:  "key",
		Model:   "model",
	})
	if client.cfg.Temperature != 0.3 {
		t.Errorf("expected default temperature 0.3, got %f", client.cfg.Temperature)
	}
	if client.cfg.MaxTokens != 4096 {
		t.Errorf("expected default max_tokens 4096, got %d", client.cfg.MaxTokens)
	}
}

func TestLLMClient_CustomConfig(t *testing.T) {
	client := NewLLMClient(LLMClientConfig{
		BaseURL:     "http://example.com",
		APIKey:      "key",
		Model:       "model",
		Temperature: 0.7,
		MaxTokens:   2048,
	})
	if client.cfg.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %f", client.cfg.Temperature)
	}
	if client.cfg.MaxTokens != 2048 {
		t.Errorf("expected max_tokens 2048, got %d", client.cfg.MaxTokens)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

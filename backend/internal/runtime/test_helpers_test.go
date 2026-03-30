package runtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

// newTestLLMClient creates an LLMClient backed by a local httptest server.
// If forceErr is non-nil, the server will return an error.
// Otherwise it returns response as the LLM content.
func newTestLLMClient(response string, forceErr error) *LLMClient {
	if forceErr != nil {
		// Return a client pointing to a broken endpoint
		return &LLMClient{
			cfg:    LLMClientConfig{BaseURL: "http://127.0.0.1:1", Model: "test"},
			client: &http.Client{},
		}
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: response}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))

	return &LLMClient{
		cfg: LLMClientConfig{
			BaseURL: server.URL,
			APIKey:  "test-key",
			Model:   "test-model",
		},
		client: server.Client(),
	}
}

// mockLLMClient wraps an LLMClient but allows injecting errors for testing.
type mockLLMClient struct {
	response string
	err      error
}

func (m *mockLLMClient) ChatJSON(ctx context.Context, system, user string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

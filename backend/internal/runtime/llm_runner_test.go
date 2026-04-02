package runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"testing"
)

func TestLLMRunner_Execute_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Prepare a mock LLM response
	mockOutput := AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: "项目分解完成",
		Phases: []PhaseAction{
			{Name: "设计阶段", Description: "系统设计", SortOrder: 1},
		},
		Tasks: []TaskAction{
			{PhaseName: "设计阶段", Title: "API 设计", Description: "设计 REST API", Priority: 3},
		},
	}
	mockJSON, _ := json.Marshal(mockOutput)

	runner := &LLMAgentRunner{
		client: newTestLLMClient(string(mockJSON), nil),
		role:   "pm",
		spec:   "",
		logger: logger,
	}

	input := &AgentTaskInput{
		RunID:       "run-1",
		AgentID:     "agent-1",
		AgentRole:   "pm",
		Instruction: "分解项目",
		Project:     &ProjectCtx{ID: "proj-1", Name: "测试项目", Description: "测试"},
	}

	output, err := runner.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Status != OutputStatusSuccess {
		t.Errorf("expected success status, got %s", output.Status)
	}
	if output.Summary != "项目分解完成" {
		t.Errorf("unexpected summary: %s", output.Summary)
	}
	if len(output.Phases) != 1 {
		t.Errorf("expected 1 phase, got %d", len(output.Phases))
	}
	if len(output.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(output.Tasks))
	}
}

func TestLLMRunner_Execute_LLMError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	runner := &LLMAgentRunner{
		client: newTestLLMClient("", fmt.Errorf("connection refused")),
		role:   "pm",
		spec:   "",
		logger: logger,
	}

	input := &AgentTaskInput{
		RunID:       "run-1",
		AgentID:     "agent-1",
		AgentRole:   "pm",
		Instruction: "分解项目",
	}

	output, err := runner.Execute(input)
	if err != nil {
		t.Fatalf("Execute should not return error, got: %v", err)
	}
	if output.Status != OutputStatusFailed {
		t.Errorf("expected failed status, got %s", output.Status)
	}
	if output.Error == "" {
		t.Error("expected error message in output")
	}
}

func TestLLMRunner_Execute_InvalidJSON(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	runner := &LLMAgentRunner{
		client: newTestLLMClient("not valid json {[", nil),
		role:   "worker",
		spec:   "backend",
		logger: logger,
	}

	input := &AgentTaskInput{
		RunID:       "run-1",
		AgentID:     "agent-1",
		AgentRole:   "worker",
		Instruction: "执行任务",
	}

	output, err := runner.Execute(input)
	if err != nil {
		t.Fatalf("Execute should not return error, got: %v", err)
	}
	if output.Status != OutputStatusFailed {
		t.Errorf("expected failed status, got %s", output.Status)
	}
	if !containsHelper(output.Error, "parse LLM response") {
		t.Errorf("error should mention parse failure: %s", output.Error)
	}
}

func TestLLMRunner_Execute_EmptyStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// LLM returns JSON without status field and without the required PM payload.
	mockOutput := `{"summary":"done","phases":[]}`
	runner := &LLMAgentRunner{
		client: newTestLLMClient(mockOutput, nil),
		role:   "pm",
		spec:   "",
		logger: logger,
	}

	input := &AgentTaskInput{
		RunID:       "run-1",
		AgentID:     "agent-1",
		AgentRole:   "pm",
		Instruction: "分解项目",
	}

	output, err := runner.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Status != OutputStatusFailed {
		t.Errorf("expected strict validation failure, got %s", output.Status)
	}
	if !containsHelper(output.Error, "output validation") {
		t.Errorf("expected validation error, got %s", output.Error)
	}
}

func TestLLMRunner_RoleAndSpec(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewLLMRunner(nil, "worker", "backend", logger)
	if runner.Role() != "worker" {
		t.Errorf("expected role 'worker', got %q", runner.Role())
	}
	if runner.Specialization() != "backend" {
		t.Errorf("expected spec 'backend', got %q", runner.Specialization())
	}
}

func TestRegisterLLMRunners(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	reg := NewRegistry()

	defaultClient := NewLLMClient(LLMClientConfig{
		BaseURL: "http://default.example.com",
		APIKey:  "default-key",
		Model:   "default-model",
	})

	pmClient := NewLLMClient(LLMClientConfig{
		BaseURL: "http://pm.example.com",
		APIKey:  "pm-key",
		Model:   "pm-model",
	})

	roleClients := map[string]*LLMClient{
		"pm": pmClient,
	}

	RegisterLLMRunners(reg, defaultClient, roleClients, nil, logger)

	// Verify all roles are registered
	roles := []string{"pm", "supervisor", "worker", "reviewer"}
	for _, role := range roles {
		runner, err := reg.Get(role)
		if err != nil {
			t.Errorf("expected runner for %s, got error: %v", role, err)
			continue
		}
		if runner.Role() != role {
			t.Errorf("expected role %q, got %q", role, runner.Role())
		}
	}

	// Verify specialized workers
	specs := []string{"backend", "frontend", "qa"}
	for _, spec := range specs {
		runner, err := reg.GetForSpec("worker", spec)
		if err != nil {
			t.Errorf("expected runner for worker:%s, got error: %v", spec, err)
		}
		if sr, ok := runner.(SpecializedRunner); ok {
			if sr.Specialization() != spec {
				t.Errorf("expected spec %q, got %q", spec, sr.Specialization())
			}
		}
	}

	// Verify PM uses custom client (check via runner's internal client)
	pmRunner, _ := reg.Get("pm")
	llmRunner, ok := pmRunner.(*LLMAgentRunner)
	if !ok {
		t.Fatal("pm runner should be *LLMAgentRunner")
	}
	if llmRunner.client != pmClient {
		t.Error("pm runner should use pm-specific client")
	}

	// Verify supervisor uses default client
	supRunner, _ := reg.Get("supervisor")
	llmSup, ok := supRunner.(*LLMAgentRunner)
	if !ok {
		t.Fatal("supervisor runner should be *LLMAgentRunner")
	}
	if llmSup.client != defaultClient {
		t.Error("supervisor runner should use default client")
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	tests := []struct {
		role string
		spec string
		want string // substring expected in the prompt
	}{
		{"pm", "", "项目经理"},
		{"supervisor", "", "主管"},
		{"worker", "backend", "后端开发"},
		{"worker", "frontend", "前端开发"},
		{"worker", "qa", "质量保障"},
		{"worker", "general", "通用"},
		{"reviewer", "", "评审"},
		{"unknown", "", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.role+"/"+tt.spec, func(t *testing.T) {
			prompt := buildSystemPrompt(tt.role, tt.spec)
			if !containsHelper(prompt, tt.want) {
				t.Errorf("prompt for %s/%s should contain %q, got:\n%s", tt.role, tt.spec, tt.want, prompt[:min(200, len(prompt))])
			}
		})
	}
}

func TestBuildUserPrompt(t *testing.T) {
	input := &AgentTaskInput{
		RunID:       "run-1",
		AgentID:     "agent-1",
		AgentRole:   "pm",
		Instruction: "分解项目",
		Project:     &ProjectCtx{ID: "proj-1", Name: "测试", Description: "测试项目"},
	}

	prompt, err := buildUserPrompt(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(prompt), &parsed); err != nil {
		t.Fatalf("user prompt should be valid JSON: %v", err)
	}

	if parsed["run_id"] != "run-1" {
		t.Errorf("expected run_id 'run-1', got %v", parsed["run_id"])
	}
	if parsed["instruction"] != "分解项目" {
		t.Errorf("expected instruction '分解项目', got %v", parsed["instruction"])
	}
}

func TestTruncateStr(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
	}
	for _, tt := range tests {
		got := truncateStr(tt.input, tt.max)
		if got != tt.want {
			t.Errorf("truncateStr(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestLLMRunner_Fallback_OnLLMError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	runner := &LLMAgentRunner{
		client:   newTestLLMClient("", fmt.Errorf("connection refused")),
		role:     "pm",
		spec:     "",
		logger:   logger,
		fallback: &MockPMAgent{},
	}

	input := &AgentTaskInput{
		RunID:       "run-1",
		AgentID:     "agent-1",
		AgentRole:   "pm",
		Instruction: "分解项目",
		Project:     &ProjectCtx{ID: "p1", Name: "Test", Description: "test"},
	}

	output, err := runner.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Status != OutputStatusFailed {
		t.Errorf("expected failed status, got %s", output.Status)
	}
}

func TestLLMRunner_Fallback_OnParseError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	runner := &LLMAgentRunner{
		client:   newTestLLMClient("not json", nil),
		role:     "worker",
		spec:     "backend",
		logger:   logger,
		fallback: &MockBackendWorkerAgent{},
	}

	input := &AgentTaskInput{
		RunID:       "run-1",
		AgentID:     "agent-1",
		AgentRole:   "worker",
		Instruction: "执行",
		Task:        &TaskCtx{ID: "t1", Title: "task1"},
	}

	output, err := runner.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Status != OutputStatusFailed {
		t.Errorf("expected failed status, got %s", output.Status)
	}
}

func TestLLMRunner_Fallback_OnValidationError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// PM output missing phases and tasks → validation fails
	mockOutput := `{"status":"success","summary":"done"}`
	runner := &LLMAgentRunner{
		client:   newTestLLMClient(mockOutput, nil),
		role:     "pm",
		spec:     "",
		logger:   logger,
		fallback: &MockPMAgent{},
	}

	input := &AgentTaskInput{
		RunID:       "run-1",
		AgentID:     "agent-1",
		AgentRole:   "pm",
		Instruction: "分解项目",
		Project:     &ProjectCtx{ID: "p1", Name: "Test", Description: "test"},
	}

	output, err := runner.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Status != OutputStatusFailed {
		t.Errorf("expected failed status, got %s", output.Status)
	}
}

func TestLLMRunner_WithFallback(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewLLMRunner(nil, "pm", "", logger)
	if runner.fallback != nil {
		t.Error("initially fallback should be nil")
	}
	fb := &MockPMAgent{}
	runner.WithFallback(fb)
	if runner.fallback != fb {
		t.Error("WithFallback should set the fallback runner")
	}
}

func TestRegisterLLMRunnersWithFallback(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	reg := NewRegistry()

	defaultClient := NewLLMClient(LLMClientConfig{
		BaseURL: "http://default.example.com",
		APIKey:  "default-key",
		Model:   "default-model",
	})

	RegisterLLMRunnersWithFallback(reg, defaultClient, nil, nil, nil, logger, true)

	// Wrapper should still register all roles, but fallback is no longer active.
	for _, role := range []string{"pm", "supervisor", "worker", "reviewer"} {
		runner, err := reg.Get(role)
		if err != nil {
			t.Errorf("expected runner for %s: %v", role, err)
			continue
		}
		llmR, ok := runner.(*LLMAgentRunner)
		if !ok {
			t.Errorf("expected *LLMAgentRunner for %s", role)
			continue
		}
		if llmR.fallback != nil {
			t.Errorf("fallback should remain inactive for %s", role)
		}
	}
}

func TestMetaHelpers(t *testing.T) {
	// nil meta
	if metaDurationMs(nil) != 0 {
		t.Error("metaDurationMs(nil) should be 0")
	}
	if metaModel(nil) != "" {
		t.Error("metaModel(nil) should be empty")
	}
	if metaTotalTokens(nil) != 0 {
		t.Error("metaTotalTokens(nil) should be 0")
	}

	// non-nil meta
	m := &LLMCallMeta{Model: "gpt-4o", DurationMs: 1234, TotalTokens: 500}
	if metaDurationMs(m) != 1234 {
		t.Errorf("expected 1234, got %d", metaDurationMs(m))
	}
	if metaModel(m) != "gpt-4o" {
		t.Errorf("expected gpt-4o, got %s", metaModel(m))
	}
	if metaTotalTokens(m) != 500 {
		t.Errorf("expected 500, got %d", metaTotalTokens(m))
	}
}

func TestLLMRunnerClientForInputUsesAgentOverride(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewLLMRunner(NewLLMClient(LLMClientConfig{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "openai-key",
		Model:   "gpt-4.1-mini",
	}), "worker", "backend", logger).WithProviderConfigs(map[string]LLMClientConfig{
		"deepseek": {
			BaseURL: "https://api.deepseek.com/v1",
			APIKey:  "deepseek-key",
		},
	})

	client, err := runner.clientForInput(&AgentTaskInput{
		AgentLLM: &AgentLLMConfig{
			Provider: "deepseek",
			Model:    "deepseek-chat",
		},
	})
	if err != nil {
		t.Fatalf("clientForInput returned error: %v", err)
	}
	if client.cfg.BaseURL != "https://api.deepseek.com/v1" {
		t.Fatalf("base URL = %s, want deepseek base URL", client.cfg.BaseURL)
	}
	if client.cfg.APIKey != "deepseek-key" {
		t.Fatalf("api key = %s, want deepseek-key", client.cfg.APIKey)
	}
	if client.cfg.Model != "deepseek-chat" {
		t.Fatalf("model = %s, want deepseek-chat", client.cfg.Model)
	}
}

func TestLLMRunnerClientForInputRejectsUnknownProvider(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runner := NewLLMRunner(NewLLMClient(LLMClientConfig{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "openai-key",
		Model:   "gpt-4.1-mini",
	}), "worker", "backend", logger)

	_, err := runner.clientForInput(&AgentTaskInput{
		AgentLLM: &AgentLLMConfig{
			Provider: "unknown-provider",
			Model:    "some-model",
		},
	})
	if err == nil {
		t.Fatal("expected unknown provider error")
	}
}

func TestLLMRunnerClientForInputDynamicLookup(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Create runner with a dynamic lookup that returns custom provider config
	runner := NewLLMRunner(NewLLMClient(LLMClientConfig{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "openai-key",
		Model:   "gpt-4.1-mini",
	}), "worker", "backend", logger).WithProviderLookup(func(key string) (string, string, bool) {
		if key == "custom-llm" {
			return "https://custom.example.com/v1", "custom-key", true
		}
		return "", "", false
	})

	// Test: dynamic lookup resolves known provider
	client, err := runner.clientForInput(&AgentTaskInput{
		AgentLLM: &AgentLLMConfig{
			Provider: "custom-llm",
			Model:    "custom-model",
		},
	})
	if err != nil {
		t.Fatalf("clientForInput returned error: %v", err)
	}
	if client.cfg.BaseURL != "https://custom.example.com/v1" {
		t.Errorf("base URL = %s, want custom base URL", client.cfg.BaseURL)
	}
	if client.cfg.APIKey != "custom-key" {
		t.Errorf("api key = %s, want custom-key", client.cfg.APIKey)
	}
	if client.cfg.Model != "custom-model" {
		t.Errorf("model = %s, want custom-model", client.cfg.Model)
	}

	// Test: dynamic lookup returns false → falls back to static map error
	_, err = runner.clientForInput(&AgentTaskInput{
		AgentLLM: &AgentLLMConfig{
			Provider: "missing-provider",
			Model:    "any-model",
		},
	})
	if err == nil {
		t.Fatal("expected error for provider not in lookup or static map")
	}
}

func TestLLMRunnerClientForInputDynamicLookupPriority(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Runner has BOTH static map and dynamic lookup for the same provider
	runner := NewLLMRunner(NewLLMClient(LLMClientConfig{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "openai-key",
		Model:   "gpt-4.1-mini",
	}), "worker", "backend", logger).
		WithProviderConfigs(map[string]LLMClientConfig{
			"dual": {BaseURL: "https://static.example.com/v1", APIKey: "static-key"},
		}).
		WithProviderLookup(func(key string) (string, string, bool) {
			if key == "dual" {
				return "https://dynamic.example.com/v1", "dynamic-key", true
			}
			return "", "", false
		})

	// Dynamic lookup should take priority over static map
	client, err := runner.clientForInput(&AgentTaskInput{
		AgentLLM: &AgentLLMConfig{Provider: "dual", Model: "test-model"},
	})
	if err != nil {
		t.Fatalf("clientForInput returned error: %v", err)
	}
	if client.cfg.BaseURL != "https://dynamic.example.com/v1" {
		t.Errorf("expected dynamic URL, got %s", client.cfg.BaseURL)
	}
	if client.cfg.APIKey != "dynamic-key" {
		t.Errorf("expected dynamic key, got %s", client.cfg.APIKey)
	}
}

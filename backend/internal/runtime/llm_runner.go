package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

// LLMAgentRunner implements AgentRunner by calling an OpenAI-compatible LLM.
type LLMAgentRunner struct {
	client *LLMClient
	role   string
	spec   string
	logger *slog.Logger
}

func NewLLMRunner(client *LLMClient, role, specialization string, logger *slog.Logger) *LLMAgentRunner {
	return &LLMAgentRunner{
		client: client,
		role:   role,
		spec:   specialization,
		logger: logger,
	}
}

func (r *LLMAgentRunner) Role() string          { return r.role }
func (r *LLMAgentRunner) Specialization() string { return r.spec }

func (r *LLMAgentRunner) Execute(input *AgentTaskInput) (*AgentTaskOutput, error) {
	systemPrompt := buildSystemPrompt(r.role, r.spec)
	userPrompt, err := buildUserPrompt(input)
	if err != nil {
		return nil, fmt.Errorf("build prompt: %w", err)
	}

	r.logger.Info("LLM agent executing",
		"role", r.role,
		"spec", r.spec,
		"instruction", input.Instruction,
	)

	ctx := context.Background()
	raw, err := r.client.ChatJSON(ctx, systemPrompt, userPrompt)
	if err != nil {
		r.logger.Error("LLM call failed", "role", r.role, "error", err)
		return &AgentTaskOutput{
			Status: OutputStatusFailed,
			Error:  fmt.Sprintf("LLM call failed: %s", err.Error()),
		}, nil
	}

	r.logger.Debug("LLM raw response", "role", r.role, "response_length", len(raw))

	var output AgentTaskOutput
	if err := json.Unmarshal([]byte(raw), &output); err != nil {
		r.logger.Error("LLM output parse failed", "role", r.role, "raw_prefix", truncateStr(raw, 500), "error", err)
		return &AgentTaskOutput{
			Status: OutputStatusFailed,
			Error:  fmt.Sprintf("failed to parse LLM response as JSON: %s", err.Error()),
		}, nil
	}

	if output.Status == "" {
		output.Status = OutputStatusSuccess
	}

	r.logger.Info("LLM agent completed",
		"role", r.role,
		"status", output.Status,
		"phases", len(output.Phases),
		"tasks", len(output.Tasks),
		"artifacts", len(output.Artifacts),
		"reviews", len(output.Reviews),
	)

	return &output, nil
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// RegisterLLMRunners registers LLM-based runners for all roles into the given registry.
// This replaces mock runners with real LLM-powered agents.
func RegisterLLMRunners(reg *Registry, client *LLMClient, logger *slog.Logger) {
	reg.Register("pm", NewLLMRunner(client, "pm", "", logger))
	reg.Register("supervisor", NewLLMRunner(client, "supervisor", "", logger))
	reg.Register("worker", NewLLMRunner(client, "worker", "general", logger))
	reg.Register("reviewer", NewLLMRunner(client, "reviewer", "", logger))

	// Specialized workers
	reg.RegisterSpecialized("worker", "backend", NewLLMRunner(client, "worker", "backend", logger))
	reg.RegisterSpecialized("worker", "frontend", NewLLMRunner(client, "worker", "frontend", logger))
	reg.RegisterSpecialized("worker", "qa", NewLLMRunner(client, "worker", "qa", logger))
}

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

func (r *LLMAgentRunner) Role() string           { return r.role }
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
// Each role can have its own LLM config (model/base_url/api_key); unconfigured fields
// fall back to the global defaults provided by defaultClient.
func RegisterLLMRunners(reg *Registry, defaultClient *LLMClient, roleClients map[string]*LLMClient, logger *slog.Logger) {
	getClient := func(role string) *LLMClient {
		if c, ok := roleClients[role]; ok {
			return c
		}
		return defaultClient
	}

	reg.Register("pm", NewLLMRunner(getClient("pm"), "pm", "", logger))
	reg.Register("supervisor", NewLLMRunner(getClient("supervisor"), "supervisor", "", logger))
	reg.Register("worker", NewLLMRunner(getClient("worker"), "worker", "general", logger))
	reg.Register("reviewer", NewLLMRunner(getClient("reviewer"), "reviewer", "", logger))

	// Specialized workers share the worker client
	wc := getClient("worker")
	reg.RegisterSpecialized("worker", "backend", NewLLMRunner(wc, "worker", "backend", logger))
	reg.RegisterSpecialized("worker", "frontend", NewLLMRunner(wc, "worker", "frontend", logger))
	reg.RegisterSpecialized("worker", "qa", NewLLMRunner(wc, "worker", "qa", logger))
}

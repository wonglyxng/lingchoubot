package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

// LLMAgentRunner implements AgentRunner by calling an OpenAI-compatible LLM.
type LLMAgentRunner struct {
	client          *LLMClient
	role            string
	spec            string
	logger          *slog.Logger
	providerConfigs map[string]LLMClientConfig
	providerLookup  ProviderConfigLookup // dynamic lookup (e.g. DB-backed), tried before static map
	fallback        AgentRunner          // optional fallback runner for degradation
}

// ProviderConfigLookup resolves a provider's connection config dynamically.
// Returns (baseURL, apiKey, found).
type ProviderConfigLookup func(providerKey string) (baseURL, apiKey string, found bool)

// LastCallMeta stores metadata from the most recent LLM call (for audit/testing).
// Only accessed from the goroutine that called Execute; not thread-safe by design.
type LastCallMeta struct {
	Meta            *LLMCallMeta
	PromptVersion   PromptVersion
	ValidationError error
	UsedFallback    bool
}

func NewLLMRunner(client *LLMClient, role, specialization string, logger *slog.Logger) *LLMAgentRunner {
	return &LLMAgentRunner{
		client:          client,
		role:            role,
		spec:            specialization,
		logger:          logger,
		providerConfigs: map[string]LLMClientConfig{},
	}
}

// WithFallback sets a fallback runner for degradation when LLM fails.
func (r *LLMAgentRunner) WithFallback(fb AgentRunner) *LLMAgentRunner {
	r.fallback = fb
	return r
}

func (r *LLMAgentRunner) WithProviderConfigs(configs map[string]LLMClientConfig) *LLMAgentRunner {
	r.providerConfigs = make(map[string]LLMClientConfig, len(configs))
	for key, cfg := range configs {
		r.providerConfigs[key] = cfg
	}
	return r
}

// WithProviderLookup sets a dynamic lookup function for provider configs.
// When set, this is tried before the static providerConfigs map.
func (r *LLMAgentRunner) WithProviderLookup(fn ProviderConfigLookup) *LLMAgentRunner {
	r.providerLookup = fn
	return r
}

func (r *LLMAgentRunner) Role() string           { return r.role }
func (r *LLMAgentRunner) Specialization() string { return r.spec }

// Execute calls the LLM, validates output, and falls back to mock runner on failure.
// Returns (*AgentTaskOutput, error). The output includes metadata via LastMeta().
func (r *LLMAgentRunner) Execute(input *AgentTaskInput) (*AgentTaskOutput, error) {
	pv := GetPromptVersion(r.role, r.spec)
	systemPrompt := buildSystemPrompt(r.role, r.spec)
	userPrompt, err := buildUserPrompt(input)
	if err != nil {
		return nil, fmt.Errorf("build prompt: %w", err)
	}

	r.logger.Info("LLM agent executing",
		"role", r.role,
		"spec", r.spec,
		"prompt_version", pv.Version,
		"instruction", input.Instruction,
	)

	client, err := r.clientForInput(input)
	if err != nil {
		r.logger.Error("resolve llm client failed", "role", r.role, "error", err)
		if r.fallback != nil {
			r.logger.Warn("falling back to mock runner", "role", r.role)
			output, fbErr := r.fallback.Execute(input)
			if fbErr == nil && output != nil {
				output.Summary = "[降级] " + output.Summary
			}
			return output, fbErr
		}
		return &AgentTaskOutput{
			Status: OutputStatusFailed,
			Error:  fmt.Sprintf("resolve llm client failed: %s", err.Error()),
		}, nil
	}

	ctx := context.Background()
	raw, meta, err := client.ChatJSONWithMeta(ctx, systemPrompt, userPrompt)
	if meta != nil {
		meta.PromptVersion = pv.Version
	}

	if err != nil {
		r.logger.Error("LLM call failed", "role", r.role, "error", err,
			"duration_ms", metaDurationMs(meta), "model", metaModel(meta))

		// Fallback to mock runner if available
		if r.fallback != nil {
			r.logger.Warn("falling back to mock runner", "role", r.role)
			output, fbErr := r.fallback.Execute(input)
			if fbErr == nil && output != nil {
				output.Summary = "[降级] " + output.Summary
			}
			return output, fbErr
		}

		return &AgentTaskOutput{
			Status: OutputStatusFailed,
			Error:  fmt.Sprintf("LLM call failed: %s", err.Error()),
		}, nil
	}

	r.logger.Debug("LLM raw response", "role", r.role, "response_length", len(raw),
		"duration_ms", metaDurationMs(meta),
		"tokens", metaTotalTokens(meta))

	var output AgentTaskOutput
	if err := json.Unmarshal([]byte(raw), &output); err != nil {
		r.logger.Error("LLM output parse failed", "role", r.role,
			"raw_prefix", truncateStr(raw, 500), "error", err)

		if r.fallback != nil {
			r.logger.Warn("falling back to mock runner after parse failure", "role", r.role)
			out, fbErr := r.fallback.Execute(input)
			if fbErr == nil && out != nil {
				out.Summary = "[降级] " + out.Summary
			}
			return out, fbErr
		}

		return &AgentTaskOutput{
			Status: OutputStatusFailed,
			Error:  fmt.Sprintf("failed to parse LLM response as JSON: %s", err.Error()),
		}, nil
	}

	if output.Status == "" {
		output.Status = OutputStatusSuccess
	}

	// Validate output against role-specific schema
	if valErr := ValidateOutput(r.role, r.spec, &output); valErr != nil {
		r.logger.Warn("LLM output validation failed", "role", r.role, "error", valErr,
			"prompt_version", pv.Version)

		if r.fallback != nil {
			r.logger.Warn("falling back to mock runner after validation failure", "role", r.role)
			out, fbErr := r.fallback.Execute(input)
			if fbErr == nil && out != nil {
				out.Summary = "[降级] " + out.Summary
			}
			return out, fbErr
		}

		// Return original output with error annotation rather than blocking
		output.Error = fmt.Sprintf("output validation: %s", valErr.Error())
	}

	r.logger.Info("LLM agent completed",
		"role", r.role,
		"status", output.Status,
		"prompt_version", pv.Version,
		"duration_ms", metaDurationMs(meta),
		"tokens", metaTotalTokens(meta),
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

func metaDurationMs(m *LLMCallMeta) int64 {
	if m == nil {
		return 0
	}
	return m.DurationMs
}

func metaModel(m *LLMCallMeta) string {
	if m == nil {
		return ""
	}
	return m.Model
}

func metaTotalTokens(m *LLMCallMeta) int {
	if m == nil {
		return 0
	}
	return m.TotalTokens
}

func (r *LLMAgentRunner) clientForInput(input *AgentTaskInput) (*LLMClient, error) {
	if input == nil || input.AgentLLM == nil {
		return r.client, nil
	}

	cfg := r.client.cfg
	if input.AgentLLM.Provider != "" {
		// 1. Try dynamic lookup (DB-backed) first
		if r.providerLookup != nil {
			if baseURL, apiKey, found := r.providerLookup(input.AgentLLM.Provider); found {
				cfg.BaseURL = baseURL
				cfg.APIKey = apiKey
			} else {
				// 2. Fall back to static map (env-var based)
				providerCfg, ok := r.providerConfigs[input.AgentLLM.Provider]
				if !ok {
					return nil, fmt.Errorf("unsupported llm provider %q", input.AgentLLM.Provider)
				}
				if providerCfg.BaseURL == "" {
					return nil, fmt.Errorf("llm provider %q missing base_url", input.AgentLLM.Provider)
				}
				cfg.BaseURL = providerCfg.BaseURL
				cfg.APIKey = providerCfg.APIKey
			}
		} else {
			providerCfg, ok := r.providerConfigs[input.AgentLLM.Provider]
			if !ok {
				return nil, fmt.Errorf("unsupported llm provider %q", input.AgentLLM.Provider)
			}
			if providerCfg.BaseURL == "" {
				return nil, fmt.Errorf("llm provider %q missing base_url", input.AgentLLM.Provider)
			}
			cfg.BaseURL = providerCfg.BaseURL
			cfg.APIKey = providerCfg.APIKey
		}
	}
	if input.AgentLLM.Model != "" {
		cfg.Model = input.AgentLLM.Model
	}
	return NewLLMClient(cfg), nil
}

// RegisterLLMRunners registers LLM-based runners for all roles into the given registry.
// Each role can have its own LLM config (model/base_url/api_key); unconfigured fields
// fall back to the global defaults provided by defaultClient.
// When enableFallback is true, each LLM runner is configured with a mock fallback.
func RegisterLLMRunners(reg *Registry, defaultClient *LLMClient, roleClients map[string]*LLMClient, providerConfigs map[string]LLMClientConfig, logger *slog.Logger) {
	RegisterLLMRunnersWithFallback(reg, defaultClient, roleClients, providerConfigs, nil, logger, false)
}

// RegisterLLMRunnersWithFallback registers LLM runners with optional mock fallback.
func RegisterLLMRunnersWithFallback(reg *Registry, defaultClient *LLMClient, roleClients map[string]*LLMClient, providerConfigs map[string]LLMClientConfig, providerLookup ProviderConfigLookup, logger *slog.Logger, enableFallback bool) {
	getClient := func(role string) *LLMClient {
		if c, ok := roleClients[role]; ok {
			return c
		}
		return defaultClient
	}

	pmRunner := NewLLMRunner(getClient("pm"), "pm", "", logger).WithProviderConfigs(providerConfigs).WithProviderLookup(providerLookup)
	supRunner := NewLLMRunner(getClient("supervisor"), "supervisor", "", logger).WithProviderConfigs(providerConfigs).WithProviderLookup(providerLookup)
	wrkRunner := NewLLMRunner(getClient("worker"), "worker", "general", logger).WithProviderConfigs(providerConfigs).WithProviderLookup(providerLookup)
	revRunner := NewLLMRunner(getClient("reviewer"), "reviewer", "", logger).WithProviderConfigs(providerConfigs).WithProviderLookup(providerLookup)

	if enableFallback {
		pmRunner.WithFallback(&MockPMAgent{})
		supRunner.WithFallback(&MockSupervisorAgent{})
		wrkRunner.WithFallback(&MockWorkerAgent{})
		revRunner.WithFallback(&MockReviewerAgent{})
	}

	reg.Register("pm", pmRunner)
	reg.Register("supervisor", supRunner)
	reg.Register("worker", wrkRunner)
	reg.Register("reviewer", revRunner)

	// Specialized workers share the worker client
	wc := getClient("worker")
	bw := NewLLMRunner(wc, "worker", "backend", logger).WithProviderConfigs(providerConfigs).WithProviderLookup(providerLookup)
	fw := NewLLMRunner(wc, "worker", "frontend", logger).WithProviderConfigs(providerConfigs).WithProviderLookup(providerLookup)
	qw := NewLLMRunner(wc, "worker", "qa", logger).WithProviderConfigs(providerConfigs).WithProviderLookup(providerLookup)
	if enableFallback {
		bw.WithFallback(&MockBackendWorkerAgent{})
		fw.WithFallback(&MockFrontendWorkerAgent{})
		qw.WithFallback(&MockQAWorkerAgent{})
	}
	reg.RegisterSpecialized("worker", "backend", bw)
	reg.RegisterSpecialized("worker", "frontend", fw)
	reg.RegisterSpecialized("worker", "qa", qw)
}

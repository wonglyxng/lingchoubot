package orchestrator

import (
	"fmt"
	"strings"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/runtime"
)

func runtimeAgentLLMConfig(agent *model.Agent) (*runtime.AgentLLMConfig, error) {
	if agent == nil {
		return nil, fmt.Errorf("agent is required")
	}
	if agent.AgentType != model.AgentTypeLLM {
		return nil, fmt.Errorf("agent %s must use llm runtime, got %s", agent.ID, agent.AgentType)
	}
	cfg, err := agent.GetLLMConfig()
	if err != nil {
		return nil, fmt.Errorf("parse agent llm config: %w", err)
	}
	if cfg == nil {
		return nil, fmt.Errorf("agent %s missing llm config", agent.ID)
	}
	if !model.IsSupportedAgentLLMProvider(cfg.Provider) {
		return nil, fmt.Errorf("agent %s has unsupported llm provider %q", agent.ID, cfg.Provider)
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return nil, fmt.Errorf("agent %s missing llm model", agent.ID)
	}
	return &runtime.AgentLLMConfig{
		Provider: string(cfg.Provider),
		Model:    cfg.Model,
	}, nil
}

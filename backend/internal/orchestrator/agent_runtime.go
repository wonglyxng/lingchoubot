package orchestrator

import (
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/runtime"
)

func runtimeAgentLLMConfig(agent *model.Agent) (*runtime.AgentLLMConfig, error) {
	if agent == nil || agent.AgentType != model.AgentTypeLLM {
		return nil, nil
	}
	cfg, err := agent.GetLLMConfig()
	if err != nil {
		return nil, fmt.Errorf("parse agent llm config: %w", err)
	}
	if cfg == nil {
		return nil, nil
	}
	return &runtime.AgentLLMConfig{
		Provider: string(cfg.Provider),
		Model:    cfg.Model,
	}, nil
}

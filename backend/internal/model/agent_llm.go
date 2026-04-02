package model

import (
	"encoding/json"
	"fmt"
)

type AgentLLMProvider string

const (
	AgentLLMProviderOpenAI      AgentLLMProvider = "openai"
	AgentLLMProviderDeepSeek    AgentLLMProvider = "deepseek"
	AgentLLMProviderQwen        AgentLLMProvider = "qwen"
	AgentLLMProviderMoonshot    AgentLLMProvider = "moonshot"
	AgentLLMProviderZhiPu       AgentLLMProvider = "zhipu"
	AgentLLMProviderSiliconFlow AgentLLMProvider = "siliconflow"
	AgentLLMProviderOpenRouter  AgentLLMProvider = "openrouter"
	AgentLLMProviderOllama      AgentLLMProvider = "ollama"
)

const (
	DefaultAgentLLMProvider AgentLLMProvider = AgentLLMProviderDeepSeek
	DefaultAgentLLMModel    string           = "deepseek-chat"
)

type AgentLLMConfig struct {
	Provider AgentLLMProvider `json:"provider"`
	Model    string           `json:"model"`
}

func SupportedAgentLLMProviders() []AgentLLMProvider {
	return []AgentLLMProvider{
		AgentLLMProviderOpenAI,
		AgentLLMProviderDeepSeek,
		AgentLLMProviderQwen,
		AgentLLMProviderMoonshot,
		AgentLLMProviderZhiPu,
		AgentLLMProviderSiliconFlow,
		AgentLLMProviderOpenRouter,
		AgentLLMProviderOllama,
	}
}

func IsSupportedAgentLLMProvider(provider AgentLLMProvider) bool {
	for _, item := range SupportedAgentLLMProviders() {
		if item == provider {
			return true
		}
	}
	return false
}

func (a *Agent) GetLLMConfig() (*AgentLLMConfig, error) {
	metadata, err := decodeMetadataMap(a.Metadata)
	if err != nil {
		return nil, err
	}
	raw, ok := metadata["llm"]
	if !ok || raw == nil {
		return nil, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal agent llm metadata: %w", err)
	}
	var cfg AgentLLMConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal agent llm metadata: %w", err)
	}
	return &cfg, nil
}

func (a *Agent) SetLLMConfig(cfg *AgentLLMConfig) error {
	metadata, err := decodeMetadataMap(a.Metadata)
	if err != nil {
		return err
	}
	if cfg == nil {
		delete(metadata, "llm")
	} else {
		metadata["llm"] = cfg
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal agent metadata: %w", err)
	}
	a.Metadata = JSON(encoded)
	return nil
}

func decodeMetadataMap(metadata JSON) (map[string]any, error) {
	if len(metadata) == 0 {
		return map[string]any{}, nil
	}
	var parsed map[string]any
	if err := json.Unmarshal(metadata, &parsed); err != nil {
		return nil, fmt.Errorf("decode agent metadata: %w", err)
	}
	if parsed == nil {
		return map[string]any{}, nil
	}
	return parsed, nil
}

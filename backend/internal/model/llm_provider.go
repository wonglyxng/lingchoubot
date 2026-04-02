package model

import "time"

// LLMProvider 代表一个大模型供应商配置
type LLMProvider struct {
	ID        string      `json:"id"`
	Key       string      `json:"key"`
	Name      string      `json:"name"`
	BaseURL   string      `json:"base_url"`
	APIKey    string      `json:"api_key,omitempty"`
	IsBuiltin bool        `json:"is_builtin"`
	IsEnabled bool        `json:"is_enabled"`
	SortOrder int         `json:"sort_order"`
	Metadata  JSON        `json:"metadata"`
	Models    []*LLMModel `json:"models,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// MaskedAPIKey 返回脱敏后的 API Key（仅显示末4位）
func (p *LLMProvider) MaskedAPIKey() string {
	if len(p.APIKey) == 0 {
		return ""
	}
	if len(p.APIKey) <= 4 {
		return "****"
	}
	return "****" + p.APIKey[len(p.APIKey)-4:]
}

// LLMModel 代表某供应商下的模型预设
type LLMModel struct {
	ID         string    `json:"id"`
	ProviderID string    `json:"provider_id"`
	ModelID    string    `json:"model_id"`
	Name       string    `json:"name"`
	IsDefault  bool      `json:"is_default"`
	SortOrder  int       `json:"sort_order"`
	Metadata   JSON      `json:"metadata"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

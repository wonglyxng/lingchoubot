package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/lingchou/lingchoubot/backend/internal/model"
)

type LLMProviderService struct {
	repo  LLMProviderRepository
	audit *AuditService
}

func NewLLMProviderService(repo LLMProviderRepository, audit *AuditService) *LLMProviderService {
	return &LLMProviderService{repo: repo, audit: audit}
}

// --- Provider CRUD ---

func (s *LLMProviderService) Create(ctx context.Context, p *model.LLMProvider) error {
	if err := validateProvider(p); err != nil {
		return err
	}
	if len(p.Metadata) == 0 {
		p.Metadata = model.JSON("{}")
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return fmt.Errorf("create llm_provider: %w", err)
	}
	s.audit.LogEvent(ctx, "user", "", "llm_provider.created",
		fmt.Sprintf("LLM 供应商「%s」(%s) 已创建", p.Name, p.Key),
		"llm_provider", p.ID, nil, p)
	return nil
}

func (s *LLMProviderService) GetByID(ctx context.Context, id string) (*model.LLMProvider, error) {
	return s.repo.GetByID(ctx, id)
}

// ListWithModels returns all providers (optionally enabled-only) with their models populated.
func (s *LLMProviderService) ListWithModels(ctx context.Context, enabledOnly bool) ([]*model.LLMProvider, error) {
	providers, err := s.repo.List(ctx, enabledOnly)
	if err != nil {
		return nil, err
	}
	models, err := s.repo.ListAllModels(ctx)
	if err != nil {
		return nil, err
	}
	// group models by provider_id
	byProvider := make(map[string][]*model.LLMModel, len(providers))
	for _, m := range models {
		byProvider[m.ProviderID] = append(byProvider[m.ProviderID], m)
	}
	for _, p := range providers {
		p.Models = byProvider[p.ID]
	}
	return providers, nil
}

func (s *LLMProviderService) Update(ctx context.Context, p *model.LLMProvider) error {
	if err := validateProvider(p); err != nil {
		return err
	}
	existing, err := s.repo.GetByID(ctx, p.ID)
	if err != nil {
		return fmt.Errorf("get existing provider: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("provider not found: %s", p.ID)
	}
	// If api_key is masked (starts with ****), preserve original
	if strings.HasPrefix(p.APIKey, "****") {
		p.APIKey = existing.APIKey
	}
	if len(p.Metadata) == 0 {
		p.Metadata = model.JSON("{}")
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return fmt.Errorf("update llm_provider: %w", err)
	}
	s.audit.LogEvent(ctx, "user", "", "llm_provider.updated",
		fmt.Sprintf("LLM 供应商「%s」已更新", p.Name),
		"llm_provider", p.ID, existing, p)
	return nil
}

func (s *LLMProviderService) Delete(ctx context.Context, id string) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get existing provider: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("provider not found: %s", id)
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete llm_provider: %w", err)
	}
	s.audit.LogEvent(ctx, "user", "", "llm_provider.deleted",
		fmt.Sprintf("LLM 供应商「%s」已删除", existing.Name),
		"llm_provider", id, existing, nil)
	return nil
}

// MaskProviderAPIKey 脱敏处理 - 用于 API 返回
func MaskProviderAPIKey(p *model.LLMProvider) {
	p.APIKey = p.MaskedAPIKey()
}

// MaskProvidersAPIKey 批量脱敏
func MaskProvidersAPIKey(providers []*model.LLMProvider) {
	for _, p := range providers {
		MaskProviderAPIKey(p)
	}
}

// --- Model CRUD ---

func (s *LLMProviderService) CreateModel(ctx context.Context, m *model.LLMModel) error {
	if err := validateModel(m); err != nil {
		return err
	}
	if len(m.Metadata) == 0 {
		m.Metadata = model.JSON("{}")
	}
	if err := s.repo.CreateModel(ctx, m); err != nil {
		return fmt.Errorf("create llm_model: %w", err)
	}
	s.audit.LogEvent(ctx, "user", "", "llm_model.created",
		fmt.Sprintf("模型预设「%s」(%s) 已创建", m.Name, m.ModelID),
		"llm_model", m.ID, nil, m)
	return nil
}

func (s *LLMProviderService) GetModelByID(ctx context.Context, id string) (*model.LLMModel, error) {
	return s.repo.GetModelByID(ctx, id)
}

func (s *LLMProviderService) ListModelsByProvider(ctx context.Context, providerID string) ([]*model.LLMModel, error) {
	return s.repo.ListModelsByProvider(ctx, providerID)
}

func (s *LLMProviderService) UpdateModel(ctx context.Context, m *model.LLMModel) error {
	if err := validateModel(m); err != nil {
		return err
	}
	if len(m.Metadata) == 0 {
		m.Metadata = model.JSON("{}")
	}
	if err := s.repo.UpdateModel(ctx, m); err != nil {
		return fmt.Errorf("update llm_model: %w", err)
	}
	s.audit.LogEvent(ctx, "user", "", "llm_model.updated",
		fmt.Sprintf("模型预设「%s」已更新", m.Name),
		"llm_model", m.ID, nil, m)
	return nil
}

func (s *LLMProviderService) DeleteModel(ctx context.Context, id string) error {
	existing, err := s.repo.GetModelByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get existing model: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("model not found: %s", id)
	}
	if err := s.repo.DeleteModel(ctx, id); err != nil {
		return fmt.Errorf("delete llm_model: %w", err)
	}
	s.audit.LogEvent(ctx, "user", "", "llm_model.deleted",
		fmt.Sprintf("模型预设「%s」已删除", existing.Name),
		"llm_model", id, existing, nil)
	return nil
}

// --- Runtime helper ---

// GetProviderConfig resolves the runtime-usable config for a provider key.
// Returns (baseURL, apiKey, found).
func (s *LLMProviderService) GetProviderConfig(ctx context.Context, key string) (baseURL, apiKey string, found bool) {
	p, err := s.repo.GetByKey(ctx, key)
	if err != nil || p == nil || !p.IsEnabled {
		return "", "", false
	}
	return p.BaseURL, p.APIKey, true
}

// --- Validation ---

func validateProvider(p *model.LLMProvider) error {
	if strings.TrimSpace(p.Key) == "" {
		return fmt.Errorf("provider key is required")
	}
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("provider name is required")
	}
	if strings.TrimSpace(p.BaseURL) == "" {
		return fmt.Errorf("provider base_url is required")
	}
	return nil
}

func validateModel(m *model.LLMModel) error {
	if strings.TrimSpace(m.ProviderID) == "" {
		return fmt.Errorf("provider_id is required")
	}
	if strings.TrimSpace(m.ModelID) == "" {
		return fmt.Errorf("model_id is required")
	}
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("model name is required")
	}
	return nil
}

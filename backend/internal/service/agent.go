package service

import (
	"context"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

type AgentService struct {
	repo  *repository.AgentRepo
	audit *AuditService
}

func NewAgentService(repo *repository.AgentRepo, audit *AuditService) *AgentService {
	return &AgentService{repo: repo, audit: audit}
}

func (s *AgentService) Create(ctx context.Context, a *model.Agent) error {
	if a.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if a.Role == "" {
		return fmt.Errorf("agent role is required")
	}
	if a.Status == "" {
		a.Status = model.AgentStatusActive
	}
	if a.AgentType == "" {
		a.AgentType = model.AgentTypeMock
	}
	if a.Specialization == "" {
		a.Specialization = model.AgentSpecGeneral
	}
	if len(a.Capabilities) == 0 {
		a.Capabilities = model.JSON("[]")
	}
	if len(a.Metadata) == 0 {
		a.Metadata = model.JSON("{}")
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return fmt.Errorf("create agent: %w", err)
	}
	s.audit.LogEvent(ctx, "user", "", "agent.created",
		fmt.Sprintf("Agent「%s」(%s/%s) 已注册", a.Name, a.Role, a.Specialization),
		"agent", a.ID, nil, a)
	return nil
}

func (s *AgentService) GetByID(ctx context.Context, id string) (*model.Agent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *AgentService) List(ctx context.Context, limit, offset int) ([]*model.Agent, int, error) {
	return s.repo.List(ctx, limit, offset)
}

func (s *AgentService) Update(ctx context.Context, a *model.Agent) error {
	old, err := s.repo.GetByID(ctx, a.ID)
	if err != nil {
		return err
	}
	if old == nil {
		return fmt.Errorf("agent not found")
	}
	if err := s.repo.Update(ctx, a); err != nil {
		return fmt.Errorf("update agent: %w", err)
	}
	s.audit.LogEvent(ctx, "user", "", "agent.updated",
		fmt.Sprintf("Agent「%s」已更新", a.Name),
		"agent", a.ID, old, a)
	return nil
}

func (s *AgentService) Delete(ctx context.Context, id string) error {
	old, _ := s.repo.GetByID(ctx, id)
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.audit.LogEvent(ctx, "user", "", "agent.deleted",
		"Agent 已删除", "agent", id, old, nil)
	return nil
}

func (s *AgentService) GetSubordinates(ctx context.Context, agentID string) ([]*model.Agent, error) {
	return s.repo.GetSubordinates(ctx, agentID)
}

// FindByRoleAndSpec finds the best matching active agent for a role + specialization.
func (s *AgentService) FindByRoleAndSpec(ctx context.Context, role model.AgentRole, spec model.AgentSpecialization) (*model.Agent, error) {
	return s.repo.FindByRoleAndSpec(ctx, role, spec)
}

// GetOrgTree returns a flat list ordered by depth. If rootID is empty, returns full tree.
func (s *AgentService) GetOrgTree(ctx context.Context, rootID string) ([]*model.Agent, error) {
	return s.repo.GetOrgTree(ctx, rootID)
}

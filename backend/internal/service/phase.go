package service

import (
	"context"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

type PhaseService struct {
	repo    *repository.PhaseRepo
	projSvc *ProjectService
	audit   *AuditService
}

func NewPhaseService(repo *repository.PhaseRepo, projSvc *ProjectService, audit *AuditService) *PhaseService {
	return &PhaseService{repo: repo, projSvc: projSvc, audit: audit}
}

func (s *PhaseService) Create(ctx context.Context, p *model.Phase) error {
	if p.ProjectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if p.Name == "" {
		return fmt.Errorf("phase name is required")
	}
	proj, err := s.projSvc.GetByID(ctx, p.ProjectID)
	if err != nil {
		return err
	}
	if proj == nil {
		return fmt.Errorf("project not found")
	}
	if p.Status == "" {
		p.Status = model.PhaseStatusPending
	}
	if len(p.Metadata) == 0 {
		p.Metadata = model.JSON("{}")
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return fmt.Errorf("create phase: %w", err)
	}
	s.audit.LogEvent(ctx, "user", "", "phase.created",
		fmt.Sprintf("阶段「%s」已创建", p.Name),
		"phase", p.ID, nil, p)
	return nil
}

func (s *PhaseService) GetByID(ctx context.Context, id string) (*model.Phase, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *PhaseService) ListByProject(ctx context.Context, projectID string) ([]*model.Phase, error) {
	return s.repo.ListByProject(ctx, projectID)
}

func (s *PhaseService) Update(ctx context.Context, p *model.Phase) error {
	old, err := s.repo.GetByID(ctx, p.ID)
	if err != nil {
		return err
	}
	if old == nil {
		return fmt.Errorf("phase not found")
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return fmt.Errorf("update phase: %w", err)
	}
	s.audit.LogEvent(ctx, "user", "", "phase.updated",
		fmt.Sprintf("阶段「%s」已更新", p.Name),
		"phase", p.ID, old, p)
	return nil
}

func (s *PhaseService) Delete(ctx context.Context, id string) error {
	old, _ := s.repo.GetByID(ctx, id)
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.audit.LogEvent(ctx, "user", "", "phase.deleted",
		"阶段已删除", "phase", id, old, nil)
	return nil
}

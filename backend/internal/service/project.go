package service

import (
	"context"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

type ProjectService struct {
	repo  *repository.ProjectRepo
	audit *AuditService
}

func NewProjectService(repo *repository.ProjectRepo, audit *AuditService) *ProjectService {
	return &ProjectService{repo: repo, audit: audit}
}

func (s *ProjectService) Create(ctx context.Context, p *model.Project) error {
	if p.Name == "" {
		return fmt.Errorf("project name is required")
	}
	if p.Status == "" {
		p.Status = model.ProjectStatusPlanning
	}
	if len(p.Metadata) == 0 {
		p.Metadata = model.JSON("{}")
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	s.audit.LogEvent(ctx, "user", "", "project.created",
		fmt.Sprintf("项目「%s」已创建", p.Name),
		"project", p.ID, nil, p)
	return nil
}

func (s *ProjectService) GetByID(ctx context.Context, id string) (*model.Project, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ProjectService) List(ctx context.Context, limit, offset int) ([]*model.Project, int, error) {
	return s.repo.List(ctx, limit, offset)
}

func (s *ProjectService) Update(ctx context.Context, p *model.Project) error {
	old, err := s.repo.GetByID(ctx, p.ID)
	if err != nil {
		return err
	}
	if old == nil {
		return fmt.Errorf("project not found")
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	s.audit.LogEvent(ctx, "user", "", "project.updated",
		fmt.Sprintf("项目「%s」已更新", p.Name),
		"project", p.ID, old, p)
	return nil
}

func (s *ProjectService) Delete(ctx context.Context, id string) error {
	old, _ := s.repo.GetByID(ctx, id)
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.audit.LogEvent(ctx, "user", "", "project.deleted",
		"项目已删除", "project", id, old, nil)
	return nil
}

package service

import (
	"context"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

type ArtifactService struct {
	repo    *repository.ArtifactRepo
	verRepo *repository.ArtifactVersionRepo
	audit   *AuditService
}

func NewArtifactService(repo *repository.ArtifactRepo, verRepo *repository.ArtifactVersionRepo, audit *AuditService) *ArtifactService {
	return &ArtifactService{repo: repo, verRepo: verRepo, audit: audit}
}

func (s *ArtifactService) Create(ctx context.Context, a *model.Artifact) error {
	if a.ProjectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if a.Name == "" {
		return fmt.Errorf("artifact name is required")
	}
	if a.ArtifactType == "" {
		return fmt.Errorf("artifact_type is required")
	}
	if len(a.Metadata) == 0 {
		a.Metadata = model.JSON("{}")
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return fmt.Errorf("create artifact: %w", err)
	}
	s.audit.LogEvent(ctx, "agent", strOrEmpty(a.CreatedBy), "artifact.created",
		fmt.Sprintf("工件「%s」(%s) 已创建", a.Name, a.ArtifactType),
		"artifact", a.ID, nil, a)
	return nil
}

func (s *ArtifactService) GetByID(ctx context.Context, id string) (*model.Artifact, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ArtifactService) List(ctx context.Context, p repository.ArtifactListParams) ([]*model.Artifact, int, error) {
	return s.repo.List(ctx, p)
}

// AddVersion auto-increments the version number for the given artifact.
func (s *ArtifactService) AddVersion(ctx context.Context, v *model.ArtifactVersion) error {
	if v.ArtifactID == "" {
		return fmt.Errorf("artifact_id is required")
	}
	art, err := s.repo.GetByID(ctx, v.ArtifactID)
	if err != nil {
		return err
	}
	if art == nil {
		return fmt.Errorf("artifact not found")
	}
	nextVer, err := s.verRepo.NextVersion(ctx, v.ArtifactID)
	if err != nil {
		return err
	}
	v.Version = nextVer
	if len(v.Metadata) == 0 {
		v.Metadata = model.JSON("{}")
	}
	if err := s.verRepo.Create(ctx, v); err != nil {
		return fmt.Errorf("create artifact version: %w", err)
	}
	s.audit.LogEvent(ctx, "agent", strOrEmpty(v.CreatedBy), "artifact_version.created",
		fmt.Sprintf("工件「%s」新版本 v%d 已创建", art.Name, v.Version),
		"artifact_version", v.ID, nil, v)
	return nil
}

func (s *ArtifactService) ListVersions(ctx context.Context, artifactID string) ([]*model.ArtifactVersion, error) {
	return s.verRepo.ListByArtifact(ctx, artifactID)
}

func (s *ArtifactService) GetVersionByID(ctx context.Context, id string) (*model.ArtifactVersion, error) {
	return s.verRepo.GetByID(ctx, id)
}

func strOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

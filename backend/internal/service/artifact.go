package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

type ArtifactContentStore interface {
	Store(ctx context.Context, name, content, contentType string) (uri string, sizeBytes int64, checksum string, err error)
}

type ArtifactService struct {
	repo    ArtifactRepository
	verRepo ArtifactVersionRepository
	audit   *AuditService
	store   ArtifactContentStore
}

func NewArtifactService(repo ArtifactRepository, verRepo ArtifactVersionRepository, audit *AuditService) *ArtifactService {
	return &ArtifactService{repo: repo, verRepo: verRepo, audit: audit}
}

func (s *ArtifactService) SetContentStore(store ArtifactContentStore) {
	s.store = store
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
	if err := s.prepareVersionContent(ctx, art, v); err != nil {
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

func (s *ArtifactService) prepareVersionContent(ctx context.Context, art *model.Artifact, v *model.ArtifactVersion) error {
	meta, err := artifactVersionMetadata(v.Metadata)
	if err != nil {
		return fmt.Errorf("parse artifact version metadata: %w", err)
	}

	content := strings.TrimSpace(v.Content)
	if content == "" {
		if v.Metadata == nil {
			v.Metadata = model.JSON("{}")
		}
		return nil
	}

	sourceName := strings.TrimSpace(v.SourceName)
	if sourceName == "" {
		sourceName = deriveArtifactSourceName(art.Name, v.ContentType)
	}
	meta["inline_content"] = v.Content
	meta["source_name"] = sourceName

	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(v.Content)))
	if s.store != nil {
		uri, sizeBytes, storedChecksum, storeErr := s.store.Store(ctx, sourceName, v.Content, v.ContentType)
		if storeErr != nil {
			return fmt.Errorf("store artifact content: %w", storeErr)
		}
		v.URI = uri
		v.SizeBytes = sizeBytes
		v.Checksum = storedChecksum
		meta["stored_in"] = "minio"
	} else {
		if v.URI == "" {
			v.URI = fmt.Sprintf("inline://%s/%s", art.ID, filepath.ToSlash(sourceName))
		}
		if v.SizeBytes == 0 {
			v.SizeBytes = int64(len(v.Content))
		}
		if v.Checksum == "" {
			v.Checksum = checksum
		}
		meta["stored_in"] = "inline"
	}

	if v.ContentType == "" {
		v.ContentType = "application/octet-stream"
	}
	if v.Checksum == "" {
		v.Checksum = checksum
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal artifact version metadata: %w", err)
	}
	v.Metadata = model.JSON(metaBytes)
	return nil
}

func artifactVersionMetadata(raw model.JSON) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	meta := map[string]any{}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, err
	}
	return meta, nil
}

func deriveArtifactSourceName(name, contentType string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		trimmed = "artifact"
	}
	if filepath.Ext(trimmed) != "" {
		return trimmed
	}
	switch contentType {
	case "text/markdown":
		return trimmed + ".md"
	case "text/x-go":
		return trimmed + ".go"
	case "text/typescript":
		return trimmed + ".tsx"
	case "application/json":
		return trimmed + ".json"
	case "text/plain":
		return trimmed + ".txt"
	default:
		return trimmed
	}
}

func strOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

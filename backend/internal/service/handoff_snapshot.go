package service

import (
	"context"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

type HandoffSnapshotService struct {
	repo  HandoffSnapshotRepository
	audit *AuditService
}

func NewHandoffSnapshotService(repo HandoffSnapshotRepository, audit *AuditService) *HandoffSnapshotService {
	return &HandoffSnapshotService{repo: repo, audit: audit}
}

func (s *HandoffSnapshotService) Create(ctx context.Context, snap *model.HandoffSnapshot) error {
	if snap.TaskID == "" {
		return fmt.Errorf("task_id is required")
	}
	if snap.AgentID == "" {
		return fmt.Errorf("agent_id is required")
	}
	s.ensureDefaults(snap)
	if err := s.repo.Create(ctx, snap); err != nil {
		return fmt.Errorf("create handoff_snapshot: %w", err)
	}
	s.audit.LogEvent(ctx, "user", "", "handoff_snapshot.created",
		fmt.Sprintf("交接快照已创建（Agent %s）", snap.AgentID),
		"handoff_snapshot", snap.ID, nil, snap)
	return nil
}

func (s *HandoffSnapshotService) GetByID(ctx context.Context, id string) (*model.HandoffSnapshot, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *HandoffSnapshotService) List(ctx context.Context, p repository.HandoffListParams) ([]*model.HandoffSnapshot, int, error) {
	return s.repo.List(ctx, p)
}

func (s *HandoffSnapshotService) GetLatestByTaskID(ctx context.Context, taskID string) (*model.HandoffSnapshot, error) {
	return s.repo.GetLatestByTaskID(ctx, taskID)
}

func (s *HandoffSnapshotService) ensureDefaults(snap *model.HandoffSnapshot) {
	if len(snap.CompletedItems) == 0 {
		snap.CompletedItems = model.JSON("[]")
	}
	if len(snap.PendingItems) == 0 {
		snap.PendingItems = model.JSON("[]")
	}
	if len(snap.Risks) == 0 {
		snap.Risks = model.JSON("[]")
	}
	if len(snap.NextSteps) == 0 {
		snap.NextSteps = model.JSON("[]")
	}
	if len(snap.ArtifactRefs) == 0 {
		snap.ArtifactRefs = model.JSON("[]")
	}
	if len(snap.Metadata) == 0 {
		snap.Metadata = model.JSON("{}")
	}
}

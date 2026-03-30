package service

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

// auditRepo defines the repository methods AuditService depends on.
type auditRepo interface {
	Create(ctx context.Context, a *model.AuditLog) error
	List(ctx context.Context, p repository.AuditListParams) ([]*model.AuditLog, int, error)
	ProjectTimeline(ctx context.Context, projectID string, limit, offset int) ([]*model.AuditLog, int, error)
	TaskTimeline(ctx context.Context, taskID string, limit, offset int) ([]*model.AuditLog, int, error)
}

type AuditService struct {
	repo   auditRepo
	logger *slog.Logger
}

func NewAuditService(repo *repository.AuditRepo, logger *slog.Logger) *AuditService {
	return &AuditService{repo: repo, logger: logger}
}

func (s *AuditService) Log(ctx context.Context, entry *model.AuditLog) {
	if err := s.repo.Create(ctx, entry); err != nil {
		s.logger.Error("audit log write failed", "error", err, "event", entry.EventType)
	}
}

func (s *AuditService) LogEvent(ctx context.Context, actorType, actorID, eventType, summary, targetType, targetID string, before, after interface{}) {
	var bs, as *model.JSON
	if before != nil {
		b, _ := json.Marshal(before)
		j := model.JSON(b)
		bs = &j
	}
	if after != nil {
		a, _ := json.Marshal(after)
		j := model.JSON(a)
		as = &j
	}
	s.Log(ctx, &model.AuditLog{
		ActorType:    actorType,
		ActorID:      actorID,
		EventType:    eventType,
		EventSummary: summary,
		TargetType:   targetType,
		TargetID:     targetID,
		BeforeState:  bs,
		AfterState:   as,
		Metadata:     model.JSON("{}"),
	})
}

func (s *AuditService) List(ctx context.Context, p repository.AuditListParams) ([]*model.AuditLog, int, error) {
	return s.repo.List(ctx, p)
}

func (s *AuditService) ProjectTimeline(ctx context.Context, projectID string, limit, offset int) ([]*model.AuditLog, int, error) {
	return s.repo.ProjectTimeline(ctx, projectID, limit, offset)
}

func (s *AuditService) TaskTimeline(ctx context.Context, taskID string, limit, offset int) ([]*model.AuditLog, int, error) {
	return s.repo.TaskTimeline(ctx, taskID, limit, offset)
}

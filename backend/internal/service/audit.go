package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

type AuditService struct {
	repo   AuditRepository
	logger *slog.Logger
	hub    *EventHub
}

func NewAuditService(repo AuditRepository, logger *slog.Logger) *AuditService {
	return &AuditService{repo: repo, logger: logger}
}

// SetEventHub attaches an EventHub for real-time event broadcasting.
func (s *AuditService) SetEventHub(hub *EventHub) {
	s.hub = hub
}

func (s *AuditService) Log(ctx context.Context, entry *model.AuditLog) {
	if err := s.repo.Create(ctx, entry); err != nil {
		s.logger.Error("audit log write failed", "error", err, "event", entry.EventType)
	}
	// Publish to SSE subscribers
	if s.hub != nil {
		s.publishEvent(entry)
	}
}

// publishEvent converts an audit log entry to an SSE event and publishes it.
func (s *AuditService) publishEvent(entry *model.AuditLog) {
	topic := topicFromEventType(entry.EventType)
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	targetID := entry.TargetID
	projectID := ""
	if topic == "workflow" {
		targetID, projectID = workflowEventIDs(entry)
	}
	s.hub.Publish(&Event{
		ID:        entry.ID,
		Topic:     topic,
		EventType: entry.EventType,
		TargetID:  targetID,
		ProjectID: projectID,
		Data:      data,
		Timestamp: time.Now(),
	})
}

func workflowEventIDs(entry *model.AuditLog) (string, string) {
	targetID := entry.TargetID
	projectID := ""

	if entry.TargetType == "project" {
		projectID = entry.TargetID
	}

	if entry.AfterState == nil {
		return targetID, projectID
	}

	var after struct {
		RunID     string `json:"run_id"`
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal([]byte(*entry.AfterState), &after); err != nil {
		return targetID, projectID
	}

	if after.RunID != "" {
		targetID = after.RunID
	}
	if projectID == "" && after.ProjectID != "" {
		projectID = after.ProjectID
	}

	return targetID, projectID
}

// topicFromEventType maps audit event types to SSE topics.
func topicFromEventType(eventType string) string {
	switch {
	case strings.HasPrefix(eventType, "workflow."):
		return "workflow"
	case strings.HasPrefix(eventType, "approval"):
		return "approval"
	case strings.HasPrefix(eventType, "tool_call"):
		return "tool_call"
	default:
		return "audit"
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

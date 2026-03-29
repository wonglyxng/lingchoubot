package service

import (
	"context"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

type ToolCallService struct {
	repo  *repository.ToolCallRepo
	audit *AuditService
}

func NewToolCallService(repo *repository.ToolCallRepo, audit *AuditService) *ToolCallService {
	return &ToolCallService{repo: repo, audit: audit}
}

func (s *ToolCallService) Create(ctx context.Context, tc *model.ToolCall) error {
	if tc.AgentID == "" {
		return fmt.Errorf("agent_id is required")
	}
	if tc.ToolName == "" {
		return fmt.Errorf("tool_name is required")
	}
	if tc.Status == "" {
		tc.Status = model.ToolCallStatusPending
	}
	if len(tc.Input) == 0 {
		tc.Input = model.JSON("{}")
	}
	if len(tc.Output) == 0 {
		tc.Output = model.JSON("{}")
	}
	if len(tc.Metadata) == 0 {
		tc.Metadata = model.JSON("{}")
	}
	return s.repo.Create(ctx, tc)
}

func (s *ToolCallService) GetByID(ctx context.Context, id string) (*model.ToolCall, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ToolCallService) List(ctx context.Context, p repository.ToolCallListParams) ([]*model.ToolCall, int, error) {
	return s.repo.List(ctx, p)
}

func (s *ToolCallService) Complete(ctx context.Context, id string, status model.ToolCallStatus, output model.JSON, errMsg string, durationMs int) error {
	return s.repo.Complete(ctx, id, status, output, errMsg, durationMs)
}

package service

import (
	"context"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

type TaskAssignmentService struct {
	repo  TaskAssignmentRepository
	audit *AuditService
}

func NewTaskAssignmentService(repo TaskAssignmentRepository, audit *AuditService) *TaskAssignmentService {
	return &TaskAssignmentService{repo: repo, audit: audit}
}

func (s *TaskAssignmentService) Create(ctx context.Context, a *model.TaskAssignment) error {
	if a.TaskID == "" {
		return fmt.Errorf("task_id is required")
	}
	if a.AgentID == "" {
		return fmt.Errorf("agent_id is required")
	}
	if a.Role == "" {
		a.Role = model.AssignmentRoleExecutor
	}
	if a.Status == "" {
		a.Status = model.AssignmentStatusActive
	}
	if len(a.Metadata) == 0 {
		a.Metadata = model.JSON("{}")
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return fmt.Errorf("create task_assignment: %w", err)
	}
	s.audit.LogEvent(ctx, "user", "", "task_assignment.created",
		fmt.Sprintf("任务已分派给 Agent %s（角色: %s）", a.AgentID, a.Role),
		"task_assignment", a.ID, nil, a)
	return nil
}

func (s *TaskAssignmentService) GetByID(ctx context.Context, id string) (*model.TaskAssignment, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *TaskAssignmentService) List(ctx context.Context, p repository.AssignmentListParams) ([]*model.TaskAssignment, int, error) {
	return s.repo.List(ctx, p)
}

func (s *TaskAssignmentService) UpdateStatus(ctx context.Context, id string, status model.AssignmentStatus) error {
	old, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if old == nil {
		return fmt.Errorf("task_assignment not found")
	}
	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return err
	}
	s.audit.LogEvent(ctx, "user", "", "task_assignment.status_changed",
		fmt.Sprintf("分派状态变更: %s -> %s", old.Status, status),
		"task_assignment", id,
		map[string]string{"status": string(old.Status)},
		map[string]string{"status": string(status)},
	)
	return nil
}

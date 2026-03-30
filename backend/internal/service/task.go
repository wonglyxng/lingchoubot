package service

import (
	"context"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

// taskRepo defines the repository methods TaskService depends on.
type taskRepo interface {
	Create(ctx context.Context, t *model.Task) error
	GetByID(ctx context.Context, id string) (*model.Task, error)
	List(ctx context.Context, p repository.TaskListParams) ([]*model.Task, int, error)
	Update(ctx context.Context, t *model.Task) error
	UpdateStatus(ctx context.Context, id string, status model.TaskStatus) error
	Delete(ctx context.Context, id string) error
}

type TaskService struct {
	repo  taskRepo
	audit *AuditService
}

func NewTaskService(repo *repository.TaskRepo, audit *AuditService) *TaskService {
	return &TaskService{repo: repo, audit: audit}
}

func (s *TaskService) Create(ctx context.Context, t *model.Task) error {
	if t.ProjectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if t.Title == "" {
		return fmt.Errorf("task title is required")
	}
	if t.Status == "" {
		t.Status = model.TaskStatusPending
	}
	if t.ExecutionDomain == "" {
		t.ExecutionDomain = model.ExecDomainGeneral
	}
	if len(t.InputContext) == 0 {
		t.InputContext = model.JSON("{}")
	}
	if len(t.OutputSummary) == 0 {
		t.OutputSummary = model.JSON("{}")
	}
	if len(t.Metadata) == 0 {
		t.Metadata = model.JSON("{}")
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	s.audit.LogEvent(ctx, "user", "", "task.created",
		fmt.Sprintf("任务「%s」已创建", t.Title),
		"task", t.ID, nil, t)
	return nil
}

func (s *TaskService) GetByID(ctx context.Context, id string) (*model.Task, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *TaskService) List(ctx context.Context, p repository.TaskListParams) ([]*model.Task, int, error) {
	return s.repo.List(ctx, p)
}

func (s *TaskService) Update(ctx context.Context, t *model.Task) error {
	old, err := s.repo.GetByID(ctx, t.ID)
	if err != nil {
		return err
	}
	if old == nil {
		return fmt.Errorf("task not found")
	}
	if old.Status != t.Status {
		if !old.Status.CanTransitionTo(t.Status) {
			return fmt.Errorf("invalid status transition: %s -> %s", old.Status, t.Status)
		}
	}
	if err := s.repo.Update(ctx, t); err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	s.audit.LogEvent(ctx, "user", "", "task.updated",
		fmt.Sprintf("任务「%s」已更新", t.Title),
		"task", t.ID, old, t)
	return nil
}

// TransitionStatus validates and applies a status change.
func (s *TaskService) TransitionStatus(ctx context.Context, id string, newStatus model.TaskStatus) error {
	old, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if old == nil {
		return fmt.Errorf("task not found")
	}
	if !old.Status.CanTransitionTo(newStatus) {
		return fmt.Errorf("invalid status transition: %s -> %s", old.Status, newStatus)
	}
	if err := s.repo.UpdateStatus(ctx, id, newStatus); err != nil {
		return err
	}
	s.audit.LogEvent(ctx, "user", "", "task.status_changed",
		fmt.Sprintf("任务「%s」状态变更: %s -> %s", old.Title, old.Status, newStatus),
		"task", id,
		map[string]string{"status": string(old.Status)},
		map[string]string{"status": string(newStatus)},
	)
	return nil
}

func (s *TaskService) Delete(ctx context.Context, id string) error {
	old, _ := s.repo.GetByID(ctx, id)
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.audit.LogEvent(ctx, "user", "", "task.deleted",
		"任务已删除", "task", id, old, nil)
	return nil
}

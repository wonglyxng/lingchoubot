package service

import (
	"context"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

type TaskContractService struct {
	repo  *repository.TaskContractRepo
	audit *AuditService
}

func NewTaskContractService(repo *repository.TaskContractRepo, audit *AuditService) *TaskContractService {
	return &TaskContractService{repo: repo, audit: audit}
}

// Create creates a new contract version for a task.
// Version is auto-assigned as the next available version number.
func (s *TaskContractService) Create(ctx context.Context, c *model.TaskContract) error {
	if c.TaskID == "" {
		return fmt.Errorf("task_id is required")
	}
	if c.Scope == "" {
		return fmt.Errorf("scope is required")
	}
	nextVer, err := s.repo.NextVersion(ctx, c.TaskID)
	if err != nil {
		return fmt.Errorf("resolve version: %w", err)
	}
	c.Version = nextVer
	s.ensureDefaults(c)

	if err := s.repo.Create(ctx, c); err != nil {
		return fmt.Errorf("create task_contract: %w", err)
	}
	s.audit.LogEvent(ctx, "user", "", "task_contract.created",
		fmt.Sprintf("任务契约 v%d 已创建", c.Version),
		"task_contract", c.ID, nil, c)
	return nil
}

func (s *TaskContractService) GetByID(ctx context.Context, id string) (*model.TaskContract, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *TaskContractService) GetLatestByTaskID(ctx context.Context, taskID string) (*model.TaskContract, error) {
	return s.repo.GetLatestByTaskID(ctx, taskID)
}

func (s *TaskContractService) ListByTaskID(ctx context.Context, taskID string) ([]*model.TaskContract, error) {
	return s.repo.ListByTaskID(ctx, taskID)
}

func (s *TaskContractService) Update(ctx context.Context, c *model.TaskContract) error {
	old, err := s.repo.GetByID(ctx, c.ID)
	if err != nil {
		return err
	}
	if old == nil {
		return fmt.Errorf("task_contract not found")
	}
	if err := s.repo.Update(ctx, c); err != nil {
		return fmt.Errorf("update task_contract: %w", err)
	}
	s.audit.LogEvent(ctx, "user", "", "task_contract.updated",
		fmt.Sprintf("任务契约 v%d 已更新", c.Version),
		"task_contract", c.ID, old, c)
	return nil
}

func (s *TaskContractService) ensureDefaults(c *model.TaskContract) {
	if len(c.NonGoals) == 0 {
		c.NonGoals = model.JSON("[]")
	}
	if len(c.DoneDefinition) == 0 {
		c.DoneDefinition = model.JSON("[]")
	}
	if len(c.VerificationPlan) == 0 {
		c.VerificationPlan = model.JSON("[]")
	}
	if len(c.AcceptanceCriteria) == 0 {
		c.AcceptanceCriteria = model.JSON("[]")
	}
	if len(c.ToolPermissions) == 0 {
		c.ToolPermissions = model.JSON("[]")
	}
	if len(c.EscalationPolicy) == 0 {
		c.EscalationPolicy = model.JSON("{}")
	}
	if len(c.Metadata) == 0 {
		c.Metadata = model.JSON("{}")
	}
}

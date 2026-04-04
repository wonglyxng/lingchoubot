package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

type WorkflowService struct {
	runRepo  WorkflowRunRepository
	stepRepo WorkflowStepRepository
	audit    *AuditService
}

func NewWorkflowService(runRepo WorkflowRunRepository, stepRepo WorkflowStepRepository, audit *AuditService) *WorkflowService {
	return &WorkflowService{runRepo: runRepo, stepRepo: stepRepo, audit: audit}
}

// CreateRun persists a new workflow run with status "running".
func (s *WorkflowService) CreateRun(ctx context.Context, projectID string) (*model.WorkflowRun, error) {
	run := &model.WorkflowRun{
		ProjectID: projectID,
		Status:    model.WorkflowRunRunning,
		Metadata:  model.JSON("{}"),
		StartedAt: time.Now(),
	}
	if err := s.runRepo.Create(ctx, run); err != nil {
		return nil, fmt.Errorf("create workflow run: %w", err)
	}
	return run, nil
}

// GetRun loads a run and its steps.
func (s *WorkflowService) GetRun(ctx context.Context, id string) (*model.WorkflowRun, error) {
	run, err := s.runRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, nil
	}
	steps, err := s.stepRepo.ListByRunID(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	run.Steps = steps
	return run, nil
}

// ListRuns returns paginated workflow runs (without steps).
func (s *WorkflowService) ListRuns(ctx context.Context, p repository.WorkflowRunListParams) ([]*model.WorkflowRun, int, error) {
	return s.runRepo.List(ctx, p)
}

// CompleteRun marks a run as completed with a summary.
func (s *WorkflowService) CompleteRun(ctx context.Context, run *model.WorkflowRun, summary string) error {
	now := time.Now()
	run.Status = model.WorkflowRunCompleted
	run.Summary = summary
	run.Error = ""
	run.Metadata = clearWorkflowManualIntervention(run.Metadata)
	run.CompletedAt = &now
	return s.runRepo.UpdateStatus(ctx, run)
}

// WaitForApproval marks a run as waiting for human approval before it can continue.
func (s *WorkflowService) WaitForApproval(ctx context.Context, run *model.WorkflowRun, summary string) error {
	run.Status = model.WorkflowRunWaitingApproval
	run.Summary = summary
	run.Error = ""
	run.Metadata = clearWorkflowManualIntervention(run.Metadata)
	run.CompletedAt = nil
	return s.runRepo.UpdateStatus(ctx, run)
}

// WaitForManualIntervention marks a run as waiting for human intervention after a recoverable LLM failure.
func (s *WorkflowService) WaitForManualIntervention(ctx context.Context, run *model.WorkflowRun, summary, errMsg string, intervention *model.WorkflowManualIntervention) error {
	run.Status = model.WorkflowRunWaitingManual
	run.Summary = summary
	run.Error = errMsg
	metadata, err := setWorkflowManualIntervention(run.Metadata, intervention)
	if err != nil {
		return fmt.Errorf("encode manual intervention metadata: %w", err)
	}
	run.Metadata = metadata
	run.CompletedAt = nil
	return s.runRepo.UpdateStatus(ctx, run)
}

// ResumeRun marks a waiting run as running again so execution can continue.
func (s *WorkflowService) ResumeRun(ctx context.Context, run *model.WorkflowRun, summary string) error {
	run.Status = model.WorkflowRunRunning
	run.Summary = summary
	run.Error = ""
	run.Metadata = clearWorkflowManualIntervention(run.Metadata)
	run.CompletedAt = nil
	return s.runRepo.UpdateStatus(ctx, run)
}

// FailRun marks a run as failed with an error message.
func (s *WorkflowService) FailRun(ctx context.Context, run *model.WorkflowRun, errMsg string) error {
	now := time.Now()
	run.Status = model.WorkflowRunFailed
	run.Error = errMsg
	run.Metadata = clearWorkflowManualIntervention(run.Metadata)
	run.CompletedAt = &now
	return s.runRepo.UpdateStatus(ctx, run)
}

// CancelRun marks a run as cancelled.
func (s *WorkflowService) CancelRun(ctx context.Context, run *model.WorkflowRun) error {
	now := time.Now()
	run.Status = model.WorkflowRunCancelled
	run.Error = "cancelled by user"
	run.Metadata = clearWorkflowManualIntervention(run.Metadata)
	run.CompletedAt = &now
	return s.runRepo.UpdateStatus(ctx, run)
}

func setWorkflowManualIntervention(raw model.JSON, intervention *model.WorkflowManualIntervention) (model.JSON, error) {
	metadata, err := workflowRunMetadataObject(raw)
	if err != nil {
		return nil, err
	}
	if intervention == nil {
		delete(metadata, "manual_intervention")
	} else {
		metadata["manual_intervention"] = intervention
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	return model.JSON(encoded), nil
}

func clearWorkflowManualIntervention(raw model.JSON) model.JSON {
	metadata, err := workflowRunMetadataObject(raw)
	if err != nil {
		return model.JSON("{}")
	}
	delete(metadata, "manual_intervention")
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return model.JSON("{}")
	}
	return model.JSON(encoded)
}

func workflowRunMetadataObject(raw model.JSON) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	parsed := map[string]any{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	if parsed == nil {
		return map[string]any{}, nil
	}
	return parsed, nil
}

// AddStep creates a new step in the "pending" state.
func (s *WorkflowService) AddStep(ctx context.Context, runID string, name, agentRole string, sortOrder int) (*model.WorkflowStep, error) {
	step := &model.WorkflowStep{
		RunID:     runID,
		Name:      name,
		AgentRole: agentRole,
		Status:    model.WorkflowStepPending,
		SortOrder: sortOrder,
	}
	if err := s.stepRepo.Create(ctx, step); err != nil {
		return nil, fmt.Errorf("create workflow step: %w", err)
	}
	return step, nil
}

// StartStep transitions a step to "running".
func (s *WorkflowService) StartStep(ctx context.Context, step *model.WorkflowStep) error {
	now := time.Now()
	step.StartedAt = &now
	step.Status = model.WorkflowStepRunning
	return s.stepRepo.UpdateStatus(ctx, step)
}

// CompleteStep transitions a step to "completed" with a summary.
func (s *WorkflowService) CompleteStep(ctx context.Context, step *model.WorkflowStep, summary string) error {
	now := time.Now()
	step.CompletedAt = &now
	step.Status = model.WorkflowStepCompleted
	step.Summary = summary
	return s.stepRepo.UpdateStatus(ctx, step)
}

// FailStep transitions a step to "failed" with an error.
func (s *WorkflowService) FailStep(ctx context.Context, step *model.WorkflowStep, errMsg string) error {
	now := time.Now()
	step.CompletedAt = &now
	step.Status = model.WorkflowStepFailed
	step.Error = errMsg
	return s.stepRepo.UpdateStatus(ctx, step)
}

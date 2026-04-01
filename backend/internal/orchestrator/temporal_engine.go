package orchestrator

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
	"github.com/lingchou/lingchoubot/backend/internal/service"
	"go.temporal.io/sdk/client"
)

// TemporalEngine implements WorkflowEngine using Temporal as the execution backend.
type TemporalEngine struct {
	client    client.Client
	taskQueue string
	services  *Services
	workflow  *service.WorkflowService
	logger    *slog.Logger
}

// NewTemporalEngine creates an engine that dispatches runs to Temporal.
func NewTemporalEngine(tc client.Client, taskQueue string, services *Services, workflow *service.WorkflowService, logger *slog.Logger) *TemporalEngine {
	return &TemporalEngine{
		client:    tc,
		taskQueue: taskQueue,
		services:  services,
		workflow:  workflow,
		logger:    logger,
	}
}

// RunAsync starts a Temporal workflow for the given project.
// Returns immediately with the created run record; the actual execution happens in a Temporal worker.
func (te *TemporalEngine) RunAsync(ctx context.Context, projectID string) (*model.WorkflowRun, error) {
	if _, err := validateWorkflowStartPreconditions(ctx, te.services, projectID); err != nil {
		return nil, err
	}

	run, err := te.workflow.CreateRun(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("create workflow run: %w", err)
	}

	workflowID := fmt.Sprintf("lingchou-run-%s", run.ID)
	opts := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: te.taskQueue,
	}

	input := ProjectWorkflowInput{
		RunID:     run.ID,
		ProjectID: projectID,
	}

	we, err := te.client.ExecuteWorkflow(ctx, opts, ProjectWorkflow, input)
	if err != nil {
		_ = te.workflow.FailRun(ctx, run, fmt.Sprintf("failed to start Temporal workflow: %s", err.Error()))
		return nil, fmt.Errorf("start Temporal workflow: %w", err)
	}

	te.logger.Info("Temporal workflow started",
		"workflow_id", we.GetID(),
		"run_id", we.GetRunID(),
		"db_run_id", run.ID,
		"project_id", projectID,
	)

	return run, nil
}

// GetRun loads a workflow run with its steps from the database.
func (te *TemporalEngine) GetRun(ctx context.Context, id string) (*model.WorkflowRun, error) {
	return te.workflow.GetRun(ctx, id)
}

// ListRuns returns paginated workflow runs from the database.
func (te *TemporalEngine) ListRuns(ctx context.Context, p repository.WorkflowRunListParams) ([]*model.WorkflowRun, int, error) {
	return te.workflow.ListRuns(ctx, p)
}

// CancelRun cancels a Temporal workflow and marks the run as cancelled in the database.
func (te *TemporalEngine) CancelRun(ctx context.Context, id string) error {
	run, err := te.workflow.GetRun(ctx, id)
	if err != nil {
		return fmt.Errorf("get run: %w", err)
	}
	if run == nil {
		return fmt.Errorf("run %s not found", id)
	}
	if run.Status != model.WorkflowRunRunning {
		return fmt.Errorf("run %s is not running (status=%s)", id, run.Status)
	}

	// Cancel the Temporal workflow
	workflowID := fmt.Sprintf("lingchou-run-%s", run.ID)
	if err := te.client.CancelWorkflow(ctx, workflowID, ""); err != nil {
		te.logger.Error("cancel Temporal workflow failed", "workflow_id", workflowID, "error", err)
		// Still mark as cancelled in our DB even if Temporal cancel fails
	}

	return te.workflow.CancelRun(ctx, run)
}

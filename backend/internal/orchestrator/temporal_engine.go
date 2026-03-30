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
	workflow  *service.WorkflowService
	logger    *slog.Logger
}

// NewTemporalEngine creates an engine that dispatches runs to Temporal.
func NewTemporalEngine(tc client.Client, taskQueue string, workflow *service.WorkflowService, logger *slog.Logger) *TemporalEngine {
	return &TemporalEngine{
		client:    tc,
		taskQueue: taskQueue,
		workflow:  workflow,
		logger:    logger,
	}
}

// RunAsync starts a Temporal workflow for the given project.
// Returns immediately with the created run record; the actual execution happens in a Temporal worker.
func (te *TemporalEngine) RunAsync(ctx context.Context, projectID string) (*model.WorkflowRun, error) {
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

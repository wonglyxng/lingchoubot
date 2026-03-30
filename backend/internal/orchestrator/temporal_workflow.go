package orchestrator

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ProjectWorkflowInput is the input for the Temporal workflow.
type ProjectWorkflowInput struct {
	RunID     string `json:"run_id"`
	ProjectID string `json:"project_id"`
}

// PMActivityResult carries the phase/task IDs created during the PM step.
type PMActivityResult struct {
	PhaseIDs []string `json:"phase_ids"`
}

// PhaseTasksResult lists tasks within a phase.
type PhaseTasksResult struct {
	TaskIDs []string `json:"task_ids"`
}

// StepResult is a generic result from an activity step.
type StepResult struct {
	Summary string `json:"summary"`
}

// ProjectWorkflow is the top-level Temporal workflow that orchestrates
// PM → Supervisor → Worker → Reviewer for a project.
func ProjectWorkflow(ctx workflow.Context, input ProjectWorkflowInput) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: PM decomposes the project
	var pmResult PMActivityResult
	err := workflow.ExecuteActivity(ctx, "ActivityPM", input).Get(ctx, &pmResult)
	if err != nil {
		// Mark run as failed
		_ = workflow.ExecuteActivity(ctx, "ActivityFailRun", FailRunInput{
			RunID: input.RunID,
			Error: fmt.Sprintf("PM activity failed: %s", err.Error()),
		}).Get(ctx, nil)
		return err
	}

	// Step 2: For each phase, get tasks and run the chain
	for _, phaseID := range pmResult.PhaseIDs {
		var tasksResult PhaseTasksResult
		err := workflow.ExecuteActivity(ctx, "ActivityListPhaseTasks", ListPhaseTasksInput{
			PhaseID: phaseID,
		}).Get(ctx, &tasksResult)
		if err != nil {
			continue
		}

		for _, taskID := range tasksResult.TaskIDs {
			chainInput := TaskChainInput{
				RunID:     input.RunID,
				ProjectID: input.ProjectID,
				PhaseID:   phaseID,
				TaskID:    taskID,
			}

			// Supervisor
			var stepResult StepResult
			_ = workflow.ExecuteActivity(ctx, "ActivitySupervisor", chainInput).Get(ctx, &stepResult)

			// Worker
			_ = workflow.ExecuteActivity(ctx, "ActivityWorker", chainInput).Get(ctx, &stepResult)

			// Reviewer
			_ = workflow.ExecuteActivity(ctx, "ActivityReviewer", chainInput).Get(ctx, &stepResult)
		}
	}

	// Step 3: Complete the run
	_ = workflow.ExecuteActivity(ctx, "ActivityCompleteRun", CompleteRunInput{
		RunID:     input.RunID,
		ProjectID: input.ProjectID,
	}).Get(ctx, nil)

	return nil
}

// --- Activity input types ---

type ListPhaseTasksInput struct {
	PhaseID string `json:"phase_id"`
}

type TaskChainInput struct {
	RunID     string `json:"run_id"`
	ProjectID string `json:"project_id"`
	PhaseID   string `json:"phase_id"`
	TaskID    string `json:"task_id"`
}

type FailRunInput struct {
	RunID string `json:"run_id"`
	Error string `json:"error"`
}

type CompleteRunInput struct {
	RunID     string `json:"run_id"`
	ProjectID string `json:"project_id"`
}

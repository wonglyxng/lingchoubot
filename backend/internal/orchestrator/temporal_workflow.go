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
	PhaseIDs  []string `json:"phase_ids"`
	StepCount int      `json:"step_count"` // running step counter for subsequent activities
}

// PhaseTasksResult lists tasks within a phase.
type PhaseTasksResult struct {
	TaskIDs []string `json:"task_ids"`
}

// StepResult is a generic result from an activity step.
type StepResult struct {
	Summary   string `json:"summary"`
	StepCount int    `json:"step_count"` // updated step counter
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

	stepCount := 0

	// Step 1: PM decomposes the project
	var pmResult PMActivityResult
	err := workflow.ExecuteActivity(ctx, "ActivityPM", input).Get(ctx, &pmResult)
	if err != nil {
		_ = workflow.ExecuteActivity(ctx, "ActivityFailRun", FailRunInput{
			RunID: input.RunID,
			Error: fmt.Sprintf("PM activity failed: %s", err.Error()),
		}).Get(ctx, nil)
		return err
	}
	stepCount = pmResult.StepCount

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
				RunID:      input.RunID,
				ProjectID:  input.ProjectID,
				PhaseID:    phaseID,
				TaskID:     taskID,
				SortOffset: stepCount,
			}

			// Supervisor
			var supResult StepResult
			err = workflow.ExecuteActivity(ctx, "ActivitySupervisor", chainInput).Get(ctx, &supResult)
			if err == nil {
				stepCount = supResult.StepCount
				chainInput.SortOffset = stepCount
			}

			// Worker → Reviewer with rework loop
			const maxReworkAttempts = 3
			for attempt := 0; attempt <= maxReworkAttempts; attempt++ {
				// Worker
				var workerResult StepResult
				chainInput.SortOffset = stepCount
				err = workflow.ExecuteActivity(ctx, "ActivityWorker", chainInput).Get(ctx, &workerResult)
				if err == nil {
					stepCount = workerResult.StepCount
					chainInput.SortOffset = stepCount
				}

				// Reviewer
				var reviewResult StepResult
				err = workflow.ExecuteActivity(ctx, "ActivityReviewer", chainInput).Get(ctx, &reviewResult)
				if err == nil {
					stepCount = reviewResult.StepCount
					chainInput.SortOffset = stepCount
				}

				// Check if rework needed
				var needsRework bool
				checkErr := workflow.ExecuteActivity(ctx, "ActivityCheckRework", CheckReworkInput{
					TaskID:  taskID,
					Attempt: attempt + 1,
				}).Get(ctx, &needsRework)
				if checkErr != nil || !needsRework {
					break
				}
			}
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
	RunID      string `json:"run_id"`
	ProjectID  string `json:"project_id"`
	PhaseID    string `json:"phase_id"`
	TaskID     string `json:"task_id"`
	SortOffset int    `json:"sort_offset"` // base sort_order for steps in this chain
}

type FailRunInput struct {
	RunID string `json:"run_id"`
	Error string `json:"error"`
}

type CompleteRunInput struct {
	RunID     string `json:"run_id"`
	ProjectID string `json:"project_id"`
}

type CheckReworkInput struct {
	TaskID  string `json:"task_id"`
	Attempt int    `json:"attempt"`
}

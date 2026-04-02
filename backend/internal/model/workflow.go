package model

import "time"

// WorkflowRunStatus represents the state of a workflow run.
type WorkflowRunStatus string

const (
	WorkflowRunPending         WorkflowRunStatus = "pending"
	WorkflowRunRunning         WorkflowRunStatus = "running"
	WorkflowRunWaitingApproval WorkflowRunStatus = "waiting_approval"
	WorkflowRunWaitingManual   WorkflowRunStatus = "waiting_manual_intervention"
	WorkflowRunCompleted       WorkflowRunStatus = "completed"
	WorkflowRunFailed          WorkflowRunStatus = "failed"
	WorkflowRunCancelled       WorkflowRunStatus = "cancelled"
)

// WorkflowStepStatus represents the state of a single step within a run.
type WorkflowStepStatus string

const (
	WorkflowStepPending   WorkflowStepStatus = "pending"
	WorkflowStepRunning   WorkflowStepStatus = "running"
	WorkflowStepCompleted WorkflowStepStatus = "completed"
	WorkflowStepFailed    WorkflowStepStatus = "failed"
	WorkflowStepSkipped   WorkflowStepStatus = "skipped"
)

// WorkflowRun is the persisted representation of an orchestration execution.
type WorkflowRun struct {
	ID          string            `json:"id"`
	ProjectID   string            `json:"project_id"`
	Status      WorkflowRunStatus `json:"status"`
	Summary     string            `json:"summary"`
	Error       string            `json:"error,omitempty"`
	StartedAt   time.Time         `json:"started_at"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`

	// Steps is populated when loading a run with its steps.
	Steps []*WorkflowStep `json:"steps,omitempty"`
}

// WorkflowStep is one unit of work within a workflow run.
type WorkflowStep struct {
	ID          string             `json:"id"`
	RunID       string             `json:"run_id"`
	Name        string             `json:"name"`
	AgentRole   string             `json:"agent_role"`
	AgentID     *string            `json:"agent_id,omitempty"`
	TaskID      *string            `json:"task_id,omitempty"`
	PhaseID     *string            `json:"phase_id,omitempty"`
	Status      WorkflowStepStatus `json:"status"`
	Summary     string             `json:"summary"`
	Error       string             `json:"error,omitempty"`
	SortOrder   int                `json:"sort_order"`
	StartedAt   *time.Time         `json:"started_at,omitempty"`
	CompletedAt *time.Time         `json:"completed_at,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
}

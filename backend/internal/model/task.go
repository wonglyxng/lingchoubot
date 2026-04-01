package model

import "time"

type TaskStatus string

// ExecutionDomain 任务执行域 — 标识任务归属的主管域
type ExecutionDomain string

const (
	ExecDomainGeneral     ExecutionDomain = "general"
	ExecDomainDevelopment ExecutionDomain = "development"
	ExecDomainQA          ExecutionDomain = "qa"
)

const (
	TaskStatusPending          TaskStatus = "pending"
	TaskStatusAssigned         TaskStatus = "assigned"
	TaskStatusInProgress       TaskStatus = "in_progress"
	TaskStatusInReview         TaskStatus = "in_review"
	TaskStatusPendingApproval  TaskStatus = "pending_approval"
	TaskStatusRevisionRequired TaskStatus = "revision_required"
	TaskStatusCompleted        TaskStatus = "completed"
	TaskStatusFailed           TaskStatus = "failed"
	TaskStatusCancelled        TaskStatus = "cancelled"
	TaskStatusBlocked          TaskStatus = "blocked"
)

var validTaskTransitions = map[TaskStatus][]TaskStatus{
	TaskStatusPending:          {TaskStatusAssigned, TaskStatusCancelled},
	TaskStatusAssigned:         {TaskStatusInProgress, TaskStatusCancelled},
	TaskStatusInProgress:       {TaskStatusInReview, TaskStatusCompleted, TaskStatusFailed, TaskStatusBlocked, TaskStatusCancelled},
	TaskStatusInReview:         {TaskStatusPendingApproval, TaskStatusRevisionRequired, TaskStatusFailed},
	TaskStatusPendingApproval:  {TaskStatusCompleted, TaskStatusRevisionRequired, TaskStatusFailed},
	TaskStatusRevisionRequired: {TaskStatusInProgress, TaskStatusCancelled},
	TaskStatusBlocked:          {TaskStatusInProgress, TaskStatusCancelled},
	TaskStatusFailed:           {TaskStatusPending, TaskStatusCancelled},
}

func (s TaskStatus) CanTransitionTo(target TaskStatus) bool {
	allowed, ok := validTaskTransitions[s]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == target {
			return true
		}
	}
	return false
}

type Task struct {
	ID                string          `json:"id"`
	ProjectID         string          `json:"project_id"`
	PhaseID           *string         `json:"phase_id,omitempty"`
	ParentTaskID      *string         `json:"parent_task_id,omitempty"`
	Title             string          `json:"title"`
	Description       string          `json:"description"`
	Status            TaskStatus      `json:"status"`
	Priority          int             `json:"priority"`
	AssigneeID        *string         `json:"assignee_id,omitempty"`
	ExecutionDomain   ExecutionDomain `json:"execution_domain"`
	OwnerSupervisorID *string         `json:"owner_supervisor_id,omitempty"`
	InputContext      JSON            `json:"input_context"`
	OutputSummary     JSON            `json:"output_summary"`
	Metadata          JSON            `json:"metadata"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

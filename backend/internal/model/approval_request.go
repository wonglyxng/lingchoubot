package model

import "time"

type ApprovalStatus string

type WorkflowResumeStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"

	WorkflowResumeStatusNotRequested WorkflowResumeStatus = "not_requested"
	WorkflowResumeStatusResumed      WorkflowResumeStatus = "resumed"
	WorkflowResumeStatusSkipped      WorkflowResumeStatus = "skipped"
	WorkflowResumeStatusWarning      WorkflowResumeStatus = "warning"
)

type ApprovalRequest struct {
	ID           string         `json:"id"`
	ProjectID    string         `json:"project_id"`
	TaskID       *string        `json:"task_id,omitempty"`
	ArtifactID   *string        `json:"artifact_id,omitempty"`
	RequestedBy  string         `json:"requested_by"`
	ApproverType string         `json:"approver_type"`
	ApproverID   string         `json:"approver_id"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Status       ApprovalStatus `json:"status"`
	DecisionNote string         `json:"decision_note"`
	DecidedAt    *time.Time     `json:"decided_at,omitempty"`
	Metadata     JSON           `json:"metadata"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type ApprovalDecisionResult struct {
	ID                    string               `json:"id"`
	Status                ApprovalStatus       `json:"status"`
	TaskStatus            *TaskStatus          `json:"task_status,omitempty"`
	WorkflowRunID         string               `json:"workflow_run_id,omitempty"`
	WorkflowResumeStatus  WorkflowResumeStatus `json:"workflow_resume_status"`
	WorkflowResumeMessage string               `json:"workflow_resume_message,omitempty"`
	Warnings              []string             `json:"warnings,omitempty"`
}

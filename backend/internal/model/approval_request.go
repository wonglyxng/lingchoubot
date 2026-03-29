package model

import "time"

type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
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

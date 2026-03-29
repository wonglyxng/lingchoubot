package model

import "time"

type AssignmentRole string

const (
	AssignmentRoleExecutor AssignmentRole = "executor"
	AssignmentRoleReviewer AssignmentRole = "reviewer"
)

type AssignmentStatus string

const (
	AssignmentStatusActive    AssignmentStatus = "active"
	AssignmentStatusCompleted AssignmentStatus = "completed"
	AssignmentStatusRevoked   AssignmentStatus = "revoked"
)

type TaskAssignment struct {
	ID          string           `json:"id"`
	TaskID      string           `json:"task_id"`
	AgentID     string           `json:"agent_id"`
	AssignedBy  *string          `json:"assigned_by,omitempty"`
	Role        AssignmentRole   `json:"role"`
	Status      AssignmentStatus `json:"status"`
	Note        string           `json:"note"`
	Metadata    JSON             `json:"metadata"`
	CreatedAt   time.Time        `json:"created_at"`
	CompletedAt *time.Time       `json:"completed_at,omitempty"`
}

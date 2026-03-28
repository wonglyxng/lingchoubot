package model

import "time"

type PhaseStatus string

const (
	PhaseStatusPending   PhaseStatus = "pending"
	PhaseStatusActive    PhaseStatus = "active"
	PhaseStatusCompleted PhaseStatus = "completed"
	PhaseStatusSkipped   PhaseStatus = "skipped"
)

type Phase struct {
	ID          string      `json:"id"`
	ProjectID   string      `json:"project_id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Status      PhaseStatus `json:"status"`
	SortOrder   int         `json:"sort_order"`
	Metadata    JSON        `json:"metadata"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

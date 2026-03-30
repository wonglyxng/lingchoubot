package model

import "time"

type ToolCallStatus string

const (
	ToolCallStatusPending   ToolCallStatus = "pending"
	ToolCallStatusRunning   ToolCallStatus = "running"
	ToolCallStatusSuccess   ToolCallStatus = "success"
	ToolCallStatusFailed    ToolCallStatus = "failed"
	ToolCallStatusDenied    ToolCallStatus = "denied"
	ToolCallStatusEscalated ToolCallStatus = "escalated"
)

type ToolCall struct {
	ID           string         `json:"id"`
	TaskID       *string        `json:"task_id,omitempty"`
	AgentID      string         `json:"agent_id"`
	ToolName     string         `json:"tool_name"`
	Action       string         `json:"action"`
	Input        JSON           `json:"input"`
	Output       JSON           `json:"output"`
	Status       ToolCallStatus `json:"status"`
	ErrorMessage string         `json:"error_message"`
	DeniedReason string         `json:"denied_reason"`
	DurationMs   int            `json:"duration_ms"`
	Metadata     JSON           `json:"metadata"`
	CreatedAt    time.Time      `json:"created_at"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
}

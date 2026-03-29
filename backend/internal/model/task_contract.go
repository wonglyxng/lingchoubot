package model

import "time"

type TaskContract struct {
	ID                 string    `json:"id"`
	TaskID             string    `json:"task_id"`
	Version            int       `json:"version"`
	Scope              string    `json:"scope"`
	NonGoals           JSON      `json:"non_goals"`
	DoneDefinition     JSON      `json:"done_definition"`
	VerificationPlan   JSON      `json:"verification_plan"`
	AcceptanceCriteria JSON      `json:"acceptance_criteria"`
	ToolPermissions    JSON      `json:"tool_permissions"`
	EscalationPolicy   JSON      `json:"escalation_policy"`
	Metadata           JSON      `json:"metadata"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

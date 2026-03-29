package model

import "time"

type HandoffSnapshot struct {
	ID             string    `json:"id"`
	TaskID         string    `json:"task_id"`
	AgentID        string    `json:"agent_id"`
	Summary        string    `json:"summary"`
	CompletedItems JSON      `json:"completed_items"`
	PendingItems   JSON      `json:"pending_items"`
	Risks          JSON      `json:"risks"`
	NextSteps      JSON      `json:"next_steps"`
	ArtifactRefs   JSON      `json:"artifact_refs"`
	Metadata       JSON      `json:"metadata"`
	CreatedAt      time.Time `json:"created_at"`
}

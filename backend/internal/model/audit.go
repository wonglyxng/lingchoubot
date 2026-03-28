package model

import "time"

type AuditLog struct {
	ID           string  `json:"id"`
	ActorType    string  `json:"actor_type"`
	ActorID      string  `json:"actor_id"`
	EventType    string  `json:"event_type"`
	EventSummary string  `json:"event_summary"`
	TargetType   string  `json:"target_type"`
	TargetID     string  `json:"target_id"`
	BeforeState  *JSON   `json:"before_state,omitempty"`
	AfterState   *JSON   `json:"after_state,omitempty"`
	Metadata     JSON    `json:"metadata"`
	CreatedAt    time.Time `json:"created_at"`
}

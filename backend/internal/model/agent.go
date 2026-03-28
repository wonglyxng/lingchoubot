package model

import "time"

type AgentRole string

const (
	AgentRolePM         AgentRole = "pm"
	AgentRoleSupervisor AgentRole = "supervisor"
	AgentRoleWorker     AgentRole = "worker"
	AgentRoleReviewer   AgentRole = "reviewer"
)

type AgentStatus string

const (
	AgentStatusActive   AgentStatus = "active"
	AgentStatusInactive AgentStatus = "inactive"
)

type Agent struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Role         AgentRole   `json:"role"`
	Description  string      `json:"description"`
	ReportsTo    *string     `json:"reports_to,omitempty"`
	Status       AgentStatus `json:"status"`
	Capabilities JSON        `json:"capabilities"`
	Metadata     JSON        `json:"metadata"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

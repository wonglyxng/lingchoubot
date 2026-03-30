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

type AgentType string

const (
	AgentTypeMock  AgentType = "mock"
	AgentTypeLLM   AgentType = "llm"
	AgentTypeHuman AgentType = "human"
)

type AgentSpecialization string

const (
	AgentSpecGeneral  AgentSpecialization = "general"
	AgentSpecBackend  AgentSpecialization = "backend"
	AgentSpecFrontend AgentSpecialization = "frontend"
	AgentSpecQA       AgentSpecialization = "qa"
	AgentSpecRelease  AgentSpecialization = "release"
	AgentSpecDevOps   AgentSpecialization = "devops"
	AgentSpecDesign   AgentSpecialization = "design"
)

type Agent struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Role           AgentRole           `json:"role"`
	AgentType      AgentType           `json:"agent_type"`
	Specialization AgentSpecialization `json:"specialization"`
	Description    string              `json:"description"`
	ReportsTo      *string             `json:"reports_to,omitempty"`
	Status         AgentStatus         `json:"status"`
	Capabilities   JSON                `json:"capabilities"`
	Metadata       JSON                `json:"metadata"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
}

// MatchesSpecialization returns true if this agent matches a requested specialization.
// A "general" agent matches any specialization request.
func (a *Agent) MatchesSpecialization(spec AgentSpecialization) bool {
	if a.Specialization == AgentSpecGeneral {
		return true
	}
	return a.Specialization == spec
}

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

// RoleCode 业务职责码 — 比 AgentRole 更细粒度，表达真实组织角色
type RoleCode string

const (
	RoleCodePMSupervisor          RoleCode = "PM_SUPERVISOR"
	RoleCodeDevelopmentSupervisor RoleCode = "DEVELOPMENT_SUPERVISOR"
	RoleCodeQASupervisor          RoleCode = "QA_SUPERVISOR"
	RoleCodeGeneralWorker         RoleCode = "GENERAL_WORKER"
	RoleCodeBackendDevWorker      RoleCode = "BACKEND_DEV_WORKER"
	RoleCodeFrontendDevWorker     RoleCode = "FRONTEND_DEV_WORKER"
	RoleCodeQAWorker              RoleCode = "QA_WORKER"
	RoleCodeReviewerWorker        RoleCode = "REVIEWER_WORKER"
)

// RiskLevel 风险级别
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

type Agent struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Role           AgentRole           `json:"role"`
	RoleCode       RoleCode            `json:"role_code"`
	AgentType      AgentType           `json:"agent_type"`
	Specialization AgentSpecialization `json:"specialization"`
	Description    string              `json:"description"`
	ReportsTo      *string             `json:"reports_to,omitempty"`
	Status         AgentStatus         `json:"status"`
	ManagedRoles   JSON                `json:"managed_roles"`
	AllowedTools   JSON                `json:"allowed_tools"`
	RiskLevel      RiskLevel           `json:"risk_level"`
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

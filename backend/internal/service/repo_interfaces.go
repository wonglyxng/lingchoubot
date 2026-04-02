package service

import (
	"context"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

// Repository interfaces for dependency injection and testing.
// Concrete repository types in the repository package satisfy these interfaces.

type ProjectRepository interface {
	Create(ctx context.Context, p *model.Project) error
	GetByID(ctx context.Context, id string) (*model.Project, error)
	List(ctx context.Context, limit, offset int) ([]*model.Project, int, error)
	Update(ctx context.Context, p *model.Project) error
	Delete(ctx context.Context, id string) error
}

type PhaseRepository interface {
	Create(ctx context.Context, p *model.Phase) error
	GetByID(ctx context.Context, id string) (*model.Phase, error)
	ListByProject(ctx context.Context, projectID string) ([]*model.Phase, error)
	Update(ctx context.Context, p *model.Phase) error
	Delete(ctx context.Context, id string) error
}

type AgentRepository interface {
	Create(ctx context.Context, a *model.Agent) error
	GetByID(ctx context.Context, id string) (*model.Agent, error)
	GetByRoleCode(ctx context.Context, roleCode model.RoleCode) (*model.Agent, error)
	List(ctx context.Context, limit, offset int) ([]*model.Agent, int, error)
	Update(ctx context.Context, a *model.Agent) error
	Delete(ctx context.Context, id string) error
	GetSubordinates(ctx context.Context, agentID string) ([]*model.Agent, error)
	GetOrgTree(ctx context.Context, rootID string) ([]*model.Agent, error)
	FindByRoleAndSpec(ctx context.Context, role model.AgentRole, spec model.AgentSpecialization) (*model.Agent, error)
	FindByRoleCode(ctx context.Context, roleCode model.RoleCode) (*model.Agent, error)
}

type TaskRepository interface {
	Create(ctx context.Context, t *model.Task) error
	GetByID(ctx context.Context, id string) (*model.Task, error)
	List(ctx context.Context, p repository.TaskListParams) ([]*model.Task, int, error)
	Update(ctx context.Context, t *model.Task) error
	UpdateStatus(ctx context.Context, id string, status model.TaskStatus) error
	Delete(ctx context.Context, id string) error
}

type TaskContractRepository interface {
	Create(ctx context.Context, c *model.TaskContract) error
	GetByID(ctx context.Context, id string) (*model.TaskContract, error)
	GetLatestByTaskID(ctx context.Context, taskID string) (*model.TaskContract, error)
	ListByTaskID(ctx context.Context, taskID string) ([]*model.TaskContract, error)
	NextVersion(ctx context.Context, taskID string) (int, error)
	Update(ctx context.Context, c *model.TaskContract) error
}

type TaskAssignmentRepository interface {
	Create(ctx context.Context, a *model.TaskAssignment) error
	GetByID(ctx context.Context, id string) (*model.TaskAssignment, error)
	List(ctx context.Context, p repository.AssignmentListParams) ([]*model.TaskAssignment, int, error)
	UpdateStatus(ctx context.Context, id string, status model.AssignmentStatus) error
}

type ArtifactRepository interface {
	Create(ctx context.Context, a *model.Artifact) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*model.Artifact, error)
	List(ctx context.Context, p repository.ArtifactListParams) ([]*model.Artifact, int, error)
}

type ArtifactVersionRepository interface {
	Create(ctx context.Context, v *model.ArtifactVersion) error
	GetByID(ctx context.Context, id string) (*model.ArtifactVersion, error)
	ListByArtifact(ctx context.Context, artifactID string) ([]*model.ArtifactVersion, error)
	NextVersion(ctx context.Context, artifactID string) (int, error)
}

type HandoffSnapshotRepository interface {
	Create(ctx context.Context, s *model.HandoffSnapshot) error
	GetByID(ctx context.Context, id string) (*model.HandoffSnapshot, error)
	List(ctx context.Context, p repository.HandoffListParams) ([]*model.HandoffSnapshot, int, error)
	GetLatestByTaskID(ctx context.Context, taskID string) (*model.HandoffSnapshot, error)
}

type ReviewReportRepository interface {
	Create(ctx context.Context, rr *model.ReviewReport) error
	GetByID(ctx context.Context, id string) (*model.ReviewReport, error)
	List(ctx context.Context, p repository.ReviewListParams) ([]*model.ReviewReport, int, error)
}

type ApprovalRepository interface {
	Create(ctx context.Context, a *model.ApprovalRequest) error
	GetByID(ctx context.Context, id string) (*model.ApprovalRequest, error)
	List(ctx context.Context, p repository.ApprovalListParams) ([]*model.ApprovalRequest, int, error)
	Decide(ctx context.Context, id string, status model.ApprovalStatus, note string) error
}

type AuditRepository interface {
	Create(ctx context.Context, a *model.AuditLog) error
	List(ctx context.Context, p repository.AuditListParams) ([]*model.AuditLog, int, error)
	ProjectTimeline(ctx context.Context, projectID string, limit, offset int) ([]*model.AuditLog, int, error)
	TaskTimeline(ctx context.Context, taskID string, limit, offset int) ([]*model.AuditLog, int, error)
}

type WorkflowRunRepository interface {
	Create(ctx context.Context, run *model.WorkflowRun) error
	GetByID(ctx context.Context, id string) (*model.WorkflowRun, error)
	UpdateStatus(ctx context.Context, run *model.WorkflowRun) error
	List(ctx context.Context, p repository.WorkflowRunListParams) ([]*model.WorkflowRun, int, error)
}

type WorkflowStepRepository interface {
	Create(ctx context.Context, step *model.WorkflowStep) error
	UpdateStatus(ctx context.Context, step *model.WorkflowStep) error
	ListByRunID(ctx context.Context, runID string) ([]*model.WorkflowStep, error)
}

type ToolCallRepository interface {
	Create(ctx context.Context, tc *model.ToolCall) error
	GetByID(ctx context.Context, id string) (*model.ToolCall, error)
	List(ctx context.Context, p repository.ToolCallListParams) ([]*model.ToolCall, int, error)
	Complete(ctx context.Context, id string, status model.ToolCallStatus, output model.JSON, errMsg string, durationMs int) error
	UpdateDenied(ctx context.Context, id string, reason string) error
}

type LLMProviderRepository interface {
	Create(ctx context.Context, p *model.LLMProvider) error
	GetByID(ctx context.Context, id string) (*model.LLMProvider, error)
	GetByKey(ctx context.Context, key string) (*model.LLMProvider, error)
	List(ctx context.Context, enabledOnly bool) ([]*model.LLMProvider, error)
	Update(ctx context.Context, p *model.LLMProvider) error
	Delete(ctx context.Context, id string) error
	CreateModel(ctx context.Context, m *model.LLMModel) error
	GetModelByID(ctx context.Context, id string) (*model.LLMModel, error)
	ListModelsByProvider(ctx context.Context, providerID string) ([]*model.LLMModel, error)
	ListAllModels(ctx context.Context) ([]*model.LLMModel, error)
	UpdateModel(ctx context.Context, m *model.LLMModel) error
	DeleteModel(ctx context.Context, id string) error
}

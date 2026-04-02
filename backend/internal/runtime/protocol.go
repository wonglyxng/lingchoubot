package runtime

// AgentRunner is the interface that all agent implementations must satisfy.
// Mock agents implement this for MVP; future agents will wrap LLM calls.
type AgentRunner interface {
	Execute(input *AgentTaskInput) (*AgentTaskOutput, error)
	Role() string
}

// SpecializedRunner extends AgentRunner with a Specialization method.
// Runners that implement this interface can be routed by specialization.
type SpecializedRunner interface {
	AgentRunner
	Specialization() string
}

// AgentTaskInput packages everything an agent needs to execute its work.
type AgentTaskInput struct {
	RunID       string          `json:"run_id"`
	AgentID     string          `json:"agent_id"`
	AgentRole   string          `json:"agent_role"`
	AgentLLM    *AgentLLMConfig `json:"agent_llm,omitempty"`
	Instruction string          `json:"instruction"`
	Project     *ProjectCtx     `json:"project,omitempty"`
	Phase       *PhaseCtx       `json:"phase,omitempty"`
	Task        *TaskCtx        `json:"task,omitempty"`
	Contract    *ContractCtx    `json:"contract,omitempty"`
	Artifacts   []ArtifactCtx   `json:"artifacts,omitempty"`
	Parameters  map[string]any  `json:"parameters,omitempty"`
}

type AgentLLMConfig struct {
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
}

type ProjectCtx struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type PhaseCtx struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

type TaskCtx struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	PhaseID     string `json:"phase_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
}

type ContractCtx struct {
	ID                 string `json:"id"`
	Scope              string `json:"scope"`
	DoneDefinition     string `json:"done_definition"`
	VerificationPlan   string `json:"verification_plan"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
}

type ArtifactCtx struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ArtifactType string `json:"artifact_type"`
	VersionURI   string `json:"version_uri"`
}

// AgentTaskOutput is the structured result returned by an agent after execution.
// The orchestrator interprets these actions and calls the appropriate services.
type AgentTaskOutput struct {
	Status      OutputStatus       `json:"status"`
	Summary     string             `json:"summary"`
	Error       string             `json:"error,omitempty"`
	Phases      []PhaseAction      `json:"phases,omitempty"`
	Tasks       []TaskAction       `json:"tasks,omitempty"`
	Contracts   []ContractAction   `json:"contracts,omitempty"`
	Assignments []AssignmentAction `json:"assignments,omitempty"`
	Artifacts   []ArtifactAction   `json:"artifacts,omitempty"`
	Handoffs    []HandoffAction    `json:"handoffs,omitempty"`
	Reviews     []ReviewAction     `json:"reviews,omitempty"`
	Transitions []TransitionAction `json:"transitions,omitempty"`
}

type OutputStatus string

const (
	OutputStatusSuccess       OutputStatus = "success"
	OutputStatusFailed        OutputStatus = "failed"
	OutputStatusNeedsReview   OutputStatus = "needs_review"
	OutputStatusNeedsRevision OutputStatus = "needs_revision"
)

type PhaseAction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

type TaskAction struct {
	PhaseName   string `json:"phase_name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
}

type ContractAction struct {
	TaskTitle          string   `json:"task_title"`
	Scope              string   `json:"scope"`
	NonGoals           []string `json:"non_goals"`
	DoneDefinition     []string `json:"done_definition"`
	VerificationSteps  []string `json:"verification_steps"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
}

type AssignmentAction struct {
	TaskTitle string `json:"task_title"`
	AgentRole string `json:"agent_role"`
	Role      string `json:"role"`
	Note      string `json:"note"`
}

type ArtifactAction struct {
	Name         string         `json:"name"`
	ArtifactType string         `json:"artifact_type"`
	Description  string         `json:"description"`
	URI          string         `json:"uri"`
	ContentType  string         `json:"content_type"`
	SizeBytes    int64          `json:"size_bytes"`
	Content      string         `json:"content"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type HandoffAction struct {
	Summary        string   `json:"summary"`
	CompletedItems []string `json:"completed_items"`
	PendingItems   []string `json:"pending_items"`
	Risks          []string `json:"risks"`
	NextSteps      []string `json:"next_steps"`
}

type ReviewAction struct {
	Verdict         string   `json:"verdict"`
	Summary         string   `json:"summary"`
	Findings        []string `json:"findings"`
	Recommendations []string `json:"recommendations"`
}

type TransitionAction struct {
	TaskTitle string `json:"task_title"`
	NewStatus string `json:"new_status"`
}

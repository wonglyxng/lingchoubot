package orchestrator

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
)

type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
)

// WorkflowRun tracks the state of a single orchestration execution.
type WorkflowRun struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	ProjectName string    `json:"project_name"`
	Status      RunStatus `json:"status"`
	Steps       []*Step   `json:"steps"`
	Summary     string    `json:"summary"`
	Error       string    `json:"error,omitempty"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// Step represents one unit of work within a workflow run.
type Step struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	AgentRole   string     `json:"agent_role"`
	AgentID     string     `json:"agent_id,omitempty"`
	TaskID      string     `json:"task_id,omitempty"`
	PhaseID     string     `json:"phase_id,omitempty"`
	Status      StepStatus `json:"status"`
	Summary     string     `json:"summary"`
	Error       string     `json:"error,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

func (r *WorkflowRun) AddStep(name, agentRole string) *Step {
	s := &Step{
		ID:        newID(),
		Name:      name,
		AgentRole: agentRole,
		Status:    StepStatusPending,
	}
	r.Steps = append(r.Steps, s)
	return s
}

func (s *Step) Start() {
	now := time.Now()
	s.StartedAt = &now
	s.Status = StepStatusRunning
}

func (s *Step) Complete(summary string) {
	now := time.Now()
	s.CompletedAt = &now
	s.Status = StepStatusCompleted
	s.Summary = summary
}

func (s *Step) Fail(errMsg string) {
	now := time.Now()
	s.CompletedAt = &now
	s.Status = StepStatusFailed
	s.Error = errMsg
}

// RunStore provides in-memory storage for workflow runs.
// Sufficient for MVP; future versions can persist to database.
type RunStore struct {
	mu   sync.RWMutex
	runs map[string]*WorkflowRun
}

func NewRunStore() *RunStore {
	return &RunStore{runs: make(map[string]*WorkflowRun)}
}

func (s *RunStore) Save(run *WorkflowRun) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[run.ID] = run
}

func (s *RunStore) Get(id string) *WorkflowRun {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.runs[id]
}

func (s *RunStore) List() []*WorkflowRun {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*WorkflowRun, 0, len(s.runs))
	for _, r := range s.runs {
		list = append(list, r)
	}
	return list
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

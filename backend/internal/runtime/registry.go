package runtime

import (
	"fmt"
	"sync"
)

// Registry manages available AgentRunner implementations keyed by role.
type Registry struct {
	mu      sync.RWMutex
	runners map[string]AgentRunner
}

func NewRegistry() *Registry {
	return &Registry{runners: make(map[string]AgentRunner)}
}

func (r *Registry) Register(role string, runner AgentRunner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runners[role] = runner
}

func (r *Registry) Get(role string) (AgentRunner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	runner, ok := r.runners[role]
	if !ok {
		return nil, fmt.Errorf("no agent runner registered for role %q", role)
	}
	return runner, nil
}

func (r *Registry) Roles() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	roles := make([]string, 0, len(r.runners))
	for role := range r.runners {
		roles = append(roles, role)
	}
	return roles
}

// RegisterDefaults registers all mock agent runners.
func (r *Registry) RegisterDefaults() {
	r.Register("pm", &MockPMAgent{})
	r.Register("supervisor", &MockSupervisorAgent{})
	r.Register("worker", &MockWorkerAgent{})
	r.Register("reviewer", &MockReviewerAgent{})
}

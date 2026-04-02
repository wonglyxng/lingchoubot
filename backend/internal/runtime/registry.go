package runtime

import (
	"fmt"
	"sync"
)

// Registry manages available AgentRunner implementations.
// Runners are keyed by "role" or "role:specialization".
type Registry struct {
	mu      sync.RWMutex
	runners map[string]AgentRunner
}

func NewRegistry() *Registry {
	return &Registry{runners: make(map[string]AgentRunner)}
}

// Register adds a runner keyed by role.
func (r *Registry) Register(role string, runner AgentRunner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runners[role] = runner
}

// RegisterSpecialized adds a runner keyed by "role:specialization".
func (r *Registry) RegisterSpecialized(role, specialization string, runner AgentRunner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := role + ":" + specialization
	r.runners[key] = runner
}

// GetForSpec looks up a runner by role + specialization.
// Specialized lookups are strict: if a specialized runner is required but not
// registered, the lookup fails instead of falling back to the base role.
func (r *Registry) GetForSpec(role, specialization string) (AgentRunner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if specialization != "" && specialization != "general" {
		key := role + ":" + specialization
		if runner, ok := r.runners[key]; ok {
			return runner, nil
		}
		return nil, fmt.Errorf("no specialized agent runner registered for role %q (specialization %q)", role, specialization)
	}

	if runner, ok := r.runners[role]; ok {
		return runner, nil
	}

	return nil, fmt.Errorf("no agent runner registered for role %q (specialization %q)", role, specialization)
}

// Get looks up a runner by role (backward compatible).
func (r *Registry) Get(role string) (AgentRunner, error) {
	return r.GetForSpec(role, "")
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

	// Specialized workers
	r.RegisterSpecialized("worker", "backend", &MockBackendWorkerAgent{})
	r.RegisterSpecialized("worker", "frontend", &MockFrontendWorkerAgent{})
	r.RegisterSpecialized("worker", "qa", &MockQAWorkerAgent{})
}

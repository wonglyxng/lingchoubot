package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/lingchou/lingchoubot/backend/internal/model"
)

type bootstrapAgentRepo struct {
	agents map[string]*model.Agent
	seq    int
}

func (r *bootstrapAgentRepo) Create(_ context.Context, a *model.Agent) error {
	r.seq++
	copyAgent := *a
	if copyAgent.ID == "" {
		copyAgent.ID = fmt.Sprintf("agent-%d", r.seq)
	}
	if r.agents == nil {
		r.agents = map[string]*model.Agent{}
	}
	r.agents[copyAgent.ID] = &copyAgent
	a.ID = copyAgent.ID
	return nil
}

func (r *bootstrapAgentRepo) GetByID(_ context.Context, id string) (*model.Agent, error) {
	agent := r.agents[id]
	if agent == nil {
		return nil, nil
	}
	copyAgent := *agent
	return &copyAgent, nil
}

func (r *bootstrapAgentRepo) List(_ context.Context, _, _ int) ([]*model.Agent, int, error) {
	items := make([]*model.Agent, 0, len(r.agents))
	for _, agent := range r.agents {
		copyAgent := *agent
		items = append(items, &copyAgent)
	}
	return items, len(items), nil
}

func (r *bootstrapAgentRepo) Update(_ context.Context, a *model.Agent) error {
	copyAgent := *a
	r.agents[a.ID] = &copyAgent
	return nil
}

func (r *bootstrapAgentRepo) Delete(_ context.Context, id string) error {
	delete(r.agents, id)
	return nil
}

func (r *bootstrapAgentRepo) GetSubordinates(_ context.Context, _ string) ([]*model.Agent, error) {
	return nil, nil
}

func (r *bootstrapAgentRepo) GetOrgTree(_ context.Context, _ string) ([]*model.Agent, error) {
	return nil, nil
}

func (r *bootstrapAgentRepo) FindByRoleAndSpec(_ context.Context, role model.AgentRole, spec model.AgentSpecialization) (*model.Agent, error) {
	for _, agent := range r.agents {
		if agent.Role == role && agent.Specialization == spec && agent.Status == model.AgentStatusActive {
			copyAgent := *agent
			return &copyAgent, nil
		}
	}
	return nil, nil
}

func (r *bootstrapAgentRepo) FindByRoleCode(_ context.Context, roleCode model.RoleCode) (*model.Agent, error) {
	for _, agent := range r.agents {
		if agent.RoleCode == roleCode && agent.Status == model.AgentStatusActive {
			copyAgent := *agent
			return &copyAgent, nil
		}
	}
	return nil, nil
}

func TestEnsureBaselineAgentsCreatesHierarchy(t *testing.T) {
	ctx := context.Background()
	auditSvc, _ := newTestAuditService()
	repo := &bootstrapAgentRepo{agents: map[string]*model.Agent{}}
	svc := NewAgentService(repo, auditSvc)

	created, err := svc.EnsureBaselineAgents(ctx)
	if err != nil {
		t.Fatalf("EnsureBaselineAgents returned error: %v", err)
	}
	if len(created) != len(BaselineAgentSpecs()) {
		t.Fatalf("created agents = %d, want %d", len(created), len(BaselineAgentSpecs()))
	}

	byRoleCode := make(map[model.RoleCode]*model.Agent, len(created))
	for _, agent := range created {
		byRoleCode[agent.RoleCode] = agent
	}

	pm := byRoleCode[model.RoleCodePMSupervisor]
	if pm == nil {
		t.Fatal("missing PM baseline agent")
	}
	devSup := byRoleCode[model.RoleCodeDevelopmentSupervisor]
	if devSup == nil || devSup.ReportsTo == nil || *devSup.ReportsTo != pm.ID {
		t.Fatal("development supervisor should report to PM")
	}
	qaSup := byRoleCode[model.RoleCodeQASupervisor]
	if qaSup == nil || qaSup.ReportsTo == nil || *qaSup.ReportsTo != pm.ID {
		t.Fatal("QA supervisor should report to PM")
	}
	reviewer := byRoleCode[model.RoleCodeReviewerWorker]
	if reviewer == nil || reviewer.ReportsTo == nil || *reviewer.ReportsTo != qaSup.ID {
		t.Fatal("reviewer should report to QA supervisor")
	}
	frontend := byRoleCode[model.RoleCodeFrontendDevWorker]
	if frontend == nil || frontend.ReportsTo == nil || *frontend.ReportsTo != devSup.ID {
		t.Fatal("frontend worker should report to development supervisor")
	}
}

func TestEnsureBaselineAgentsIsIdempotent(t *testing.T) {
	ctx := context.Background()
	auditSvc, _ := newTestAuditService()
	repo := &bootstrapAgentRepo{agents: map[string]*model.Agent{}}
	svc := NewAgentService(repo, auditSvc)

	created, err := svc.EnsureBaselineAgents(ctx)
	if err != nil {
		t.Fatalf("first EnsureBaselineAgents returned error: %v", err)
	}
	if len(created) != len(BaselineAgentSpecs()) {
		t.Fatalf("first created agents = %d, want %d", len(created), len(BaselineAgentSpecs()))
	}

	created, err = svc.EnsureBaselineAgents(ctx)
	if err != nil {
		t.Fatalf("second EnsureBaselineAgents returned error: %v", err)
	}
	if len(created) != 0 {
		t.Fatalf("second created agents = %d, want 0", len(created))
	}
	if len(repo.agents) != len(BaselineAgentSpecs()) {
		t.Fatalf("stored agents = %d, want %d", len(repo.agents), len(BaselineAgentSpecs()))
	}
}

func TestEnsureBaselineAgentsCreatesOnlyMissingEntries(t *testing.T) {
	ctx := context.Background()
	auditSvc, _ := newTestAuditService()
	repo := &bootstrapAgentRepo{agents: map[string]*model.Agent{
		"pm-existing": {
			ID:             "pm-existing",
			Name:           "Existing PM",
			Role:           model.AgentRolePM,
			RoleCode:       model.RoleCodePMSupervisor,
			Status:         model.AgentStatusActive,
			AgentType:      model.AgentTypeLLM,
			Specialization: model.AgentSpecGeneral,
		},
	}}
	svc := NewAgentService(repo, auditSvc)

	created, err := svc.EnsureBaselineAgents(ctx)
	if err != nil {
		t.Fatalf("EnsureBaselineAgents returned error: %v", err)
	}
	if len(created) != len(BaselineAgentSpecs())-1 {
		t.Fatalf("created agents = %d, want %d", len(created), len(BaselineAgentSpecs())-1)
	}

	for _, agent := range created {
		if agent.RoleCode == model.RoleCodePMSupervisor {
			t.Fatal("PM should not be recreated when already present")
		}
		if (agent.RoleCode == model.RoleCodeDevelopmentSupervisor || agent.RoleCode == model.RoleCodeQASupervisor) && (agent.ReportsTo == nil || *agent.ReportsTo != "pm-existing") {
			t.Fatalf("supervisor %s should report to existing PM", agent.RoleCode)
		}
	}
}

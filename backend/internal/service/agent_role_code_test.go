package service

import (
	"context"
	"testing"

	"github.com/lingchou/lingchoubot/backend/internal/model"
)

func TestAgentServiceCreateDefaultsRoleCodeBySpecialization(t *testing.T) {
	tests := []struct {
		name           string
		role           model.AgentRole
		specialization model.AgentSpecialization
		want           model.RoleCode
	}{
		{name: "frontend worker", role: model.AgentRoleWorker, specialization: model.AgentSpecFrontend, want: model.RoleCodeFrontendDevWorker},
		{name: "backend worker", role: model.AgentRoleWorker, specialization: model.AgentSpecBackend, want: model.RoleCodeBackendDevWorker},
		{name: "qa worker", role: model.AgentRoleWorker, specialization: model.AgentSpecQA, want: model.RoleCodeQAWorker},
		{name: "qa supervisor", role: model.AgentRoleSupervisor, specialization: model.AgentSpecQA, want: model.RoleCodeQASupervisor},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			auditSvc, _ := newTestAuditService()
			repo := &bootstrapAgentRepo{agents: map[string]*model.Agent{}}
			svc := NewAgentService(repo, auditSvc)
			agent := &model.Agent{
				Name:           "Spec Agent",
				Role:           tt.role,
				Specialization: tt.specialization,
				AgentType:      model.AgentTypeMock,
			}

			if err := svc.Create(ctx, agent); err != nil {
				t.Fatalf("Create returned error: %v", err)
			}
			if agent.RoleCode != tt.want {
				t.Fatalf("role_code = %s, want %s", agent.RoleCode, tt.want)
			}
		})
	}
}

func TestAgentServiceUpdatePreservesExistingRoleCodeWhenOmitted(t *testing.T) {
	ctx := context.Background()
	auditSvc, _ := newTestAuditService()
	repo := &bootstrapAgentRepo{agents: map[string]*model.Agent{
		"frontend-worker": {
			ID:             "frontend-worker",
			Name:           "Frontend Worker",
			Role:           model.AgentRoleWorker,
			RoleCode:       model.RoleCodeFrontendDevWorker,
			Status:         model.AgentStatusActive,
			AgentType:      model.AgentTypeLLM,
			Specialization: model.AgentSpecFrontend,
			Metadata:       model.JSON(`{"llm":{"provider":"deepseek","model":"deepseek-chat"}}`),
		},
		"backend-worker": {
			ID:             "backend-worker",
			Name:           "Backend Worker",
			Role:           model.AgentRoleWorker,
			RoleCode:       model.RoleCodeBackendDevWorker,
			Status:         model.AgentStatusActive,
			AgentType:      model.AgentTypeLLM,
			Specialization: model.AgentSpecBackend,
			Metadata:       model.JSON(`{"llm":{"provider":"deepseek","model":"deepseek-chat"}}`),
		},
	}}
	svc := NewAgentService(repo, auditSvc)

	err := svc.Update(ctx, &model.Agent{
		ID:             "frontend-worker",
		Name:           "Frontend Worker",
		Role:           model.AgentRoleWorker,
		AgentType:      model.AgentTypeLLM,
		Specialization: model.AgentSpecFrontend,
		Description:    "updated",
		Metadata:       model.JSON(`{"llm":{"provider":"deepseek","model":"deepseek-chat"}}`),
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	updated, err := svc.GetByID(ctx, "frontend-worker")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if updated.RoleCode != model.RoleCodeFrontendDevWorker {
		t.Fatalf("role_code = %s, want %s", updated.RoleCode, model.RoleCodeFrontendDevWorker)
	}
}

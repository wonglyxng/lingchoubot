package service

import (
	"context"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
)

type BaselineAgentSpec struct {
	Agent             model.Agent
	ReportsToRoleCode model.RoleCode
}

func defaultBaselineAgentMetadata() model.JSON {
	return model.JSON(`{"llm":{"provider":"openai","model":"gpt-4.1-mini"}}`)
}

// BaselineAgentSpecs returns the minimal built-in agent set required for the MVP workflow.
// The slice order is significant: parents appear before children so startup bootstrap can
// wire reports_to relationships as agents are created.
func BaselineAgentSpecs() []BaselineAgentSpec {
	return []BaselineAgentSpec{
		{
			Agent: model.Agent{
				Name:           "PM Agent",
				Role:           model.AgentRolePM,
				RoleCode:       model.RoleCodePMSupervisor,
				AgentType:      model.AgentTypeLLM,
				Status:         model.AgentStatusActive,
				Specialization: model.AgentSpecGeneral,
				Description:    "项目负责人，负责项目级规划、协调与汇总。",
				ManagedRoles:   model.JSON(`["DEVELOPMENT_SUPERVISOR","QA_SUPERVISOR"]`),
				AllowedTools:   model.JSON(`["doc_generator"]`),
				Capabilities:   model.JSON(`{}`),
				Metadata:       defaultBaselineAgentMetadata(),
			},
		},
		{
			Agent: model.Agent{
				Name:           "Development Supervisor",
				Role:           model.AgentRoleSupervisor,
				RoleCode:       model.RoleCodeDevelopmentSupervisor,
				AgentType:      model.AgentTypeLLM,
				Status:         model.AgentStatusActive,
				Specialization: model.AgentSpecGeneral,
				Description:    "开发主管，负责开发域任务契约、分派与返工协调。",
				ManagedRoles:   model.JSON(`["BACKEND_DEV_WORKER","FRONTEND_DEV_WORKER"]`),
				AllowedTools:   model.JSON(`["doc_generator"]`),
				Capabilities:   model.JSON(`{}`),
				Metadata:       defaultBaselineAgentMetadata(),
			},
			ReportsToRoleCode: model.RoleCodePMSupervisor,
		},
		{
			Agent: model.Agent{
				Name:           "QA Supervisor",
				Role:           model.AgentRoleSupervisor,
				RoleCode:       model.RoleCodeQASupervisor,
				AgentType:      model.AgentTypeLLM,
				Status:         model.AgentStatusActive,
				Specialization: model.AgentSpecQA,
				Description:    "测试主管，负责 QA 任务编排、质量门把控与评审协调。",
				ManagedRoles:   model.JSON(`["QA_WORKER","REVIEWER_WORKER"]`),
				AllowedTools:   model.JSON(`["test_runner"]`),
				Capabilities:   model.JSON(`{}`),
				Metadata:       defaultBaselineAgentMetadata(),
			},
			ReportsToRoleCode: model.RoleCodePMSupervisor,
		},
		{
			Agent: model.Agent{
				Name:           "Backend Worker",
				Role:           model.AgentRoleWorker,
				RoleCode:       model.RoleCodeBackendDevWorker,
				AgentType:      model.AgentTypeLLM,
				Status:         model.AgentStatusActive,
				Specialization: model.AgentSpecBackend,
				Description:    "后端执行 Agent，负责 API、服务和数据库相关开发。",
				ManagedRoles:   model.JSON(`[]`),
				AllowedTools:   model.JSON(`["doc_generator","artifact_storage"]`),
				Capabilities:   model.JSON(`{}`),
				Metadata:       defaultBaselineAgentMetadata(),
			},
			ReportsToRoleCode: model.RoleCodeDevelopmentSupervisor,
		},
		{
			Agent: model.Agent{
				Name:           "Frontend Worker",
				Role:           model.AgentRoleWorker,
				RoleCode:       model.RoleCodeFrontendDevWorker,
				AgentType:      model.AgentTypeLLM,
				Status:         model.AgentStatusActive,
				Specialization: model.AgentSpecFrontend,
				Description:    "前端执行 Agent，负责页面实现与交互开发。",
				ManagedRoles:   model.JSON(`[]`),
				AllowedTools:   model.JSON(`["doc_generator","artifact_storage"]`),
				Capabilities:   model.JSON(`{}`),
				Metadata:       defaultBaselineAgentMetadata(),
			},
			ReportsToRoleCode: model.RoleCodeDevelopmentSupervisor,
		},
		{
			Agent: model.Agent{
				Name:           "QA Worker",
				Role:           model.AgentRoleWorker,
				RoleCode:       model.RoleCodeQAWorker,
				AgentType:      model.AgentTypeLLM,
				Status:         model.AgentStatusActive,
				Specialization: model.AgentSpecQA,
				Description:    "测试执行 Agent，负责验证、回归和测试交付。",
				ManagedRoles:   model.JSON(`[]`),
				AllowedTools:   model.JSON(`["test_runner"]`),
				Capabilities:   model.JSON(`{}`),
				Metadata:       defaultBaselineAgentMetadata(),
			},
			ReportsToRoleCode: model.RoleCodeQASupervisor,
		},
		{
			Agent: model.Agent{
				Name:           "Reviewer Agent",
				Role:           model.AgentRoleReviewer,
				RoleCode:       model.RoleCodeReviewerWorker,
				AgentType:      model.AgentTypeLLM,
				Status:         model.AgentStatusActive,
				Specialization: model.AgentSpecGeneral,
				Description:    "独立评审 Agent，负责对交付结果进行质量评审。",
				ManagedRoles:   model.JSON(`[]`),
				AllowedTools:   model.JSON(`[]`),
				Capabilities:   model.JSON(`{}`),
				Metadata:       defaultBaselineAgentMetadata(),
			},
			ReportsToRoleCode: model.RoleCodeQASupervisor,
		},
	}
}

// EnsureBaselineAgents creates the missing MVP baseline agents at startup.
// Existing agents are detected by role_code, so repeated calls are idempotent.
func (s *AgentService) EnsureBaselineAgents(ctx context.Context) ([]*model.Agent, error) {
	existing, _, err := s.repo.List(ctx, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}

	byRoleCode := make(map[model.RoleCode]*model.Agent, len(existing))
	for _, agent := range existing {
		if agent == nil || agent.RoleCode == "" {
			continue
		}
		if _, ok := byRoleCode[agent.RoleCode]; ok {
			continue
		}
		copyAgent := *agent
		byRoleCode[agent.RoleCode] = &copyAgent
	}

	created := make([]*model.Agent, 0)
	for _, spec := range BaselineAgentSpecs() {
		if _, ok := byRoleCode[spec.Agent.RoleCode]; ok {
			continue
		}

		agent := spec.Agent
		if spec.ReportsToRoleCode != "" {
			if parent := byRoleCode[spec.ReportsToRoleCode]; parent != nil {
				parentID := parent.ID
				agent.ReportsTo = &parentID
			}
		}

		if err := s.Create(ctx, &agent); err != nil {
			return nil, fmt.Errorf("bootstrap baseline agent %s: %w", agent.RoleCode, err)
		}

		copyAgent := agent
		byRoleCode[agent.RoleCode] = &copyAgent
		created = append(created, &copyAgent)
	}

	return created, nil
}

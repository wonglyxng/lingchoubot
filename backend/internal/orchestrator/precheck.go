package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lingchou/lingchoubot/backend/internal/model"
)

var ErrWorkflowPrecheckFailed = errors.New("workflow precheck failed")

var requiredWorkflowRoleCodes = []model.RoleCode{
	model.RoleCodePMSupervisor,
	model.RoleCodeDevelopmentSupervisor,
	model.RoleCodeQASupervisor,
	model.RoleCodeGeneralWorker,
	model.RoleCodeBackendDevWorker,
	model.RoleCodeFrontendDevWorker,
	model.RoleCodeQAWorker,
	model.RoleCodeReviewerWorker,
}

func validateWorkflowStartPreconditions(ctx context.Context, services *Services, projectID string) (*model.Project, error) {
	proj, err := services.Project.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	if proj == nil {
		return nil, fmt.Errorf("project %s not found", projectID)
	}

	agents, _, err := services.Agent.List(ctx, 200, 0)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}

	required := make(map[model.RoleCode]bool, len(requiredWorkflowRoleCodes))
	for _, roleCode := range requiredWorkflowRoleCodes {
		required[roleCode] = false
	}

	misconfigured := make([]string, 0)
	for _, agent := range agents {
		if agent.Status != model.AgentStatusActive {
			continue
		}
		if _, ok := required[agent.RoleCode]; !ok {
			continue
		}
		if err := validateWorkflowAgent(agent); err != nil {
			misconfigured = append(misconfigured, fmt.Sprintf("%s(%v)", agent.RoleCode, err))
			continue
		}
		required[agent.RoleCode] = true
	}

	missing := make([]string, 0)
	for _, roleCode := range requiredWorkflowRoleCodes {
		if !required[roleCode] {
			missing = append(missing, string(roleCode))
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: missing active llm agents for role_codes: %s", ErrWorkflowPrecheckFailed, strings.Join(missing, ", "))
	}
	if len(misconfigured) > 0 {
		return nil, fmt.Errorf("%w: misconfigured agents: %s", ErrWorkflowPrecheckFailed, strings.Join(misconfigured, "; "))
	}

	return proj, nil
}

func validateWorkflowAgent(agent *model.Agent) error {
	if agent == nil {
		return fmt.Errorf("agent is nil")
	}
	if agent.AgentType != model.AgentTypeLLM {
		return fmt.Errorf("agent_type=%s", agent.AgentType)
	}
	cfg, err := agent.GetLLMConfig()
	if err != nil {
		return fmt.Errorf("invalid llm config: %w", err)
	}
	if cfg == nil {
		return fmt.Errorf("missing llm config")
	}
	if !model.IsSupportedAgentLLMProvider(cfg.Provider) {
		return fmt.Errorf("unsupported llm provider=%s", cfg.Provider)
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return fmt.Errorf("missing llm model")
	}
	return nil
}

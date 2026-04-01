package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lingchou/lingchoubot/backend/internal/model"
)

var ErrWorkflowPrecheckFailed = errors.New("workflow precheck failed")

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

	required := map[model.AgentRole]bool{
		model.AgentRolePM:         false,
		model.AgentRoleSupervisor: false,
		model.AgentRoleWorker:     false,
		model.AgentRoleReviewer:   false,
	}
	for _, agent := range agents {
		if agent.Status != model.AgentStatusActive {
			continue
		}
		if _, ok := required[agent.Role]; ok {
			required[agent.Role] = true
		}
	}

	missing := make([]string, 0)
	orderedRoles := []model.AgentRole{
		model.AgentRolePM,
		model.AgentRoleSupervisor,
		model.AgentRoleWorker,
		model.AgentRoleReviewer,
	}
	for _, role := range orderedRoles {
		if !required[role] {
			missing = append(missing, string(role))
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: missing active agents for roles: %s", ErrWorkflowPrecheckFailed, strings.Join(missing, ", "))
	}

	return proj, nil
}
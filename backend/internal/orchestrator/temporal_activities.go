package orchestrator

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
	"github.com/lingchou/lingchoubot/backend/internal/runtime"
	"github.com/lingchou/lingchoubot/backend/internal/service"
)

// Activities groups all Temporal activity implementations.
// Each activity represents a recoverable step in the workflow.
type Activities struct {
	Registry *runtime.Registry
	Services *Services
	Workflow *service.WorkflowService
	Logger   *slog.Logger
}

// ActivityPM runs the PM agent to decompose a project into phases and tasks.
func (a *Activities) ActivityPM(ctx context.Context, input ProjectWorkflowInput) (*PMActivityResult, error) {
	proj, err := a.Services.Project.GetByID(ctx, input.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}

	agent, err := a.findAgent(ctx, model.AgentRolePM)
	if err != nil {
		return nil, err
	}

	runner, err := a.Registry.Get("pm")
	if err != nil {
		return nil, err
	}

	taskInput := &runtime.AgentTaskInput{
		RunID:       input.RunID,
		AgentID:     agent.ID,
		AgentRole:   "pm",
		Instruction: fmt.Sprintf("分解项目「%s」为阶段和任务", proj.Name),
		Project:     &runtime.ProjectCtx{ID: proj.ID, Name: proj.Name, Description: proj.Description},
	}

	output, err := runner.Execute(taskInput)
	if err != nil {
		return nil, fmt.Errorf("PM agent execution failed: %w", err)
	}
	if output.Status == runtime.OutputStatusFailed {
		return nil, fmt.Errorf("PM agent returned failure: %s", output.Error)
	}

	// Process phase actions
	for _, pa := range output.Phases {
		phase := &model.Phase{
			ProjectID:   input.ProjectID,
			Name:        pa.Name,
			Description: pa.Description,
			SortOrder:   pa.SortOrder,
			Status:      model.PhaseStatusPending,
		}
		if err := a.Services.Phase.Create(ctx, phase); err != nil {
			a.Logger.Error("create phase failed", "name", pa.Name, "error", err)
			return nil, err
		}
	}

	// Build phase name→ID map for task creation
	phases, _ := a.Services.Phase.ListByProject(ctx, input.ProjectID)
	phaseMap := make(map[string]string)
	for _, p := range phases {
		phaseMap[p.Name] = p.ID
	}

	for _, ta := range output.Tasks {
		phaseID, ok := phaseMap[ta.PhaseName]
		if !ok {
			a.Logger.Warn("phase not found for task", "phase", ta.PhaseName, "task", ta.Title)
			continue
		}
		task := &model.Task{
			ProjectID:   input.ProjectID,
			PhaseID:     &phaseID,
			Title:       ta.Title,
			Description: ta.Description,
			Priority:    ta.Priority,
			Status:      model.TaskStatusPending,
		}
		if err := a.Services.Task.Create(ctx, task); err != nil {
			a.Logger.Error("create task failed", "title", ta.Title, "error", err)
			return nil, err
		}
	}

	// Collect phase IDs for the workflow to iterate
	result := &PMActivityResult{PhaseIDs: make([]string, 0, len(phases))}
	for _, p := range phases {
		result.PhaseIDs = append(result.PhaseIDs, p.ID)
	}

	return result, nil
}

// ActivityListPhaseTasks returns the task IDs belonging to a given phase.
func (a *Activities) ActivityListPhaseTasks(ctx context.Context, input ListPhaseTasksInput) (*PhaseTasksResult, error) {
	tasks, _, err := a.Services.Task.List(ctx, repository.TaskListParams{
		PhaseID: input.PhaseID,
		Limit:   100,
		Offset:  0,
	})
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(tasks))
	for _, t := range tasks {
		ids = append(ids, t.ID)
	}
	return &PhaseTasksResult{TaskIDs: ids}, nil
}

// ActivitySupervisor runs the Supervisor agent for a given task.
func (a *Activities) ActivitySupervisor(ctx context.Context, input TaskChainInput) (*StepResult, error) {
	proj, _ := a.Services.Project.GetByID(ctx, input.ProjectID)
	phase, _ := a.Services.Phase.GetByID(ctx, input.PhaseID)
	task, _ := a.Services.Task.GetByID(ctx, input.TaskID)

	if proj == nil || task == nil {
		return nil, fmt.Errorf("project or task not found")
	}

	agent, err := a.findAgent(ctx, model.AgentRoleSupervisor)
	if err != nil {
		return nil, err
	}

	runner, err := a.Registry.Get("supervisor")
	if err != nil {
		return nil, err
	}

	taskInput := &runtime.AgentTaskInput{
		RunID:       input.RunID,
		AgentID:     agent.ID,
		AgentRole:   "supervisor",
		Instruction: fmt.Sprintf("为任务「%s」创建契约并分派执行者", task.Title),
		Project:     &runtime.ProjectCtx{ID: proj.ID, Name: proj.Name, Description: proj.Description},
		Task:        &runtime.TaskCtx{ID: task.ID, ProjectID: proj.ID, PhaseID: input.PhaseID, Title: task.Title, Description: task.Description, Priority: task.Priority},
	}
	if phase != nil {
		taskInput.Phase = &runtime.PhaseCtx{ID: phase.ID, ProjectID: proj.ID, Name: phase.Name, Description: phase.Description, SortOrder: phase.SortOrder}
	}

	output, err := runner.Execute(taskInput)
	if err != nil {
		return nil, err
	}
	if output.Status == runtime.OutputStatusFailed {
		return nil, fmt.Errorf("supervisor failed: %s", output.Error)
	}

	// Use the same Engine action processors via a temporary local engine
	eng := &Engine{registry: a.Registry, services: a.Services, workflow: a.Workflow, logger: a.Logger}
	eng.processContractActions(ctx, task.ID, output.Contracts)
	eng.processAssignmentActions(ctx, task.ID, agent.ID, output.Assignments)
	eng.processTransitionActions(ctx, task, output.Transitions)

	return &StepResult{Summary: output.Summary}, nil
}

// ActivityWorker runs the Worker agent for a given task.
func (a *Activities) ActivityWorker(ctx context.Context, input TaskChainInput) (*StepResult, error) {
	proj, _ := a.Services.Project.GetByID(ctx, input.ProjectID)
	task, _ := a.Services.Task.GetByID(ctx, input.TaskID)

	if proj == nil || task == nil {
		return nil, fmt.Errorf("project or task not found")
	}

	spec := inferSpecialization(task)
	agent, err := a.findAgentWithSpec(ctx, model.AgentRoleWorker, spec)
	if err != nil {
		return nil, err
	}

	_ = a.Services.Task.TransitionStatus(ctx, task.ID, model.TaskStatusInProgress)

	runner, err := a.Registry.GetForSpec("worker", string(agent.Specialization))
	if err != nil {
		return nil, err
	}

	var contractCtx *runtime.ContractCtx
	contract, _ := a.Services.Contract.GetLatestByTaskID(ctx, task.ID)
	if contract != nil {
		contractCtx = &runtime.ContractCtx{ID: contract.ID, Scope: contract.Scope}
	}

	taskInput := &runtime.AgentTaskInput{
		RunID:       input.RunID,
		AgentID:     agent.ID,
		AgentRole:   "worker",
		Instruction: fmt.Sprintf("执行任务「%s」并生成交付物", task.Title),
		Project:     &runtime.ProjectCtx{ID: proj.ID, Name: proj.Name, Description: proj.Description},
		Task:        &runtime.TaskCtx{ID: task.ID, ProjectID: proj.ID, PhaseID: input.PhaseID, Title: task.Title, Description: task.Description, Priority: task.Priority},
		Contract:    contractCtx,
	}

	output, err := runner.Execute(taskInput)
	if err != nil {
		return nil, err
	}
	if output.Status == runtime.OutputStatusFailed {
		return nil, fmt.Errorf("worker failed: %s", output.Error)
	}

	eng := &Engine{registry: a.Registry, services: a.Services, workflow: a.Workflow, logger: a.Logger}
	eng.processArtifactActions(ctx, proj.ID, task.ID, agent.ID, output.Artifacts)
	eng.processHandoffActions(ctx, task.ID, agent.ID, output.Handoffs)
	eng.processTransitionActions(ctx, task, output.Transitions)

	return &StepResult{Summary: output.Summary}, nil
}

// ActivityReviewer runs the Reviewer agent for a given task.
func (a *Activities) ActivityReviewer(ctx context.Context, input TaskChainInput) (*StepResult, error) {
	proj, _ := a.Services.Project.GetByID(ctx, input.ProjectID)
	task, _ := a.Services.Task.GetByID(ctx, input.TaskID)

	if proj == nil || task == nil {
		return nil, fmt.Errorf("project or task not found")
	}

	agent, err := a.findAgent(ctx, model.AgentRoleReviewer)
	if err != nil {
		return nil, err
	}

	runner, err := a.Registry.Get("reviewer")
	if err != nil {
		return nil, err
	}

	taskInput := &runtime.AgentTaskInput{
		RunID:       input.RunID,
		AgentID:     agent.ID,
		AgentRole:   "reviewer",
		Instruction: fmt.Sprintf("评审任务「%s」的交付物", task.Title),
		Project:     &runtime.ProjectCtx{ID: proj.ID, Name: proj.Name, Description: proj.Description},
		Task:        &runtime.TaskCtx{ID: task.ID, ProjectID: proj.ID, PhaseID: input.PhaseID, Title: task.Title, Description: task.Description, Priority: task.Priority},
	}

	output, err := runner.Execute(taskInput)
	if err != nil {
		return nil, err
	}
	if output.Status == runtime.OutputStatusFailed {
		return nil, fmt.Errorf("reviewer failed: %s", output.Error)
	}

	eng := &Engine{registry: a.Registry, services: a.Services, workflow: a.Workflow, logger: a.Logger}
	eng.processReviewActions(ctx, task.ID, agent.ID, output.Reviews)

	return &StepResult{Summary: output.Summary}, nil
}

// ActivityCompleteRun marks the workflow run as completed.
func (a *Activities) ActivityCompleteRun(ctx context.Context, input CompleteRunInput) error {
	run, err := a.Workflow.GetRun(ctx, input.RunID)
	if err != nil {
		return err
	}
	if run == nil {
		return fmt.Errorf("run %s not found", input.RunID)
	}

	summary := fmt.Sprintf("项目工作流完成（Temporal 模式）")
	return a.Workflow.CompleteRun(ctx, run, summary)
}

// ActivityFailRun marks the workflow run as failed.
func (a *Activities) ActivityFailRun(ctx context.Context, input FailRunInput) error {
	run, err := a.Workflow.GetRun(ctx, input.RunID)
	if err != nil {
		return err
	}
	if run == nil {
		return nil
	}
	return a.Workflow.FailRun(ctx, run, input.Error)
}

// --- Helpers ---

func (a *Activities) findAgent(ctx context.Context, role model.AgentRole) (*model.Agent, error) {
	return a.findAgentWithSpec(ctx, role, model.AgentSpecGeneral)
}

func (a *Activities) findAgentWithSpec(ctx context.Context, role model.AgentRole, spec model.AgentSpecialization) (*model.Agent, error) {
	agent, err := a.Services.Agent.FindByRoleAndSpec(ctx, role, spec)
	if err != nil {
		return nil, fmt.Errorf("find agent (%s/%s): %w", role, spec, err)
	}
	if agent != nil {
		return agent, nil
	}

	agents, _, err := a.Services.Agent.List(ctx, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	for _, ag := range agents {
		if ag.Role == role && ag.Status == model.AgentStatusActive {
			return ag, nil
		}
	}
	return nil, fmt.Errorf("no active agent with role %q (specialization %q) found", role, spec)
}

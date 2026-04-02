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

// stepTracker is used in Temporal activities to create and manage workflow_step records.
type stepTracker struct {
	workflow *service.WorkflowService
	runID    string
	step     *model.WorkflowStep
}

func (a *Activities) newStep(ctx context.Context, runID, name, agentRole string, sortOrder int) (*stepTracker, error) {
	step, err := a.Workflow.AddStep(ctx, runID, name, agentRole, sortOrder)
	if err != nil {
		return nil, fmt.Errorf("add workflow step: %w", err)
	}
	if err := a.Workflow.StartStep(ctx, step); err != nil {
		return nil, fmt.Errorf("start workflow step: %w", err)
	}
	return &stepTracker{workflow: a.Workflow, runID: runID, step: step}, nil
}

func (st *stepTracker) complete(ctx context.Context, summary string) {
	_ = st.workflow.CompleteStep(ctx, st.step, summary)
}

func (st *stepTracker) fail(ctx context.Context, errMsg string) {
	_ = st.workflow.FailStep(ctx, st.step, errMsg)
}

func (st *stepTracker) setAgent(id string) { st.step.AgentID = &id }
func (st *stepTracker) setTask(id string)  { st.step.TaskID = &id }
func (st *stepTracker) setPhase(id string) { st.step.PhaseID = &id }

// ActivityPM runs the PM agent to decompose a project into phases and tasks.
func (a *Activities) ActivityPM(ctx context.Context, input ProjectWorkflowInput) (*PMActivityResult, error) {
	proj, err := a.Services.Project.GetByID(ctx, input.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}

	st, err := a.newStep(ctx, input.RunID, "PM 项目分解", "pm", 1)
	if err != nil {
		return nil, err
	}

	agent, err := a.findAgent(ctx, model.AgentRolePM)
	if err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}
	st.setAgent(agent.ID)

	runner, err := a.Registry.Get("pm")
	if err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}
	agentLLM, err := runtimeAgentLLMConfig(agent)
	if err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}

	taskInput := &runtime.AgentTaskInput{
		RunID:       input.RunID,
		AgentID:     agent.ID,
		AgentRole:   "pm",
		AgentLLM:    agentLLM,
		Instruction: fmt.Sprintf("分解项目「%s」为阶段和任务", proj.Name),
		Project:     &runtime.ProjectCtx{ID: proj.ID, Name: proj.Name, Description: proj.Description},
	}

	output, err := runner.Execute(taskInput)
	if err != nil {
		st.fail(ctx, err.Error())
		return nil, fmt.Errorf("PM agent execution failed: %w", err)
	}
	if output.Status == runtime.OutputStatusFailed {
		st.fail(ctx, output.Error)
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
			st.fail(ctx, err.Error())
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
		task.ExecutionDomain = inferExecutionDomain(task)
		if err := a.Services.Task.Create(ctx, task); err != nil {
			a.Logger.Error("create task failed", "title", ta.Title, "error", err)
			st.fail(ctx, err.Error())
			return nil, err
		}
	}

	// Collect phase IDs for the workflow to iterate
	result := &PMActivityResult{
		PhaseIDs:  make([]string, 0, len(phases)),
		StepCount: 1,
	}
	for _, p := range phases {
		result.PhaseIDs = append(result.PhaseIDs, p.ID)
	}

	st.complete(ctx, output.Summary)
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

	// Infer execution domain
	domain := inferExecutionDomain(task)
	if task.ExecutionDomain != domain {
		task.ExecutionDomain = domain
	}

	sortOrder := input.SortOffset + 1
	st, err := a.newStep(ctx, input.RunID, fmt.Sprintf("主管规划「%s」(%s)", task.Title, domain), "supervisor", sortOrder)
	if err != nil {
		return nil, err
	}
	st.setTask(task.ID)
	if phase != nil {
		st.setPhase(phase.ID)
	}

	// Route to correct supervisor by domain
	agent, err := a.findSupervisorByDomain(ctx, domain)
	if err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}
	st.setAgent(agent.ID)

	// Record owner supervisor on task
	task.OwnerSupervisorID = &agent.ID
	_ = a.Services.Task.Update(ctx, task)

	runner, err := a.Registry.Get("supervisor")
	if err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}
	agentLLM, err := runtimeAgentLLMConfig(agent)
	if err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}

	taskInput := &runtime.AgentTaskInput{
		RunID:       input.RunID,
		AgentID:     agent.ID,
		AgentRole:   "supervisor",
		AgentLLM:    agentLLM,
		Instruction: fmt.Sprintf("为任务「%s」创建契约并分派执行者", task.Title),
		Project:     &runtime.ProjectCtx{ID: proj.ID, Name: proj.Name, Description: proj.Description},
		Task:        &runtime.TaskCtx{ID: task.ID, ProjectID: proj.ID, PhaseID: input.PhaseID, Title: task.Title, Description: task.Description, Priority: task.Priority},
	}
	if phase != nil {
		taskInput.Phase = &runtime.PhaseCtx{ID: phase.ID, ProjectID: proj.ID, Name: phase.Name, Description: phase.Description, SortOrder: phase.SortOrder}
	}

	output, err := runner.Execute(taskInput)
	if err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}
	if output.Status == runtime.OutputStatusFailed {
		st.fail(ctx, output.Error)
		return nil, fmt.Errorf("supervisor failed: %s", output.Error)
	}

	eng := &Engine{registry: a.Registry, services: a.Services, workflow: a.Workflow, logger: a.Logger}
	if err := eng.processContractActions(ctx, task.ID, output.Contracts); err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}
	if err := eng.processAssignmentActions(ctx, task, agent.ID, output.Assignments); err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}
	if err := eng.processTransitionActions(ctx, task, output.Transitions); err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}

	st.complete(ctx, output.Summary)
	return &StepResult{Summary: output.Summary, StepCount: sortOrder}, nil
}

// ActivityWorker runs the Worker agent for a given task.
func (a *Activities) ActivityWorker(ctx context.Context, input TaskChainInput) (*StepResult, error) {
	proj, _ := a.Services.Project.GetByID(ctx, input.ProjectID)
	task, _ := a.Services.Task.GetByID(ctx, input.TaskID)

	if proj == nil || task == nil {
		return nil, fmt.Errorf("project or task not found")
	}

	spec := inferSpecialization(task)
	sortOrder := input.SortOffset + 1
	stepName := fmt.Sprintf("执行「%s」", task.Title)
	if spec != model.AgentSpecGeneral {
		stepName = fmt.Sprintf("执行「%s」(%s)", task.Title, spec)
	}

	st, err := a.newStep(ctx, input.RunID, stepName, "worker", sortOrder)
	if err != nil {
		return nil, err
	}
	st.setTask(task.ID)

	agent, err := a.findAgentWithSpec(ctx, model.AgentRoleWorker, spec)
	if err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}
	st.setAgent(agent.ID)

	_ = a.Services.Task.TransitionStatus(ctx, task.ID, model.TaskStatusInProgress)

	runner, err := a.Registry.GetForSpec("worker", string(agent.Specialization))
	if err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}
	agentLLM, err := runtimeAgentLLMConfig(agent)
	if err != nil {
		st.fail(ctx, err.Error())
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
		AgentLLM:    agentLLM,
		Instruction: fmt.Sprintf("执行任务「%s」并生成交付物", task.Title),
		Project:     &runtime.ProjectCtx{ID: proj.ID, Name: proj.Name, Description: proj.Description},
		Task:        &runtime.TaskCtx{ID: task.ID, ProjectID: proj.ID, PhaseID: input.PhaseID, Title: task.Title, Description: task.Description, Priority: task.Priority},
		Contract:    contractCtx,
	}

	output, err := runner.Execute(taskInput)
	if err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}
	if output.Status == runtime.OutputStatusFailed {
		st.fail(ctx, output.Error)
		return nil, fmt.Errorf("worker failed: %s", output.Error)
	}

	eng := &Engine{registry: a.Registry, services: a.Services, workflow: a.Workflow, logger: a.Logger}
	if err := eng.processArtifactActions(ctx, proj.ID, task.ID, agent.ID, output.Artifacts); err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}
	if err := eng.processHandoffActions(ctx, task.ID, agent.ID, output.Handoffs); err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}
	if err := eng.processTransitionActions(ctx, task, output.Transitions); err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}

	st.complete(ctx, output.Summary)
	return &StepResult{Summary: output.Summary, StepCount: sortOrder}, nil
}

// ActivityReviewer runs the Reviewer agent for a given task.
func (a *Activities) ActivityReviewer(ctx context.Context, input TaskChainInput) (*StepResult, error) {
	proj, _ := a.Services.Project.GetByID(ctx, input.ProjectID)
	task, _ := a.Services.Task.GetByID(ctx, input.TaskID)

	if proj == nil || task == nil {
		return nil, fmt.Errorf("project or task not found")
	}

	sortOrder := input.SortOffset + 1
	st, err := a.newStep(ctx, input.RunID, fmt.Sprintf("评审「%s」", task.Title), "reviewer", sortOrder)
	if err != nil {
		return nil, err
	}
	st.setTask(task.ID)

	agent, err := a.findAgent(ctx, model.AgentRoleReviewer)
	if err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}
	st.setAgent(agent.ID)

	runner, err := a.Registry.Get("reviewer")
	if err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}
	agentLLM, err := runtimeAgentLLMConfig(agent)
	if err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}
	eng := &Engine{registry: a.Registry, services: a.Services, workflow: a.Workflow, logger: a.Logger}
	artifactCtxs, err := eng.loadTaskArtifactContexts(ctx, task.ID)
	if err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}
	var phaseCtx *runtime.PhaseCtx
	if input.PhaseID != "" {
		phase, phaseErr := a.Services.Phase.GetByID(ctx, input.PhaseID)
		if phaseErr != nil {
			st.fail(ctx, phaseErr.Error())
			return nil, phaseErr
		}
		if phase != nil {
			phaseCtx = &runtime.PhaseCtx{ID: phase.ID, ProjectID: proj.ID, Name: phase.Name, Description: phase.Description, SortOrder: phase.SortOrder}
		}
	}

	taskInput := &runtime.AgentTaskInput{
		RunID:       input.RunID,
		AgentID:     agent.ID,
		AgentRole:   "reviewer",
		AgentLLM:    agentLLM,
		Instruction: fmt.Sprintf("评审任务「%s」的交付物", task.Title),
		Project:     &runtime.ProjectCtx{ID: proj.ID, Name: proj.Name, Description: proj.Description},
		Phase:       phaseCtx,
		Task:        &runtime.TaskCtx{ID: task.ID, ProjectID: proj.ID, PhaseID: input.PhaseID, Title: task.Title, Description: task.Description, Priority: task.Priority},
		Artifacts:   artifactCtxs,
	}

	output, err := runner.Execute(taskInput)
	if err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}
	if output.Status == runtime.OutputStatusFailed {
		st.fail(ctx, output.Error)
		return nil, fmt.Errorf("reviewer failed: %s", output.Error)
	}

	if err := eng.processReviewActions(ctx, input.RunID, task.ID, agent.ID, artifactCtxs, output.Reviews); err != nil {
		st.fail(ctx, err.Error())
		return nil, err
	}

	st.complete(ctx, output.Summary)
	return &StepResult{Summary: output.Summary, StepCount: sortOrder}, nil
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

// ActivityCheckRework checks if a task needs rework after review.
// Returns true if the task status is "revision_required" and rework should continue.
func (a *Activities) ActivityCheckRework(ctx context.Context, input CheckReworkInput) (bool, error) {
	task, err := a.Services.Task.GetByID(ctx, input.TaskID)
	if err != nil {
		return false, fmt.Errorf("get task for rework check: %w", err)
	}
	if task == nil {
		return false, fmt.Errorf("task %s not found", input.TaskID)
	}

	if task.Status != model.TaskStatusRevisionRequired {
		return false, nil
	}

	a.Logger.Info("rework triggered (Temporal)", "task", task.Title, "attempt", input.Attempt)
	// Transition back to in_progress for the rework cycle
	_ = a.Services.Task.TransitionStatus(ctx, task.ID, model.TaskStatusInProgress)
	return true, nil
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
	if agent != nil && agent.Specialization == spec {
		return agent, nil
	}
	return nil, fmt.Errorf("no active agent with role %q (specialization %q) found", role, spec)
}

// findSupervisorByDomain locates the correct supervisor for a task's execution domain (Temporal mode).
func (a *Activities) findSupervisorByDomain(ctx context.Context, domain model.ExecutionDomain) (*model.Agent, error) {
	var roleCode model.RoleCode
	switch domain {
	case model.ExecDomainDevelopment:
		roleCode = model.RoleCodeDevelopmentSupervisor
	case model.ExecDomainQA:
		roleCode = model.RoleCodeQASupervisor
	default:
		return a.findAgent(ctx, model.AgentRoleSupervisor)
	}
	agent, err := a.Services.Agent.FindByRoleCode(ctx, roleCode)
	if err != nil {
		return nil, fmt.Errorf("find supervisor by domain (%s): %w", domain, err)
	}
	if agent != nil {
		return agent, nil
	}
	return nil, fmt.Errorf("no active supervisor with role_code %q found", roleCode)
}

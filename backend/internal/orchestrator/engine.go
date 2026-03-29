package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
	"github.com/lingchou/lingchoubot/backend/internal/runtime"
	"github.com/lingchou/lingchoubot/backend/internal/service"
)

// Services bundles all the service dependencies the engine needs.
type Services struct {
	Project      *service.ProjectService
	Phase        *service.PhaseService
	Agent        *service.AgentService
	Task         *service.TaskService
	Contract     *service.TaskContractService
	Assignment   *service.TaskAssignmentService
	Artifact     *service.ArtifactService
	Handoff      *service.HandoffSnapshotService
	Review       *service.ReviewReportService
	Approval     *service.ApprovalRequestService
	Audit        *service.AuditService
}

// Engine orchestrates the Supervisor → Worker → Reviewer chain.
type Engine struct {
	registry *runtime.Registry
	services *Services
	store    *RunStore
	logger   *slog.Logger
}

func NewEngine(reg *runtime.Registry, svc *Services, store *RunStore, logger *slog.Logger) *Engine {
	return &Engine{registry: reg, services: svc, store: store, logger: logger}
}

func (e *Engine) Store() *RunStore { return e.store }

// Run executes the full workflow for a project: PM → Supervisor → Worker → Reviewer.
func (e *Engine) Run(ctx context.Context, projectID string) (*WorkflowRun, error) {
	proj, err := e.services.Project.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	if proj == nil {
		return nil, fmt.Errorf("project %s not found", projectID)
	}

	run := &WorkflowRun{
		ID:          newID(),
		ProjectID:   proj.ID,
		ProjectName: proj.Name,
		Status:      RunStatusRunning,
		StartedAt:   time.Now(),
	}
	e.store.Save(run)

	e.services.Audit.LogEvent(ctx, "system", "", "workflow.started",
		fmt.Sprintf("项目「%s」工作流已启动", proj.Name),
		"project", proj.ID, nil, map[string]string{"run_id": run.ID})

	if err := e.runPMPhase(ctx, run, proj); err != nil {
		e.failRun(ctx, run, err)
		return run, nil
	}

	phases, err := e.services.Phase.ListByProject(ctx, proj.ID)
	if err != nil {
		e.failRun(ctx, run, err)
		return run, nil
	}

	for _, phase := range phases {
		if err := e.runPhase(ctx, run, proj, phase); err != nil {
			e.logger.Error("phase failed", "phase", phase.Name, "error", err)
			continue
		}
	}

	now := time.Now()
	run.Status = RunStatusCompleted
	run.CompletedAt = &now
	run.Summary = fmt.Sprintf("项目「%s」工作流完成：%d 个阶段已处理，共 %d 步",
		proj.Name, len(phases), len(run.Steps))
	e.store.Save(run)

	e.services.Audit.LogEvent(ctx, "system", "", "workflow.completed",
		run.Summary, "project", proj.ID, nil, map[string]string{"run_id": run.ID})

	return run, nil
}

// runPMPhase runs the PM agent to decompose the project.
func (e *Engine) runPMPhase(ctx context.Context, run *WorkflowRun, proj *model.Project) error {
	step := run.AddStep("PM 项目分解", "pm")
	step.Start()

	pmAgent, err := e.findAgent(ctx, model.AgentRolePM)
	if err != nil {
		step.Fail(err.Error())
		return err
	}
	step.AgentID = pmAgent.ID

	runner, err := e.registry.Get("pm")
	if err != nil {
		step.Fail(err.Error())
		return err
	}

	input := &runtime.AgentTaskInput{
		RunID:       run.ID,
		AgentID:     pmAgent.ID,
		AgentRole:   "pm",
		Instruction: fmt.Sprintf("分解项目「%s」为阶段和任务", proj.Name),
		Project:     &runtime.ProjectCtx{ID: proj.ID, Name: proj.Name, Description: proj.Description},
	}

	output, err := runner.Execute(input)
	if err != nil {
		step.Fail(err.Error())
		return fmt.Errorf("PM agent execution failed: %w", err)
	}
	if output.Status == runtime.OutputStatusFailed {
		step.Fail(output.Error)
		return fmt.Errorf("PM agent returned failure: %s", output.Error)
	}

	if err := e.processPhaseActions(ctx, proj.ID, output.Phases); err != nil {
		step.Fail(err.Error())
		return err
	}

	phases, _ := e.services.Phase.ListByProject(ctx, proj.ID)
	phaseMap := make(map[string]string)
	for _, p := range phases {
		phaseMap[p.Name] = p.ID
	}
	if err := e.processTaskActions(ctx, proj.ID, phaseMap, output.Tasks); err != nil {
		step.Fail(err.Error())
		return err
	}

	step.Complete(output.Summary)
	e.store.Save(run)
	return nil
}

// runPhase processes all tasks within a phase.
func (e *Engine) runPhase(ctx context.Context, run *WorkflowRun, proj *model.Project, phase *model.Phase) error {
	tasks, _, err := e.services.Task.List(ctx, repository.TaskListParams{
		PhaseID: phase.ID,
		Limit:   100,
		Offset:  0,
	})
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		return nil
	}

	for _, task := range tasks {
		if err := e.runTaskChain(ctx, run, proj, phase, task); err != nil {
			e.logger.Error("task chain failed", "task", task.Title, "error", err)
		}
	}
	return nil
}

// runTaskChain executes Supervisor → Worker → Reviewer for a single task.
func (e *Engine) runTaskChain(ctx context.Context, run *WorkflowRun, proj *model.Project, phase *model.Phase, task *model.Task) error {
	if err := e.runSupervisor(ctx, run, proj, phase, task); err != nil {
		return err
	}

	if err := e.runWorker(ctx, run, proj, phase, task); err != nil {
		return err
	}

	if err := e.runReviewer(ctx, run, proj, phase, task); err != nil {
		return err
	}

	return nil
}

func (e *Engine) runSupervisor(ctx context.Context, run *WorkflowRun, proj *model.Project, phase *model.Phase, task *model.Task) error {
	step := run.AddStep(fmt.Sprintf("主管规划「%s」", task.Title), "supervisor")
	step.Start()
	step.TaskID = task.ID
	step.PhaseID = phase.ID

	agent, err := e.findAgent(ctx, model.AgentRoleSupervisor)
	if err != nil {
		step.Fail(err.Error())
		return err
	}
	step.AgentID = agent.ID

	runner, err := e.registry.Get("supervisor")
	if err != nil {
		step.Fail(err.Error())
		return err
	}

	input := &runtime.AgentTaskInput{
		RunID:       run.ID,
		AgentID:     agent.ID,
		AgentRole:   "supervisor",
		Instruction: fmt.Sprintf("为任务「%s」创建契约并分派执行者", task.Title),
		Project:     &runtime.ProjectCtx{ID: proj.ID, Name: proj.Name, Description: proj.Description},
		Phase:       &runtime.PhaseCtx{ID: phase.ID, ProjectID: proj.ID, Name: phase.Name, Description: phase.Description, SortOrder: phase.SortOrder},
		Task:        &runtime.TaskCtx{ID: task.ID, ProjectID: proj.ID, PhaseID: phase.ID, Title: task.Title, Description: task.Description, Priority: task.Priority},
	}

	output, err := runner.Execute(input)
	if err != nil {
		step.Fail(err.Error())
		return err
	}
	if output.Status == runtime.OutputStatusFailed {
		step.Fail(output.Error)
		return fmt.Errorf("supervisor failed: %s", output.Error)
	}

	e.processContractActions(ctx, task.ID, output.Contracts)
	e.processAssignmentActions(ctx, task.ID, agent.ID, output.Assignments)
	e.processTransitionActions(ctx, task, output.Transitions)

	step.Complete(output.Summary)
	e.store.Save(run)
	return nil
}

func (e *Engine) runWorker(ctx context.Context, run *WorkflowRun, proj *model.Project, phase *model.Phase, task *model.Task) error {
	step := run.AddStep(fmt.Sprintf("执行「%s」", task.Title), "worker")
	step.Start()
	step.TaskID = task.ID

	agent, err := e.findAgent(ctx, model.AgentRoleWorker)
	if err != nil {
		step.Fail(err.Error())
		return err
	}
	step.AgentID = agent.ID

	// Transition to in_progress
	_ = e.services.Task.TransitionStatus(ctx, task.ID, model.TaskStatusInProgress)

	runner, err := e.registry.Get("worker")
	if err != nil {
		step.Fail(err.Error())
		return err
	}

	var contractCtx *runtime.ContractCtx
	contract, _ := e.services.Contract.GetLatestByTaskID(ctx, task.ID)
	if contract != nil {
		contractCtx = &runtime.ContractCtx{
			ID:    contract.ID,
			Scope: contract.Scope,
		}
	}

	input := &runtime.AgentTaskInput{
		RunID:       run.ID,
		AgentID:     agent.ID,
		AgentRole:   "worker",
		Instruction: fmt.Sprintf("执行任务「%s」并生成交付物", task.Title),
		Project:     &runtime.ProjectCtx{ID: proj.ID, Name: proj.Name, Description: proj.Description},
		Phase:       &runtime.PhaseCtx{ID: phase.ID, ProjectID: proj.ID, Name: phase.Name, Description: phase.Description, SortOrder: phase.SortOrder},
		Task:        &runtime.TaskCtx{ID: task.ID, ProjectID: proj.ID, PhaseID: phase.ID, Title: task.Title, Description: task.Description, Priority: task.Priority},
		Contract:    contractCtx,
	}

	output, err := runner.Execute(input)
	if err != nil {
		step.Fail(err.Error())
		return err
	}
	if output.Status == runtime.OutputStatusFailed {
		step.Fail(output.Error)
		return fmt.Errorf("worker failed: %s", output.Error)
	}

	e.processArtifactActions(ctx, proj.ID, task.ID, agent.ID, output.Artifacts)
	e.processHandoffActions(ctx, task.ID, agent.ID, output.Handoffs)
	e.processTransitionActions(ctx, task, output.Transitions)

	step.Complete(output.Summary)
	e.store.Save(run)
	return nil
}

func (e *Engine) runReviewer(ctx context.Context, run *WorkflowRun, proj *model.Project, phase *model.Phase, task *model.Task) error {
	step := run.AddStep(fmt.Sprintf("评审「%s」", task.Title), "reviewer")
	step.Start()
	step.TaskID = task.ID

	agent, err := e.findAgent(ctx, model.AgentRoleReviewer)
	if err != nil {
		step.Fail(err.Error())
		return err
	}
	step.AgentID = agent.ID

	runner, err := e.registry.Get("reviewer")
	if err != nil {
		step.Fail(err.Error())
		return err
	}

	input := &runtime.AgentTaskInput{
		RunID:       run.ID,
		AgentID:     agent.ID,
		AgentRole:   "reviewer",
		Instruction: fmt.Sprintf("评审任务「%s」的交付物", task.Title),
		Project:     &runtime.ProjectCtx{ID: proj.ID, Name: proj.Name, Description: proj.Description},
		Task:        &runtime.TaskCtx{ID: task.ID, ProjectID: proj.ID, PhaseID: phase.ID, Title: task.Title, Description: task.Description, Priority: task.Priority},
	}

	output, err := runner.Execute(input)
	if err != nil {
		step.Fail(err.Error())
		return err
	}
	if output.Status == runtime.OutputStatusFailed {
		step.Fail(output.Error)
		return fmt.Errorf("reviewer failed: %s", output.Error)
	}

	e.processReviewActions(ctx, task.ID, agent.ID, output.Reviews)

	step.Complete(output.Summary)
	e.store.Save(run)
	return nil
}

// --- Action processors: translate agent outputs into service calls ---

func (e *Engine) processPhaseActions(ctx context.Context, projectID string, actions []runtime.PhaseAction) error {
	for _, a := range actions {
		phase := &model.Phase{
			ProjectID:   projectID,
			Name:        a.Name,
			Description: a.Description,
			SortOrder:   a.SortOrder,
			Status:      model.PhaseStatusPending,
		}
		if err := e.services.Phase.Create(ctx, phase); err != nil {
			e.logger.Error("create phase failed", "name", a.Name, "error", err)
			return err
		}
	}
	return nil
}

func (e *Engine) processTaskActions(ctx context.Context, projectID string, phaseMap map[string]string, actions []runtime.TaskAction) error {
	for _, a := range actions {
		phaseID, ok := phaseMap[a.PhaseName]
		if !ok {
			e.logger.Warn("phase not found for task", "phase", a.PhaseName, "task", a.Title)
			continue
		}
		task := &model.Task{
			ProjectID:   projectID,
			PhaseID:     &phaseID,
			Title:       a.Title,
			Description: a.Description,
			Priority:    a.Priority,
			Status:      model.TaskStatusPending,
		}
		if err := e.services.Task.Create(ctx, task); err != nil {
			e.logger.Error("create task failed", "title", a.Title, "error", err)
			return err
		}
	}
	return nil
}

func (e *Engine) processContractActions(ctx context.Context, taskID string, actions []runtime.ContractAction) {
	for _, a := range actions {
		nonGoals, _ := json.Marshal(a.NonGoals)
		doneDef, _ := json.Marshal(a.DoneDefinition)
		verSteps, _ := json.Marshal(a.VerificationSteps)
		accCrit, _ := json.Marshal(a.AcceptanceCriteria)

		contract := &model.TaskContract{
			TaskID:             taskID,
			Scope:              a.Scope,
			NonGoals:           model.JSON(nonGoals),
			DoneDefinition:     model.JSON(doneDef),
			VerificationPlan:   model.JSON(verSteps),
			AcceptanceCriteria: model.JSON(accCrit),
		}
		if err := e.services.Contract.Create(ctx, contract); err != nil {
			e.logger.Error("create contract failed", "task", taskID, "error", err)
		}
	}
}

func (e *Engine) processAssignmentActions(ctx context.Context, taskID, assignedBy string, actions []runtime.AssignmentAction) {
	for _, a := range actions {
		workerAgent, err := e.findAgent(ctx, model.AgentRole(a.AgentRole))
		if err != nil {
			e.logger.Error("find agent for assignment", "role", a.AgentRole, "error", err)
			continue
		}
		assignment := &model.TaskAssignment{
			TaskID:     taskID,
			AgentID:    workerAgent.ID,
			AssignedBy: &assignedBy,
			Role:       model.AssignmentRole(a.Role),
			Note:       a.Note,
		}
		if err := e.services.Assignment.Create(ctx, assignment); err != nil {
			e.logger.Error("create assignment failed", "task", taskID, "error", err)
		}
	}
}

func (e *Engine) processArtifactActions(ctx context.Context, projectID, taskID, agentID string, actions []runtime.ArtifactAction) {
	for _, a := range actions {
		artifact := &model.Artifact{
			ProjectID:    projectID,
			TaskID:       &taskID,
			Name:         a.Name,
			ArtifactType: model.ArtifactType(a.ArtifactType),
			Description:  a.Description,
			CreatedBy:    &agentID,
		}
		if err := e.services.Artifact.Create(ctx, artifact); err != nil {
			e.logger.Error("create artifact failed", "name", a.Name, "error", err)
			continue
		}

		metaBytes, _ := json.Marshal(a.Metadata)
		version := &model.ArtifactVersion{
			ArtifactID:    artifact.ID,
			URI:           a.URI,
			ContentType:   a.ContentType,
			SizeBytes:     a.SizeBytes,
			Checksum:      fmt.Sprintf("%x", a.Content),
			ChangeSummary: "初始版本（Mock 生成）",
			CreatedBy:     &agentID,
			Metadata:      model.JSON(metaBytes),
		}
		if err := e.services.Artifact.AddVersion(ctx, version); err != nil {
			e.logger.Error("add artifact version failed", "artifact", artifact.ID, "error", err)
		}
	}
}

func (e *Engine) processHandoffActions(ctx context.Context, taskID, agentID string, actions []runtime.HandoffAction) {
	for _, a := range actions {
		completedItems, _ := json.Marshal(a.CompletedItems)
		pendingItems, _ := json.Marshal(a.PendingItems)
		risks, _ := json.Marshal(a.Risks)
		nextSteps, _ := json.Marshal(a.NextSteps)

		snapshot := &model.HandoffSnapshot{
			TaskID:         taskID,
			AgentID:        agentID,
			Summary:        a.Summary,
			CompletedItems: model.JSON(completedItems),
			PendingItems:   model.JSON(pendingItems),
			Risks:          model.JSON(risks),
			NextSteps:      model.JSON(nextSteps),
		}
		if err := e.services.Handoff.Create(ctx, snapshot); err != nil {
			e.logger.Error("create handoff failed", "task", taskID, "error", err)
		}
	}
}

func (e *Engine) processReviewActions(ctx context.Context, taskID, reviewerID string, actions []runtime.ReviewAction) {
	for _, a := range actions {
		findings, _ := json.Marshal(a.Findings)
		recommendations, _ := json.Marshal(a.Recommendations)

		report := &model.ReviewReport{
			TaskID:          taskID,
			ReviewerID:      reviewerID,
			Verdict:         model.ReviewVerdict(a.Verdict),
			Summary:         a.Summary,
			Findings:        model.JSON(findings),
			Recommendations: model.JSON(recommendations),
		}
		if err := e.services.Review.Create(ctx, report); err != nil {
			e.logger.Error("create review failed", "task", taskID, "error", err)
		}
	}
}

func (e *Engine) processTransitionActions(ctx context.Context, task *model.Task, actions []runtime.TransitionAction) {
	for _, a := range actions {
		newStatus := model.TaskStatus(a.NewStatus)
		if err := e.services.Task.TransitionStatus(ctx, task.ID, newStatus); err != nil {
			e.logger.Warn("transition failed", "task", task.Title, "to", a.NewStatus, "error", err)
		}
	}
}

// findAgent locates the first active agent with the given role.
func (e *Engine) findAgent(ctx context.Context, role model.AgentRole) (*model.Agent, error) {
	agents, _, err := e.services.Agent.List(ctx, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	for _, a := range agents {
		if a.Role == role && a.Status == model.AgentStatusActive {
			return a, nil
		}
	}
	return nil, fmt.Errorf("no active agent with role %q found", role)
}

func (e *Engine) failRun(ctx context.Context, run *WorkflowRun, err error) {
	now := time.Now()
	run.Status = RunStatusFailed
	run.Error = err.Error()
	run.CompletedAt = &now
	e.store.Save(run)

	e.services.Audit.LogEvent(ctx, "system", "", "workflow.failed",
		fmt.Sprintf("工作流失败: %s", err.Error()),
		"project", run.ProjectID, nil, map[string]string{"run_id": run.ID, "error": err.Error()})
}


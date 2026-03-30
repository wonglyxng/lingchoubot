package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
	"github.com/lingchou/lingchoubot/backend/internal/runtime"
	"github.com/lingchou/lingchoubot/backend/internal/service"
)

// Services bundles all the service dependencies the engine needs.
type Services struct {
	Project    *service.ProjectService
	Phase      *service.PhaseService
	Agent      *service.AgentService
	Task       *service.TaskService
	Contract   *service.TaskContractService
	Assignment *service.TaskAssignmentService
	Artifact   *service.ArtifactService
	Handoff    *service.HandoffSnapshotService
	Review     *service.ReviewReportService
	Approval   *service.ApprovalRequestService
	Audit      *service.AuditService
}

// runCtx holds per-execution state during a workflow run.
type runCtx struct {
	run       *model.WorkflowRun
	stepCount int
}

func (rc *runCtx) nextOrder() int {
	rc.stepCount++
	return rc.stepCount
}

// Engine orchestrates the PM → Supervisor → Worker → Reviewer chain.
type Engine struct {
	registry *runtime.Registry
	services *Services
	workflow *service.WorkflowService
	logger   *slog.Logger
}

func NewEngine(reg *runtime.Registry, svc *Services, workflow *service.WorkflowService, logger *slog.Logger) *Engine {
	return &Engine{registry: reg, services: svc, workflow: workflow, logger: logger}
}

// GetRun loads a workflow run with its steps.
func (e *Engine) GetRun(ctx context.Context, id string) (*model.WorkflowRun, error) {
	return e.workflow.GetRun(ctx, id)
}

// ListRuns returns paginated workflow runs.
func (e *Engine) ListRuns(ctx context.Context, p repository.WorkflowRunListParams) ([]*model.WorkflowRun, int, error) {
	return e.workflow.ListRuns(ctx, p)
}

// CancelRun marks a workflow run as cancelled.
// For the local engine, this updates the DB record; the goroutine will not be interrupted
// but will notice the status on next step boundary.
func (e *Engine) CancelRun(ctx context.Context, id string) error {
	run, err := e.workflow.GetRun(ctx, id)
	if err != nil {
		return fmt.Errorf("get run: %w", err)
	}
	if run == nil {
		return fmt.Errorf("run %s not found", id)
	}
	if run.Status != model.WorkflowRunRunning {
		return fmt.Errorf("run %s is not running (status=%s)", id, run.Status)
	}
	return e.workflow.CancelRun(ctx, run)
}

// Run executes the full workflow for a project: PM → Supervisor → Worker → Reviewer.
// This is a synchronous call — use RunAsync for non-blocking execution.
func (e *Engine) Run(ctx context.Context, projectID string) (*model.WorkflowRun, error) {
	proj, err := e.services.Project.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	if proj == nil {
		return nil, fmt.Errorf("project %s not found", projectID)
	}

	run, err := e.workflow.CreateRun(ctx, proj.ID)
	if err != nil {
		return nil, fmt.Errorf("create workflow run: %w", err)
	}
	rc := &runCtx{run: run}

	e.services.Audit.LogEvent(ctx, "system", "", "workflow.started",
		fmt.Sprintf("项目「%s」工作流已启动", proj.Name),
		"project", proj.ID, nil, map[string]string{"run_id": run.ID})

	if err := e.runPMPhase(ctx, rc, proj); err != nil {
		e.failRun(ctx, rc, err)
		return e.workflow.GetRun(ctx, run.ID)
	}

	phases, err := e.services.Phase.ListByProject(ctx, proj.ID)
	if err != nil {
		e.failRun(ctx, rc, err)
		return e.workflow.GetRun(ctx, run.ID)
	}

	for _, phase := range phases {
		if err := e.runPhase(ctx, rc, proj, phase); err != nil {
			e.logger.Error("phase failed", "phase", phase.Name, "error", err)
			continue
		}
	}

	summary := fmt.Sprintf("项目「%s」工作流完成：%d 个阶段已处理，共 %d 步",
		proj.Name, len(phases), rc.stepCount)
	if err := e.workflow.CompleteRun(ctx, run, summary); err != nil {
		e.logger.Error("complete run failed", "error", err)
	}

	e.services.Audit.LogEvent(ctx, "system", "", "workflow.completed",
		summary, "project", proj.ID, nil, map[string]string{"run_id": run.ID})

	return e.workflow.GetRun(ctx, run.ID)
}

// RunAsync starts a workflow in the background and returns the run immediately.
// The caller can poll GetRun to check progress. Uses context.Background() to
// decouple the workflow lifetime from the HTTP request.
func (e *Engine) RunAsync(ctx context.Context, projectID string) (*model.WorkflowRun, error) {
	proj, err := e.services.Project.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	if proj == nil {
		return nil, fmt.Errorf("project %s not found", projectID)
	}

	run, err := e.workflow.CreateRun(ctx, proj.ID)
	if err != nil {
		return nil, fmt.Errorf("create workflow run: %w", err)
	}

	go func() {
		bgCtx := context.Background()
		rc := &runCtx{run: run}

		e.services.Audit.LogEvent(bgCtx, "system", "", "workflow.started",
			fmt.Sprintf("项目「%s」工作流已启动（异步）", proj.Name),
			"project", proj.ID, nil, map[string]string{"run_id": run.ID})

		if err := e.runPMPhase(bgCtx, rc, proj); err != nil {
			e.failRun(bgCtx, rc, err)
			return
		}

		phases, err := e.services.Phase.ListByProject(bgCtx, proj.ID)
		if err != nil {
			e.failRun(bgCtx, rc, err)
			return
		}

		for _, phase := range phases {
			if err := e.runPhase(bgCtx, rc, proj, phase); err != nil {
				e.logger.Error("phase failed", "phase", phase.Name, "error", err)
				continue
			}
		}

		summary := fmt.Sprintf("项目「%s」工作流完成：%d 个阶段已处理，共 %d 步",
			proj.Name, len(phases), rc.stepCount)
		if err := e.workflow.CompleteRun(bgCtx, run, summary); err != nil {
			e.logger.Error("complete run failed", "error", err)
		}

		e.services.Audit.LogEvent(bgCtx, "system", "", "workflow.completed",
			summary, "project", proj.ID, nil, map[string]string{"run_id": run.ID})
	}()

	return run, nil
}

// runPMPhase runs the PM agent to decompose the project.
func (e *Engine) runPMPhase(ctx context.Context, rc *runCtx, proj *model.Project) error {
	step, err := e.workflow.AddStep(ctx, rc.run.ID, "PM 项目分解", "pm", rc.nextOrder())
	if err != nil {
		return err
	}
	if err := e.workflow.StartStep(ctx, step); err != nil {
		return err
	}

	pmAgent, err := e.findAgent(ctx, model.AgentRolePM)
	if err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}
	step.AgentID = &pmAgent.ID

	runner, err := e.registry.Get("pm")
	if err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}

	input := &runtime.AgentTaskInput{
		RunID:       rc.run.ID,
		AgentID:     pmAgent.ID,
		AgentRole:   "pm",
		Instruction: fmt.Sprintf("分解项目「%s」为阶段和任务", proj.Name),
		Project:     &runtime.ProjectCtx{ID: proj.ID, Name: proj.Name, Description: proj.Description},
	}

	output, err := runner.Execute(input)
	if err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return fmt.Errorf("PM agent execution failed: %w", err)
	}
	if output.Status == runtime.OutputStatusFailed {
		_ = e.workflow.FailStep(ctx, step, output.Error)
		return fmt.Errorf("PM agent returned failure: %s", output.Error)
	}

	if err := e.processPhaseActions(ctx, proj.ID, output.Phases); err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}

	phases, _ := e.services.Phase.ListByProject(ctx, proj.ID)
	phaseMap := make(map[string]string)
	for _, p := range phases {
		phaseMap[p.Name] = p.ID
	}
	if err := e.processTaskActions(ctx, proj.ID, phaseMap, output.Tasks); err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}

	_ = e.workflow.CompleteStep(ctx, step, output.Summary)
	return nil
}

// runPhase processes all tasks within a phase.
func (e *Engine) runPhase(ctx context.Context, rc *runCtx, proj *model.Project, phase *model.Phase) error {
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
		if err := e.runTaskChain(ctx, rc, proj, phase, task); err != nil {
			e.logger.Error("task chain failed", "task", task.Title, "error", err)
		}
	}
	return nil
}

// runTaskChain executes Supervisor → Worker → Reviewer for a single task.
// If review returns needs_revision, the chain loops back to the supervisor for rework.
const maxReworkAttempts = 3

func (e *Engine) runTaskChain(ctx context.Context, rc *runCtx, proj *model.Project, phase *model.Phase, task *model.Task) error {
	if err := e.runSupervisor(ctx, rc, proj, phase, task); err != nil {
		return err
	}

	for attempt := 0; attempt <= maxReworkAttempts; attempt++ {
		if err := e.runWorker(ctx, rc, proj, phase, task); err != nil {
			return err
		}

		if err := e.runReviewer(ctx, rc, proj, phase, task); err != nil {
			return err
		}

		// Reload task status after review
		fresh, err := e.services.Task.GetByID(ctx, task.ID)
		if err != nil {
			return fmt.Errorf("reload task after review: %w", err)
		}
		if fresh == nil {
			return fmt.Errorf("task %s disappeared after review", task.ID)
		}
		*task = *fresh

		if task.Status != model.TaskStatusRevisionRequired {
			return nil // completed or other terminal state
		}

		// Rework: route back to owner supervisor
		e.logger.Info("rework triggered", "task", task.Title, "attempt", attempt+1)
		e.services.Audit.LogEvent(ctx, "system", "", "task.rework",
			fmt.Sprintf("任务「%s」评审打回，第 %d 次返工，回到责任主管", task.Title, attempt+1),
			"task", task.ID, nil, map[string]string{
				"attempt":             fmt.Sprintf("%d", attempt+1),
				"owner_supervisor_id": stringOrEmpty(task.OwnerSupervisorID),
			})

		// Transition back to assigned for the rework cycle
		_ = e.services.Task.TransitionStatus(ctx, task.ID, model.TaskStatusInProgress)
	}

	e.logger.Warn("max rework attempts reached", "task", task.Title)
	return fmt.Errorf("task %q exceeded max rework attempts (%d)", task.Title, maxReworkAttempts)
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (e *Engine) runSupervisor(ctx context.Context, rc *runCtx, proj *model.Project, phase *model.Phase, task *model.Task) error {
	// Infer and persist execution domain if not already set
	domain := inferExecutionDomain(task)
	if task.ExecutionDomain != domain {
		task.ExecutionDomain = domain
	}

	step, err := e.workflow.AddStep(ctx, rc.run.ID, fmt.Sprintf("主管规划「%s」(%s)", task.Title, domain), "supervisor", rc.nextOrder())
	if err != nil {
		return err
	}
	if err := e.workflow.StartStep(ctx, step); err != nil {
		return err
	}
	step.TaskID = &task.ID
	step.PhaseID = &phase.ID

	// Route to the correct supervisor by domain
	agent, err := e.findSupervisorByDomain(ctx, domain)
	if err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}
	step.AgentID = &agent.ID

	// Record owner supervisor on task
	task.OwnerSupervisorID = &agent.ID
	_ = e.services.Task.Update(ctx, task)

	runner, err := e.registry.Get("supervisor")
	if err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}

	input := &runtime.AgentTaskInput{
		RunID:       rc.run.ID,
		AgentID:     agent.ID,
		AgentRole:   "supervisor",
		Instruction: fmt.Sprintf("为任务「%s」创建契约并分派执行者", task.Title),
		Project:     &runtime.ProjectCtx{ID: proj.ID, Name: proj.Name, Description: proj.Description},
		Phase:       &runtime.PhaseCtx{ID: phase.ID, ProjectID: proj.ID, Name: phase.Name, Description: phase.Description, SortOrder: phase.SortOrder},
		Task:        &runtime.TaskCtx{ID: task.ID, ProjectID: proj.ID, PhaseID: phase.ID, Title: task.Title, Description: task.Description, Priority: task.Priority},
	}

	output, err := runner.Execute(input)
	if err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}
	if output.Status == runtime.OutputStatusFailed {
		_ = e.workflow.FailStep(ctx, step, output.Error)
		return fmt.Errorf("supervisor failed: %s", output.Error)
	}

	e.processContractActions(ctx, task.ID, output.Contracts)
	e.processAssignmentActions(ctx, task.ID, agent.ID, output.Assignments)
	e.processTransitionActions(ctx, task, output.Transitions)

	_ = e.workflow.CompleteStep(ctx, step, output.Summary)
	return nil
}

func (e *Engine) runWorker(ctx context.Context, rc *runCtx, proj *model.Project, phase *model.Phase, task *model.Task) error {
	spec := inferSpecialization(task)
	stepName := fmt.Sprintf("执行「%s」", task.Title)
	if spec != model.AgentSpecGeneral {
		stepName = fmt.Sprintf("执行「%s」(%s)", task.Title, spec)
	}

	step, err := e.workflow.AddStep(ctx, rc.run.ID, stepName, "worker", rc.nextOrder())
	if err != nil {
		return err
	}
	if err := e.workflow.StartStep(ctx, step); err != nil {
		return err
	}
	step.TaskID = &task.ID

	agent, err := e.findAgentWithSpec(ctx, model.AgentRoleWorker, spec)
	if err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}
	step.AgentID = &agent.ID

	// Transition to in_progress
	_ = e.services.Task.TransitionStatus(ctx, task.ID, model.TaskStatusInProgress)

	runner, err := e.registry.GetForSpec("worker", string(agent.Specialization))
	if err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
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
		RunID:       rc.run.ID,
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
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}
	if output.Status == runtime.OutputStatusFailed {
		_ = e.workflow.FailStep(ctx, step, output.Error)
		return fmt.Errorf("worker failed: %s", output.Error)
	}

	e.processArtifactActions(ctx, proj.ID, task.ID, agent.ID, output.Artifacts)
	e.processHandoffActions(ctx, task.ID, agent.ID, output.Handoffs)
	e.processTransitionActions(ctx, task, output.Transitions)

	_ = e.workflow.CompleteStep(ctx, step, output.Summary)
	return nil
}

func (e *Engine) runReviewer(ctx context.Context, rc *runCtx, proj *model.Project, phase *model.Phase, task *model.Task) error {
	step, err := e.workflow.AddStep(ctx, rc.run.ID, fmt.Sprintf("评审「%s」", task.Title), "reviewer", rc.nextOrder())
	if err != nil {
		return err
	}
	if err := e.workflow.StartStep(ctx, step); err != nil {
		return err
	}
	step.TaskID = &task.ID

	agent, err := e.findAgent(ctx, model.AgentRoleReviewer)
	if err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}
	step.AgentID = &agent.ID

	runner, err := e.registry.Get("reviewer")
	if err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}

	input := &runtime.AgentTaskInput{
		RunID:       rc.run.ID,
		AgentID:     agent.ID,
		AgentRole:   "reviewer",
		Instruction: fmt.Sprintf("评审任务「%s」的交付物", task.Title),
		Project:     &runtime.ProjectCtx{ID: proj.ID, Name: proj.Name, Description: proj.Description},
		Task:        &runtime.TaskCtx{ID: task.ID, ProjectID: proj.ID, PhaseID: phase.ID, Title: task.Title, Description: task.Description, Priority: task.Priority},
	}

	output, err := runner.Execute(input)
	if err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}
	if output.Status == runtime.OutputStatusFailed {
		_ = e.workflow.FailStep(ctx, step, output.Error)
		return fmt.Errorf("reviewer failed: %s", output.Error)
	}

	e.processReviewActions(ctx, task.ID, agent.ID, output.Reviews)

	_ = e.workflow.CompleteStep(ctx, step, output.Summary)
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
		// Infer execution domain from task content
		task.ExecutionDomain = inferExecutionDomain(task)
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
		workerAgent, err := e.findAgentWithSpec(ctx, model.AgentRole(a.AgentRole), model.AgentSpecGeneral)
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
	return e.findAgentWithSpec(ctx, role, model.AgentSpecGeneral)
}

// findAgentWithSpec locates an active agent matching role + specialization.
// Prefers exact specialization match; falls back to "general".
func (e *Engine) findAgentWithSpec(ctx context.Context, role model.AgentRole, spec model.AgentSpecialization) (*model.Agent, error) {
	agent, err := e.services.Agent.FindByRoleAndSpec(ctx, role, spec)
	if err != nil {
		return nil, fmt.Errorf("find agent (%s/%s): %w", role, spec, err)
	}
	if agent != nil {
		return agent, nil
	}

	// Fallback: scan all agents (backward compatibility for unspecialized setups)
	agents, _, err := e.services.Agent.List(ctx, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	for _, a := range agents {
		if a.Role == role && a.Status == model.AgentStatusActive {
			return a, nil
		}
	}
	return nil, fmt.Errorf("no active agent with role %q (specialization %q) found", role, spec)
}

// findSupervisorByDomain locates the correct supervisor for a task's execution domain.
// Falls back to findAgent(supervisor) for backward compatibility.
func (e *Engine) findSupervisorByDomain(ctx context.Context, domain model.ExecutionDomain) (*model.Agent, error) {
	var roleCode model.RoleCode
	switch domain {
	case model.ExecDomainDevelopment:
		roleCode = model.RoleCodeDevelopmentSupervisor
	case model.ExecDomainQA:
		roleCode = model.RoleCodeQASupervisor
	default:
		// general → fall back to any supervisor
		return e.findAgent(ctx, model.AgentRoleSupervisor)
	}
	agent, err := e.services.Agent.FindByRoleCode(ctx, roleCode)
	if err != nil {
		return nil, fmt.Errorf("find supervisor by domain (%s): %w", domain, err)
	}
	if agent != nil {
		return agent, nil
	}
	// Fallback: any supervisor
	e.logger.Warn("no supervisor with role_code, falling back", "role_code", roleCode)
	return e.findAgent(ctx, model.AgentRoleSupervisor)
}

// inferExecutionDomain determines the task's execution domain from its content.
// If the task already has a non-empty domain set, it is preserved.
func inferExecutionDomain(task *model.Task) model.ExecutionDomain {
	if task.ExecutionDomain != "" && task.ExecutionDomain != model.ExecDomainGeneral {
		return task.ExecutionDomain
	}
	combined := task.Title + " " + task.Description

	qaKeywords := []string{"测试", "验证", "回归", "QA", "test", "质量", "评审"}
	for _, w := range qaKeywords {
		if containsCI(combined, w) {
			return model.ExecDomainQA
		}
	}

	devKeywords := []string{"API", "接口", "后端", "前端", "页面", "组件", "数据库",
		"migration", "服务端", "handler", "repository", "React", "Next.js", "实现", "开发"}
	for _, w := range devKeywords {
		if containsCI(combined, w) {
			return model.ExecDomainDevelopment
		}
	}

	return model.ExecDomainGeneral
}

// inferSpecialization determines the best worker specialization based on task content.
func inferSpecialization(task *model.Task) model.AgentSpecialization {
	title := task.Title
	desc := task.Description
	combined := title + " " + desc

	keywords := map[model.AgentSpecialization][]string{
		model.AgentSpecBackend:  {"API", "接口", "后端", "数据库", "migration", "服务端", "handler", "repository"},
		model.AgentSpecFrontend: {"前端", "页面", "组件", "UI", "React", "Next.js", "样式", "布局"},
		model.AgentSpecQA:       {"测试", "验证", "回归", "QA", "test", "质量"},
		model.AgentSpecRelease:  {"发布", "部署", "release", "deploy", "上线"},
		model.AgentSpecDevOps:   {"CI", "CD", "Docker", "Kubernetes", "基础设施", "监控"},
	}

	for spec, words := range keywords {
		for _, w := range words {
			if containsCI(combined, w) {
				return spec
			}
		}
	}
	return model.AgentSpecGeneral
}

// containsCI performs a case-insensitive substring check.
func containsCI(s, substr string) bool {
	sl := len(s)
	subl := len(substr)
	if subl > sl {
		return false
	}
	for i := 0; i <= sl-subl; i++ {
		if equalFoldSlice(s[i:i+subl], substr) {
			return true
		}
	}
	return false
}

func equalFoldSlice(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func (e *Engine) failRun(ctx context.Context, rc *runCtx, err error) {
	if fErr := e.workflow.FailRun(ctx, rc.run, err.Error()); fErr != nil {
		e.logger.Error("fail run persistence error", "error", fErr)
	}

	e.services.Audit.LogEvent(ctx, "system", "", "workflow.failed",
		fmt.Sprintf("工作流失败: %s", err.Error()),
		"project", rc.run.ProjectID, nil, map[string]string{"run_id": rc.run.ID, "error": err.Error()})
}

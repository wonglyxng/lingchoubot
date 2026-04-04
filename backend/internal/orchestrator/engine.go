package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
	"github.com/lingchou/lingchoubot/backend/internal/reviewpolicy"
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

var errPhaseWaitingApproval = errors.New("phase waiting approval")

type manualInterventionError struct {
	reasonCode model.ManualInterventionReasonCode
	agentRole  string
	phaseID    string
	phaseName  string
	taskID     string
	taskTitle  string
	reason     string
}

func (e *manualInterventionError) Error() string {
	if e == nil {
		return ""
	}
	return e.reason
}

func newManualInterventionError(reasonCode model.ManualInterventionReasonCode, agentRole, phaseID, phaseName, taskID, taskTitle, reason string) error {
	return &manualInterventionError{
		reasonCode: reasonCode,
		agentRole:  agentRole,
		phaseID:    phaseID,
		phaseName:  phaseName,
		taskID:     taskID,
		taskTitle:  taskTitle,
		reason:     reason,
	}
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

// ResumeRun continues a workflow that is currently waiting for approvals or manual intervention.
func (e *Engine) ResumeRun(ctx context.Context, id string) error {
	run, err := e.workflow.GetRun(ctx, id)
	if err != nil {
		return fmt.Errorf("get run: %w", err)
	}
	if run == nil {
		return fmt.Errorf("run %s not found", id)
	}
	previousStatus := run.Status
	if run.Status != model.WorkflowRunWaitingApproval && run.Status != model.WorkflowRunWaitingManual {
		return fmt.Errorf("run %s is not resumable (status=%s)", id, run.Status)
	}

	proj, err := e.services.Project.GetByID(ctx, run.ProjectID)
	if err != nil {
		return fmt.Errorf("load project: %w", err)
	}
	if proj == nil {
		return fmt.Errorf("project %s not found", run.ProjectID)
	}

	resumeSummary := fmt.Sprintf("项目「%s」审批已收口，工作流继续执行", proj.Name)
	if previousStatus == model.WorkflowRunWaitingManual {
		resumeSummary = fmt.Sprintf("项目「%s」人工介入已处理，工作流继续执行", proj.Name)
	}
	if err := e.workflow.ResumeRun(ctx, run, resumeSummary); err != nil {
		return fmt.Errorf("resume workflow run: %w", err)
	}
	e.services.Audit.LogEvent(ctx, "system", "", "workflow.resumed",
		fmt.Sprintf("项目「%s」工作流已恢复执行", proj.Name),
		"project", proj.ID, nil, map[string]string{"run_id": run.ID, "previous_status": string(previousStatus)})

	stepCount := len(run.Steps)
	go func() {
		bgCtx := context.Background()
		rc := &runCtx{run: run, stepCount: stepCount}
		if shouldResumeFromPM(run) {
			if err := e.runPMPhase(bgCtx, rc, proj); err != nil {
				if e.handleRunInterruption(bgCtx, rc, proj, err) {
					return
				}
				e.failRun(bgCtx, rc, err)
				return
			}
		}
		if err := e.continueRun(bgCtx, rc, proj); err != nil {
			if e.handleRunInterruption(bgCtx, rc, proj, err) {
				return
			}
			e.failRun(bgCtx, rc, err)
		}
	}()

	return nil
}

// ResolveManualIntervention applies a human decision for a waiting_manual_intervention run.
func (e *Engine) ResolveManualIntervention(ctx context.Context, id string, action model.ManualInterventionAction, note string) error {
	run, err := e.workflow.GetRun(ctx, id)
	if err != nil {
		return fmt.Errorf("get run: %w", err)
	}
	if run == nil {
		return fmt.Errorf("run %s not found", id)
	}
	if run.Status != model.WorkflowRunWaitingManual {
		return fmt.Errorf("run %s is not waiting for manual intervention (status=%s)", id, run.Status)
	}

	intervention, err := decodeWorkflowManualIntervention(run.Metadata)
	if err != nil {
		return fmt.Errorf("decode manual intervention: %w", err)
	}
	if intervention == nil {
		return fmt.Errorf("run %s has no manual intervention context", id)
	}

	switch action {
	case model.ManualInterventionActionEscalateToApproval:
		if intervention.ReasonCode != model.ManualInterventionReasonReworkLimitReached {
			return fmt.Errorf("manual action %q is not allowed for reason %q", action, intervention.ReasonCode)
		}
		if intervention.TaskID == "" {
			return fmt.Errorf("manual intervention task context is missing")
		}
		if err := e.escalateTaskToApproval(ctx, run, intervention, note); err != nil {
			return err
		}
		return e.ResumeRun(ctx, id)
	default:
		return fmt.Errorf("unsupported manual intervention action %q", action)
	}
}

// Run executes the full workflow for a project: PM → Supervisor → Worker → Reviewer.
// This is a synchronous call — use RunAsync for non-blocking execution.
func (e *Engine) Run(ctx context.Context, projectID string) (*model.WorkflowRun, error) {
	proj, err := validateWorkflowStartPreconditions(ctx, e.services, projectID)
	if err != nil {
		return nil, err
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
		if !e.handleRunInterruption(ctx, rc, proj, err) {
			e.failRun(ctx, rc, err)
		}
		return e.workflow.GetRun(ctx, run.ID)
	}

	if err := e.continueRun(ctx, rc, proj); err != nil {
		if !e.handleRunInterruption(ctx, rc, proj, err) {
			e.failRun(ctx, rc, err)
		}
		return e.workflow.GetRun(ctx, run.ID)
	}

	return e.workflow.GetRun(ctx, run.ID)
}

// RunAsync starts a workflow in the background and returns the run immediately.
// The caller can poll GetRun to check progress. Uses context.Background() to
// decouple the workflow lifetime from the HTTP request.
func (e *Engine) RunAsync(ctx context.Context, projectID string) (*model.WorkflowRun, error) {
	proj, err := validateWorkflowStartPreconditions(ctx, e.services, projectID)
	if err != nil {
		return nil, err
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
			if e.handleRunInterruption(bgCtx, rc, proj, err) {
				return
			}
			e.failRun(bgCtx, rc, err)
			return
		}

		if err := e.continueRun(bgCtx, rc, proj); err != nil {
			if e.handleRunInterruption(bgCtx, rc, proj, err) {
				return
			}
			e.failRun(bgCtx, rc, err)
			return
		}
	}()

	return run, nil
}

func (e *Engine) continueRun(ctx context.Context, rc *runCtx, proj *model.Project) error {
	phases, err := e.services.Phase.ListByProject(ctx, proj.ID)
	if err != nil {
		return err
	}

	for _, phase := range phases {
		tasks, _, err := e.services.Task.List(ctx, repository.TaskListParams{
			PhaseID: phase.ID,
			Limit:   100,
			Offset:  0,
		})
		if err != nil {
			return err
		}
		if len(tasks) == 0 {
			if err := e.setPhaseStatus(ctx, phase, model.PhaseStatusCompleted); err != nil {
				return err
			}
			continue
		}
		if phaseTasksCompleted(tasks) {
			if err := e.setPhaseStatus(ctx, phase, model.PhaseStatusCompleted); err != nil {
				return err
			}
			continue
		}

		if err := e.setPhaseStatus(ctx, phase, model.PhaseStatusActive); err != nil {
			return err
		}
		if err := e.runPhase(ctx, rc, proj, phase); err != nil {
			if errors.Is(err, errPhaseWaitingApproval) {
				summary := fmt.Sprintf("项目「%s」当前停在阶段「%s」，等待审批收口后继续", proj.Name, phase.Name)
				if waitErr := e.workflow.WaitForApproval(ctx, rc.run, summary); waitErr != nil {
					return fmt.Errorf("mark run waiting approval: %w", waitErr)
				}
				e.services.Audit.LogEvent(ctx, "system", "", "workflow.waiting_approval",
					summary, "project", proj.ID, nil, map[string]string{"run_id": rc.run.ID, "phase_id": phase.ID})
				return err
			}
			return err
		}
		if err := e.setPhaseStatus(ctx, phase, model.PhaseStatusCompleted); err != nil {
			return err
		}
	}

	summary := fmt.Sprintf("项目「%s」工作流完成：%d 个阶段已处理，共 %d 步",
		proj.Name, len(phases), rc.stepCount)
	if err := e.workflow.CompleteRun(ctx, rc.run, summary); err != nil {
		e.logger.Error("complete run failed", "error", err)
	}
	e.services.Audit.LogEvent(ctx, "system", "", "workflow.completed",
		summary, "project", proj.ID, nil, map[string]string{"run_id": rc.run.ID})
	return nil
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
	agentLLM, err := runtimeAgentLLMConfig(pmAgent)
	if err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}

	input := &runtime.AgentTaskInput{
		RunID:       rc.run.ID,
		AgentID:     pmAgent.ID,
		AgentRole:   "pm",
		AgentLLM:    agentLLM,
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
		return newManualInterventionError(model.ManualInterventionReasonLLMExecutionFailed, "pm", "", "", "", "", output.Error)
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

	waitingApproval := false

	for _, task := range tasks {
		if task.Status == model.TaskStatusCompleted {
			continue
		}
		if task.Status == model.TaskStatusPendingApproval {
			waitingApproval = true
			continue
		}
		if err := e.runTaskChain(ctx, rc, proj, phase, task); err != nil {
			e.logger.Error("task chain failed", "task", task.Title, "error", err)
			return err
		}
		fresh, err := e.services.Task.GetByID(ctx, task.ID)
		if err != nil {
			return fmt.Errorf("reload task after phase execution: %w", err)
		}
		if fresh != nil && fresh.Status == model.TaskStatusPendingApproval {
			waitingApproval = true
		}
	}
	if waitingApproval {
		return errPhaseWaitingApproval
	}
	return nil
}

func (e *Engine) setPhaseStatus(ctx context.Context, phase *model.Phase, status model.PhaseStatus) error {
	if phase.Status == status {
		return nil
	}
	updated := *phase
	updated.Status = status
	if err := e.services.Phase.Update(ctx, &updated); err != nil {
		return fmt.Errorf("update phase %s status to %s: %w", phase.ID, status, err)
	}
	phase.Status = status
	phase.UpdatedAt = updated.UpdatedAt
	return nil
}

func phaseTasksCompleted(tasks []*model.Task) bool {
	if len(tasks) == 0 {
		return true
	}
	for _, task := range tasks {
		if task.Status != model.TaskStatusCompleted {
			return false
		}
	}
	return true
}

// runTaskChain executes Supervisor → Worker → Reviewer for a single task.
// If review returns needs_revision, the chain loops back to the supervisor for rework.
const maxReworkAttempts = 3

func (e *Engine) runTaskChain(ctx context.Context, rc *runCtx, proj *model.Project, phase *model.Phase, task *model.Task) error {
	switch task.Status {
	case model.TaskStatusCompleted, model.TaskStatusPendingApproval:
		return nil
	case model.TaskStatusPending, model.TaskStatusRevisionRequired:
		if err := e.runSupervisor(ctx, rc, proj, phase, task); err != nil {
			return err
		}
		if err := e.reloadTask(ctx, task, "supervisor"); err != nil {
			return err
		}
		return e.runWorkerReviewerLoop(ctx, rc, proj, phase, task, false)
	case model.TaskStatusAssigned, model.TaskStatusInProgress:
		return e.runWorkerReviewerLoop(ctx, rc, proj, phase, task, false)
	case model.TaskStatusInReview:
		return e.runWorkerReviewerLoop(ctx, rc, proj, phase, task, true)
	default:
		return fmt.Errorf("task %q is not resumable from status %s", task.Title, task.Status)
	}
}

func (e *Engine) runWorkerReviewerLoop(ctx context.Context, rc *runCtx, proj *model.Project, phase *model.Phase, task *model.Task, startWithReviewer bool) error {
	for attempt := 0; attempt <= maxReworkAttempts; attempt++ {
		if !startWithReviewer {
			if err := e.runWorker(ctx, rc, proj, phase, task); err != nil {
				return err
			}
		}

		if err := e.runReviewer(ctx, rc, proj, phase, task); err != nil {
			return err
		}

		if err := e.reloadTask(ctx, task, "review"); err != nil {
			return err
		}

		if task.Status != model.TaskStatusRevisionRequired {
			return nil
		}

		reworkBrief := extractCurrentReworkBrief(task.Metadata)
		e.logger.Info("rework triggered", "task", task.Title, "attempt", attempt+1)
		e.services.Audit.LogEvent(ctx, "system", "", "task.rework",
			fmt.Sprintf("任务「%s」评审打回，第 %d 次返工，回到责任主管", task.Title, attempt+1),
			"task", task.ID, nil, map[string]any{
				"attempt":             attempt + 1,
				"owner_supervisor_id": stringOrEmpty(task.OwnerSupervisorID),
				"rework_brief":        reworkBrief,
			})

		if attempt == maxReworkAttempts {
			e.logger.Warn("max rework attempts reached", "task", task.Title)
			return newManualInterventionError(
				model.ManualInterventionReasonReworkLimitReached,
				"supervisor",
				phase.ID,
				phase.Name,
				task.ID,
				task.Title,
				fmt.Sprintf("任务「%s」超过最大返工次数（%d），需要人工介入", task.Title, maxReworkAttempts),
			)
		}

		if err := e.runSupervisor(ctx, rc, proj, phase, task); err != nil {
			return err
		}
		if err := e.reloadTask(ctx, task, "supervisor"); err != nil {
			return err
		}
		startWithReviewer = false
	}

	return nil
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
	agentLLM, err := runtimeAgentLLMConfig(agent)
	if err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}

	input := &runtime.AgentTaskInput{
		RunID:       rc.run.ID,
		AgentID:     agent.ID,
		AgentRole:   "supervisor",
		AgentLLM:    agentLLM,
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
		return newManualInterventionError(model.ManualInterventionReasonLLMExecutionFailed, "supervisor", phase.ID, phase.Name, task.ID, task.Title, output.Error)
	}

	if err := e.processContractActions(ctx, task, output.Contracts); err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}
	if err := e.processAssignmentActions(ctx, task, agent.ID, output.Assignments); err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}
	if err := e.processTransitionActions(ctx, task, output.Transitions); err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}

	_ = e.workflow.CompleteStep(ctx, step, output.Summary)
	return nil
}

func (e *Engine) runWorker(ctx context.Context, rc *runCtx, proj *model.Project, phase *model.Phase, task *model.Task) error {
	contractCtx, err := e.loadContractContext(ctx, task)
	if err != nil {
		return err
	}
	spec := resolveWorkerSpecialization(task, contractCtx)
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
	agentLLM, err := runtimeAgentLLMConfig(agent)
	if err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}

	input := &runtime.AgentTaskInput{
		RunID:       rc.run.ID,
		AgentID:     agent.ID,
		AgentRole:   "worker",
		AgentLLM:    agentLLM,
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
		return newManualInterventionError(model.ManualInterventionReasonLLMExecutionFailed, "worker", phase.ID, phase.Name, task.ID, task.Title, output.Error)
	}

	if err := e.processArtifactActions(ctx, proj.ID, task.ID, agent.ID, output.Artifacts); err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}
	if err := e.processHandoffActions(ctx, task.ID, agent.ID, output.Handoffs); err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}
	if err := e.processTransitionActions(ctx, task, output.Transitions); err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}

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
	agentLLM, err := runtimeAgentLLMConfig(agent)
	if err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}

	artifactCtxs, err := e.loadTaskArtifactContexts(ctx, task.ID)
	if err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}
	contractCtx, err := e.loadContractContext(ctx, task)
	if err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}

	input := &runtime.AgentTaskInput{
		RunID:       rc.run.ID,
		AgentID:     agent.ID,
		AgentRole:   "reviewer",
		AgentLLM:    agentLLM,
		Instruction: fmt.Sprintf("评审任务「%s」的交付物", task.Title),
		Project:     &runtime.ProjectCtx{ID: proj.ID, Name: proj.Name, Description: proj.Description},
		Phase:       &runtime.PhaseCtx{ID: phase.ID, ProjectID: proj.ID, Name: phase.Name, Description: phase.Description, SortOrder: phase.SortOrder},
		Task:        &runtime.TaskCtx{ID: task.ID, ProjectID: proj.ID, PhaseID: phase.ID, Title: task.Title, Description: task.Description, Priority: task.Priority},
		Contract:    contractCtx,
		Artifacts:   artifactCtxs,
	}

	output, err := runner.Execute(input)
	if err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}
	if output.Status == runtime.OutputStatusFailed {
		_ = e.workflow.FailStep(ctx, step, output.Error)
		return newManualInterventionError(model.ManualInterventionReasonLLMExecutionFailed, "reviewer", phase.ID, phase.Name, task.ID, task.Title, output.Error)
	}

	if err := e.processReviewActions(ctx, rc.run.ID, task.ID, agent.ID, artifactCtxs, contractCtx, output.Reviews); err != nil {
		_ = e.workflow.FailStep(ctx, step, err.Error())
		return err
	}

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

func (e *Engine) processContractActions(ctx context.Context, task *model.Task, actions []runtime.ContractAction) error {
	for _, a := range actions {
		nonGoals, _ := json.Marshal(a.NonGoals)
		doneDef, _ := json.Marshal(a.DoneDefinition)
		verSteps, _ := json.Marshal(a.VerificationSteps)
		accCrit, _ := json.Marshal(a.AcceptanceCriteria)
		metadata, err := buildContractMetadata(task, a)
		if err != nil {
			return fmt.Errorf("build contract metadata for task %s: %w", task.ID, err)
		}
		metadataBytes, _ := json.Marshal(metadata)

		contract := &model.TaskContract{
			TaskID:             task.ID,
			Scope:              a.Scope,
			NonGoals:           model.JSON(nonGoals),
			DoneDefinition:     model.JSON(doneDef),
			VerificationPlan:   model.JSON(verSteps),
			AcceptanceCriteria: model.JSON(accCrit),
			Metadata:           model.JSON(metadataBytes),
		}
		if err := e.services.Contract.Create(ctx, contract); err != nil {
			return fmt.Errorf("create contract for task %s: %w", task.ID, err)
		}
	}
	return nil
}

func (e *Engine) processAssignmentActions(ctx context.Context, task *model.Task, assignedBy string, actions []runtime.AssignmentAction) error {
	contractCtx, err := e.loadContractContext(ctx, task)
	if err != nil {
		return fmt.Errorf("load contract context for assignment routing: %w", err)
	}
	for _, a := range actions {
		spec := model.AgentSpecGeneral
		if model.AgentRole(a.AgentRole) == model.AgentRoleWorker {
			spec = resolveWorkerSpecialization(task, contractCtx)
		}
		workerAgent, err := e.findAgentWithSpec(ctx, model.AgentRole(a.AgentRole), spec)
		if err != nil {
			return fmt.Errorf("find agent for assignment (%s/%s): %w", a.AgentRole, spec, err)
		}
		assignment := &model.TaskAssignment{
			TaskID:     task.ID,
			AgentID:    workerAgent.ID,
			AssignedBy: &assignedBy,
			Role:       model.AssignmentRole(a.Role),
			Note:       a.Note,
		}
		if err := e.services.Assignment.Create(ctx, assignment); err != nil {
			return fmt.Errorf("create assignment for task %s: %w", task.ID, err)
		}
	}
	return nil
}

func (e *Engine) processArtifactActions(ctx context.Context, projectID, taskID, agentID string, actions []runtime.ArtifactAction) error {
	for _, a := range actions {
		metaBytes, _ := json.Marshal(a.Metadata)
		existing, err := e.findReusableArtifact(ctx, taskID, a)
		if err != nil {
			return fmt.Errorf("find reusable artifact %s: %w", a.Name, err)
		}
		if existing != nil {
			version := &model.ArtifactVersion{
				ArtifactID:    existing.ID,
				URI:           a.URI,
				ContentType:   a.ContentType,
				SizeBytes:     a.SizeBytes,
				ChangeSummary: "返工迭代版本（Agent 重新生成）",
				CreatedBy:     &agentID,
				Metadata:      model.JSON(metaBytes),
				Content:       a.Content,
				SourceName:    a.Name,
			}
			if err := e.services.Artifact.AddVersion(ctx, version); err != nil {
				return fmt.Errorf("add artifact version %s: %w", a.Name, err)
			}
			continue
		}

		artifact := &model.Artifact{
			ProjectID:    projectID,
			TaskID:       &taskID,
			Name:         a.Name,
			ArtifactType: model.ArtifactType(a.ArtifactType),
			Description:  a.Description,
			CreatedBy:    &agentID,
		}
		version := &model.ArtifactVersion{
			URI:           a.URI,
			ContentType:   a.ContentType,
			SizeBytes:     a.SizeBytes,
			ChangeSummary: "初始版本（Agent 生成）",
			CreatedBy:     &agentID,
			Metadata:      model.JSON(metaBytes),
			Content:       a.Content,
			SourceName:    a.Name,
		}
		if err := e.services.Artifact.CreateWithInitialVersion(ctx, artifact, version); err != nil {
			return fmt.Errorf("create artifact %s with initial version: %w", a.Name, err)
		}
	}
	return nil
}

func (e *Engine) findReusableArtifact(ctx context.Context, taskID string, action runtime.ArtifactAction) (*model.Artifact, error) {
	artifacts, _, err := e.services.Artifact.List(ctx, repository.ArtifactListParams{
		TaskID:       taskID,
		ArtifactType: action.ArtifactType,
		Limit:        100,
		Offset:       0,
	})
	if err != nil {
		return nil, err
	}
	for _, artifact := range artifacts {
		if artifact.Name == action.Name && string(artifact.ArtifactType) == action.ArtifactType {
			return artifact, nil
		}
	}
	return nil, nil
}

func (e *Engine) processHandoffActions(ctx context.Context, taskID, agentID string, actions []runtime.HandoffAction) error {
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
			return fmt.Errorf("create handoff for task %s: %w", taskID, err)
		}
	}
	return nil
}

func (e *Engine) processReviewActions(ctx context.Context, runID, taskID, reviewerID string, artifactCtxs []runtime.ArtifactCtx, contractCtx *runtime.ContractCtx, actions []runtime.ReviewAction) error {
	if len(actions) != 1 {
		return fmt.Errorf("reviewer must output exactly 1 review action, got %d", len(actions))
	}
	for _, a := range actions {
		findings, _ := json.Marshal(a.Findings)
		recommendations, _ := json.Marshal(a.Recommendations)
		metadata, artifactVersionID := buildReviewMetadata(artifactCtxs, contractCtx, a)
		metadataBytes, _ := json.Marshal(metadata)

		runIDCopy := runID
		report := &model.ReviewReport{
			RunID:             &runIDCopy,
			TaskID:            taskID,
			ReviewerID:        reviewerID,
			ArtifactVersionID: artifactVersionID,
			Verdict:           model.ReviewVerdict(a.Verdict),
			Summary:           a.Summary,
			Findings:          model.JSON(findings),
			Recommendations:   model.JSON(recommendations),
			Metadata:          model.JSON(metadataBytes),
		}
		if err := e.services.Review.Create(ctx, report); err != nil {
			return fmt.Errorf("create review for task %s: %w", taskID, err)
		}
	}
	return nil
}

func (e *Engine) loadTaskArtifactContexts(ctx context.Context, taskID string) ([]runtime.ArtifactCtx, error) {
	artifacts, _, err := e.services.Artifact.List(ctx, repository.ArtifactListParams{TaskID: taskID, Limit: 100, Offset: 0})
	if err != nil {
		return nil, fmt.Errorf("list task artifacts: %w", err)
	}
	contexts := make([]runtime.ArtifactCtx, 0, len(artifacts))
	for _, artifact := range artifacts {
		versions, err := e.services.Artifact.ListVersions(ctx, artifact.ID)
		if err != nil {
			return nil, fmt.Errorf("list artifact versions: %w", err)
		}
		if len(versions) == 0 {
			continue
		}
		latest := versions[0]
		contexts = append(contexts, runtime.ArtifactCtx{
			ID:           artifact.ID,
			VersionID:    latest.ID,
			Version:      latest.Version,
			Name:         artifact.Name,
			ArtifactType: string(artifact.ArtifactType),
			VersionURI:   latest.URI,
			ContentType:  latest.ContentType,
			Content:      extractInlineContent(latest.Metadata),
		})
	}
	return contexts, nil
}

func (e *Engine) loadContractContext(ctx context.Context, task *model.Task) (*runtime.ContractCtx, error) {
	contract, err := e.services.Contract.GetLatestByTaskID(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("load latest contract for task %s: %w", task.ID, err)
	}
	if contract == nil {
		return nil, nil
	}

	doneDefinition, err := jsonArrayToStringsSafe(contract.DoneDefinition)
	if err != nil {
		return nil, fmt.Errorf("parse contract done_definition: %w", err)
	}
	verificationPlan, err := jsonArrayToStringsSafe(contract.VerificationPlan)
	if err != nil {
		return nil, fmt.Errorf("parse contract verification_plan: %w", err)
	}
	acceptanceCriteria, err := jsonArrayToStringsSafe(contract.AcceptanceCriteria)
	if err != nil {
		return nil, fmt.Errorf("parse contract acceptance_criteria: %w", err)
	}

	reviewPolicy, err := loadContractReviewPolicy(contract.Metadata)
	if err != nil {
		return nil, fmt.Errorf("parse contract review_policy: %w", err)
	}
	reviewPolicyReason, reviewPolicySource, err := loadContractReviewPolicyOverrideContext(contract.Metadata)
	if err != nil {
		return nil, fmt.Errorf("parse contract review policy override context: %w", err)
	}
	if reviewPolicy == nil {
		defaultPolicy, resolveErr := reviewpolicy.ResolvePolicy(inferTaskCategory(task), nil)
		if resolveErr != nil {
			return nil, fmt.Errorf("resolve fallback review policy: %w", resolveErr)
		}
		reviewPolicy = defaultPolicy
	}

	contractCtx := &runtime.ContractCtx{
		ID:                 contract.ID,
		Scope:              contract.Scope,
		DoneDefinition:     doneDefinition,
		VerificationPlan:   verificationPlan,
		AcceptanceCriteria: acceptanceCriteria,
	}
	if reviewPolicy != nil {
		contractCtx.ReviewPolicy = resolvedPolicyToRuntime(reviewPolicy)
	}
	contractCtx.ReviewPolicyReason = reviewPolicyReason
	contractCtx.ReviewPolicySource = reviewPolicySource
	return contractCtx, nil
}

func buildReviewMetadata(artifactCtxs []runtime.ArtifactCtx, contractCtx *runtime.ContractCtx, action runtime.ReviewAction) (map[string]any, *string) {
	artifacts := make([]map[string]any, 0, len(artifactCtxs))
	var artifactVersionID *string
	for _, artifact := range artifactCtxs {
		if artifact.VersionID != "" && artifactVersionID == nil {
			versionIDCopy := artifact.VersionID
			artifactVersionID = &versionIDCopy
		}
		entry := map[string]any{
			"artifact_id":   artifact.ID,
			"version_id":    artifact.VersionID,
			"version":       artifact.Version,
			"name":          artifact.Name,
			"artifact_type": artifact.ArtifactType,
			"version_uri":   artifact.VersionURI,
			"content_type":  artifact.ContentType,
		}
		if artifact.Content != "" {
			entry["content_preview"] = artifact.Content
		}
		artifacts = append(artifacts, entry)
	}
	metadata := map[string]any{
		"artifact_count": len(artifacts),
		"artifacts":      artifacts,
	}
	if contractCtx != nil && contractCtx.ReviewPolicy != nil {
		metadata["task_category"] = contractCtx.ReviewPolicy.TaskCategory
		if action.TemplateKey == "" {
			metadata["template_key"] = contractCtx.ReviewPolicy.TemplateKey
		}
		if action.PassThreshold == 0 {
			metadata["pass_threshold"] = contractCtx.ReviewPolicy.PassThreshold
		}
	}
	if contractCtx != nil {
		if contractCtx.ReviewPolicyReason != "" {
			metadata["review_policy_reason"] = contractCtx.ReviewPolicyReason
		}
		if len(contractCtx.ReviewPolicySource) > 0 {
			metadata["review_policy_source"] = contractCtx.ReviewPolicySource
		}
	}
	if action.TemplateKey != "" {
		metadata["template_key"] = action.TemplateKey
	}
	if action.PassThreshold > 0 {
		metadata["pass_threshold"] = action.PassThreshold
	}
	metadata["total_score"] = action.TotalScore
	if len(action.HardGateResults) > 0 {
		metadata["hard_gate_results"] = action.HardGateResults
	}
	if len(action.ScoreItems) > 0 {
		metadata["score_items"] = action.ScoreItems
	}
	if len(action.MustFixItems) > 0 {
		metadata["must_fix_items"] = action.MustFixItems
	}
	if len(action.Suggestions) > 0 {
		metadata["suggestions"] = action.Suggestions
	}
	return metadata, artifactVersionID
}

func buildContractMetadata(task *model.Task, action runtime.ContractAction) (map[string]any, error) {
	taskCategory := action.TaskCategory
	if taskCategory == "" {
		taskCategory = inferTaskCategory(task)
	}
	override, err := normalizePolicyOverride(action.ReviewPolicy)
	if err != nil {
		return nil, err
	}
	resolved, err := reviewpolicy.ResolvePolicy(taskCategory, override)
	if err != nil {
		return nil, err
	}
	if action.ReviewTemplateKey != "" && action.ReviewTemplateKey != resolved.TemplateKey {
		if strings.TrimSpace(action.ReviewTemplateKey) == resolved.TaskCategory {
			action.ReviewTemplateKey = resolved.TemplateKey
		} else {
			return nil, fmt.Errorf("review_template_key %q does not match resolved template %q", action.ReviewTemplateKey, resolved.TemplateKey)
		}
	}
	reviewPolicyReason := strings.TrimSpace(action.ReviewPolicyReason)
	reviewPolicySource := normalizeReviewPolicySources(action.ReviewPolicySource)
	if len(override) == 0 {
		if reviewPolicyReason != "" || len(reviewPolicySource) > 0 {
			return nil, fmt.Errorf("review_policy_reason/source requires review_policy override")
		}
	} else {
		if reviewPolicyReason == "" {
			return nil, fmt.Errorf("review_policy_reason is required when review_policy override is present")
		}
		if len(reviewPolicySource) == 0 {
			return nil, fmt.Errorf("review_policy_source is required when review_policy override is present")
		}
	}
	metadata := map[string]any{
		"task_category":       resolved.TaskCategory,
		"review_template_key": resolved.TemplateKey,
		"review_policy":       resolved,
	}
	if len(override) > 0 {
		metadata["review_policy_override"] = override
		metadata["review_policy_override_reason"] = reviewPolicyReason
		metadata["review_policy_override_source"] = reviewPolicySource
	}
	return metadata, nil
}

func normalizePolicyOverride(raw any) (map[string]any, error) {
	if raw == nil {
		return nil, nil
	}
	if override, ok := raw.(map[string]any); ok {
		return override, nil
	}
	bytes, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal review policy override: %w", err)
	}
	if string(bytes) == "null" {
		return nil, nil
	}
	override := map[string]any{}
	if err := json.Unmarshal(bytes, &override); err != nil {
		return nil, fmt.Errorf("unmarshal review policy override: %w", err)
	}
	if len(override) == 0 {
		return nil, nil
	}
	return override, nil
}

func loadContractReviewPolicy(raw model.JSON) (*reviewpolicy.ResolvedPolicy, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var metadata struct {
		ReviewPolicy *reviewpolicy.ResolvedPolicy `json:"review_policy"`
	}
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return nil, err
	}
	return metadata.ReviewPolicy, nil
}

func loadContractReviewPolicyOverrideContext(raw model.JSON) (string, []string, error) {
	if len(raw) == 0 {
		return "", nil, nil
	}
	var metadata struct {
		ReviewPolicyOverrideReason string   `json:"review_policy_override_reason"`
		ReviewPolicyOverrideSource []string `json:"review_policy_override_source"`
	}
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return "", nil, err
	}
	return strings.TrimSpace(metadata.ReviewPolicyOverrideReason), normalizeReviewPolicySources(metadata.ReviewPolicyOverrideSource), nil
}

func resolvedPolicyToRuntime(policy *reviewpolicy.ResolvedPolicy) *runtime.ReviewPolicyCtx {
	if policy == nil {
		return nil
	}
	hardGates := make([]runtime.HardGateCtx, 0, len(policy.HardGates))
	for _, gate := range policy.HardGates {
		hardGates = append(hardGates, runtime.HardGateCtx{
			Key:         gate.Key,
			Name:        gate.Name,
			Description: gate.Description,
		})
	}
	scoreItems := make([]runtime.ScoreItemCtx, 0, len(policy.ScoreItems))
	for _, item := range policy.ScoreItems {
		scoreItems = append(scoreItems, runtime.ScoreItemCtx{
			Key:         item.Key,
			Name:        item.Name,
			Weight:      item.Weight,
			Description: item.Description,
		})
	}
	return &runtime.ReviewPolicyCtx{
		TemplateKey:   policy.TemplateKey,
		TaskCategory:  policy.TaskCategory,
		PassThreshold: policy.PassThreshold,
		HardGates:     hardGates,
		ScoreItems:    scoreItems,
	}
}

func normalizeReviewPolicySources(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	sources := make([]string, 0, len(raw))
	for _, source := range raw {
		source = strings.TrimSpace(source)
		if source == "" {
			continue
		}
		sources = append(sources, source)
	}
	if len(sources) == 0 {
		return nil
	}
	return sources
}

func jsonArrayToStringsSafe(raw model.JSON) ([]string, error) {
	if len(raw) == 0 {
		return []string{}, nil
	}
	var items []string
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func inferTaskCategory(task *model.Task) string {
	if task == nil {
		return "architecture"
	}
	if category := inferTaskCategoryFromKeywords(task); category != "" {
		return category
	}
	return "architecture"
}

func extractInlineContent(raw model.JSON) string {
	if len(raw) == 0 {
		return ""
	}
	meta := map[string]any{}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return ""
	}
	if content, ok := meta["inline_content"].(string); ok {
		return content
	}
	return ""
}

func extractCurrentReworkBrief(raw model.JSON) map[string]any {
	metadata := map[string]any{}
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return map[string]any{}
	}
	brief, ok := metadata["current_rework_brief"].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return brief
}

func (e *Engine) reloadTask(ctx context.Context, task *model.Task, stage string) error {
	fresh, err := e.services.Task.GetByID(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("reload task after %s: %w", stage, err)
	}
	if fresh == nil {
		return fmt.Errorf("task %s disappeared after %s", task.ID, stage)
	}
	*task = *fresh
	return nil
}

func (e *Engine) processTransitionActions(ctx context.Context, task *model.Task, actions []runtime.TransitionAction) error {
	for _, a := range actions {
		newStatus := model.TaskStatus(a.NewStatus)
		if err := e.services.Task.TransitionStatus(ctx, task.ID, newStatus); err != nil {
			return fmt.Errorf("transition task %s to %s: %w", task.ID, a.NewStatus, err)
		}
	}
	return nil
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
		return e.findAgent(ctx, model.AgentRoleSupervisor)
	}
	agent, err := e.services.Agent.FindByRoleCode(ctx, roleCode)
	if err != nil {
		return nil, fmt.Errorf("find supervisor by domain (%s): %w", domain, err)
	}
	if agent != nil {
		return agent, nil
	}
	return nil, fmt.Errorf("no active supervisor with role_code %q found", roleCode)
}

// inferExecutionDomain determines the task's execution domain from its content.
// If the task already has a non-empty domain set, it is preserved.
func inferExecutionDomain(task *model.Task) model.ExecutionDomain {
	if task.ExecutionDomain != "" && task.ExecutionDomain != model.ExecDomainGeneral {
		return task.ExecutionDomain
	}
	switch inferTaskCategory(task) {
	case "qa":
		return model.ExecDomainQA
	case "architecture", "backend", "frontend", "release":
		return model.ExecDomainDevelopment
	}

	switch inferSpecialization(task) {
	case model.AgentSpecBackend, model.AgentSpecFrontend, model.AgentSpecDevOps:
		return model.ExecDomainDevelopment
	case model.AgentSpecQA:
		return model.ExecDomainQA
	default:
		return model.ExecDomainGeneral
	}
}

// inferSpecialization determines the best worker specialization based on task content.
func inferSpecialization(task *model.Task) model.AgentSpecialization {
	if task == nil {
		return model.AgentSpecGeneral
	}
	if spec := specializationForTaskCategory(inferTaskCategoryFromKeywords(task)); spec != model.AgentSpecGeneral {
		return spec
	}

	combined := task.Title + " " + task.Description
	for _, rule := range []struct {
		spec     model.AgentSpecialization
		keywords []string
	}{
		{spec: model.AgentSpecDevOps, keywords: []string{"CI", "CD", "Docker", "Kubernetes", "基础设施", "监控"}},
		{spec: model.AgentSpecDesign, keywords: []string{"视觉设计", "交互设计", "原型", "Figma"}},
	} {
		if containsAnyCI(combined, rule.keywords) {
			return rule.spec
		}
	}
	return model.AgentSpecGeneral
}

func inferTaskCategoryFromKeywords(task *model.Task) string {
	if task == nil {
		return ""
	}
	title := strings.TrimSpace(task.Title)
	if category := matchTaskCategoryText(title); category != "" {
		return category
	}
	combined := task.Title + " " + task.Description
	return matchTaskCategoryText(combined)
}

func matchTaskCategoryText(text string) string {
	for _, rule := range []struct {
		category string
		keywords []string
	}{
		{category: "release", keywords: []string{"版本发布", "发布计划", "生产环境", "部署", "上线", "release", "deploy", "回滚", "发布说明", "发布单"}},
		{category: "qa", keywords: []string{"测试", "验收", "QA", "quality", "验证", "回归", "评审"}},
		{category: "frontend", keywords: []string{"前端", "页面", "组件", "UI", "交互", "React", "Next.js", "样式", "布局", "响应式"}},
		{category: "prd", keywords: []string{"PRD", "需求", "规格", "文档", "梳理"}},
		{category: "architecture", keywords: []string{"架构", "方案", "可行性", "评估"}},
		{category: "backend", keywords: []string{"API", "接口", "后端", "数据库", "migration", "服务端", "handler", "repository", "service", "实现", "开发"}},
	} {
		if containsAnyCI(text, rule.keywords) {
			return rule.category
		}
	}
	return ""
}

func resolveWorkerSpecialization(task *model.Task, contractCtx *runtime.ContractCtx) model.AgentSpecialization {
	if contractCtx != nil && contractCtx.ReviewPolicy != nil {
		if spec := specializationForTaskCategory(contractCtx.ReviewPolicy.TaskCategory); spec != model.AgentSpecGeneral {
			return spec
		}
	}
	return inferSpecialization(task)
}

func specializationForTaskCategory(taskCategory string) model.AgentSpecialization {
	switch strings.TrimSpace(taskCategory) {
	case "backend":
		return model.AgentSpecBackend
	case "frontend":
		return model.AgentSpecFrontend
	case "qa":
		return model.AgentSpecQA
	case "release":
		return model.AgentSpecRelease
	default:
		return model.AgentSpecGeneral
	}
}

func containsAnyCI(s string, keywords []string) bool {
	for _, keyword := range keywords {
		if containsCI(s, keyword) {
			return true
		}
	}
	return false
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

func (e *Engine) handleRunInterruption(ctx context.Context, rc *runCtx, proj *model.Project, err error) bool {
	if errors.Is(err, errPhaseWaitingApproval) {
		return true
	}
	var manualErr *manualInterventionError
	if !errors.As(err, &manualErr) {
		return false
	}
	if waitErr := e.waitForManualIntervention(ctx, rc, proj, manualErr); waitErr != nil {
		e.logger.Error("wait for manual intervention persistence error", "error", waitErr)
		e.failRun(ctx, rc, waitErr)
	}
	return true
}

func (e *Engine) waitForManualIntervention(ctx context.Context, rc *runCtx, proj *model.Project, manualErr *manualInterventionError) error {
	intervention := manualInterventionStateFromError(manualErr)
	summary := fmt.Sprintf("项目「%s」等待人工介入", proj.Name)
	switch intervention.ReasonCode {
	case model.ManualInterventionReasonReworkLimitReached:
		if intervention.TaskTitle != "" {
			summary = fmt.Sprintf("项目「%s」任务「%s」返工次数已达上限，等待人工处理", proj.Name, intervention.TaskTitle)
		} else {
			summary = fmt.Sprintf("项目「%s」返工次数已达上限，等待人工处理", proj.Name)
		}
	default:
		if intervention.TaskTitle != "" {
			summary = fmt.Sprintf("项目「%s」任务「%s」因 %s 执行失败等待人工介入", proj.Name, intervention.TaskTitle, intervention.AgentRole)
		} else if intervention.AgentRole != "" {
			summary = fmt.Sprintf("项目「%s」在 %s 步骤因 LLM 执行失败等待人工介入", proj.Name, intervention.AgentRole)
		} else {
			summary = fmt.Sprintf("项目「%s」因 LLM 执行失败等待人工介入", proj.Name)
		}
	}
	if err := e.workflow.WaitForManualIntervention(ctx, rc.run, summary, manualErr.reason, intervention); err != nil {
		return fmt.Errorf("mark run waiting manual intervention: %w", err)
	}
	after := map[string]string{
		"run_id":      rc.run.ID,
		"agent_role":  manualErr.agentRole,
		"error":       manualErr.reason,
		"reason_code": string(intervention.ReasonCode),
	}
	if manualErr.phaseID != "" {
		after["phase_id"] = manualErr.phaseID
	}
	if manualErr.taskID != "" {
		after["task_id"] = manualErr.taskID
	}
	e.services.Audit.LogEvent(ctx, "system", "", "workflow.waiting_manual_intervention",
		summary, "project", proj.ID, nil, after)
	return nil
}

func manualInterventionStateFromError(manualErr *manualInterventionError) *model.WorkflowManualIntervention {
	if manualErr == nil {
		return nil
	}
	actions := []model.ManualInterventionAction{
		model.ManualInterventionActionResume,
		model.ManualInterventionActionCancelRun,
	}
	if manualErr.reasonCode == model.ManualInterventionReasonReworkLimitReached {
		actions = append([]model.ManualInterventionAction{model.ManualInterventionActionEscalateToApproval}, actions...)
	}
	return &model.WorkflowManualIntervention{
		ReasonCode:       manualErr.reasonCode,
		Reason:           manualErr.reason,
		AgentRole:        manualErr.agentRole,
		PhaseID:          manualErr.phaseID,
		PhaseName:        manualErr.phaseName,
		TaskID:           manualErr.taskID,
		TaskTitle:        manualErr.taskTitle,
		AvailableActions: actions,
	}
}

func decodeWorkflowManualIntervention(raw model.JSON) (*model.WorkflowManualIntervention, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var metadata struct {
		ManualIntervention *model.WorkflowManualIntervention `json:"manual_intervention"`
	}
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return nil, err
	}
	return metadata.ManualIntervention, nil
}

func (e *Engine) escalateTaskToApproval(ctx context.Context, run *model.WorkflowRun, intervention *model.WorkflowManualIntervention, note string) error {
	task, err := e.services.Task.GetByID(ctx, intervention.TaskID)
	if err != nil {
		return fmt.Errorf("load task for manual intervention: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task %s not found", intervention.TaskID)
	}
	if err := e.services.Task.EscalateToPendingApproval(ctx, task.ID, note); err != nil {
		return fmt.Errorf("escalate task to pending approval: %w", err)
	}

	metadata, description, err := e.buildManualInterventionApprovalPayload(ctx, run, task, intervention, note)
	if err != nil {
		return err
	}
	metaJSON := model.JSON("{}")
	if len(metadata) > 0 {
		encoded, marshalErr := json.Marshal(metadata)
		if marshalErr != nil {
			return fmt.Errorf("marshal manual intervention approval metadata: %w", marshalErr)
		}
		metaJSON = model.JSON(encoded)
	}
	approval := &model.ApprovalRequest{
		ProjectID:    task.ProjectID,
		TaskID:       &task.ID,
		RequestedBy:  "system",
		ApproverType: "user",
		Title:        fmt.Sprintf("任务「%s」已由人工介入放行，等待审批", task.Title),
		Description:  description,
		Metadata:     metaJSON,
	}
	if err := e.services.Approval.Create(ctx, approval); err != nil {
		return fmt.Errorf("create approval request for manual intervention: %w", err)
	}

	e.services.Audit.LogEvent(ctx, "user", "", "workflow.manual_intervention_resolved",
		fmt.Sprintf("任务「%s」已由人工介入放行至审批", task.Title),
		"task", task.ID, nil, map[string]string{
			"run_id":      run.ID,
			"action":      string(model.ManualInterventionActionEscalateToApproval),
			"reason_code": string(intervention.ReasonCode),
			"note":        note,
		})
	return nil
}

func (e *Engine) buildManualInterventionApprovalPayload(ctx context.Context, run *model.WorkflowRun, task *model.Task, intervention *model.WorkflowManualIntervention, note string) (map[string]any, string, error) {
	metadata := map[string]any{
		"source":                          "manual_intervention",
		"run_id":                          run.ID,
		"task_title":                      task.Title,
		"manual_intervention_action":      string(model.ManualInterventionActionEscalateToApproval),
		"manual_intervention_reason_code": string(intervention.ReasonCode),
	}
	if note != "" {
		metadata["manual_intervention_note"] = note
	}
	if task.PhaseID != nil {
		metadata["phase_id"] = *task.PhaseID
	}

	description := fmt.Sprintf("任务「%s」因人工介入放行进入审批。", task.Title)
	reviews, _, err := e.services.Review.List(ctx, repository.ReviewListParams{
		TaskID: task.ID,
		Limit:  1,
		Offset: 0,
	})
	if err != nil {
		return nil, "", fmt.Errorf("load latest review for manual intervention: %w", err)
	}
	if len(reviews) > 0 {
		review := reviews[0]
		metadata["review_id"] = review.ID
		metadata["review_verdict"] = review.Verdict
		metadata["review_summary"] = review.Summary
		findings, err := jsonArrayToStringsSafe(review.Findings)
		if err != nil {
			return nil, "", fmt.Errorf("parse review findings: %w", err)
		}
		recommendations, err := jsonArrayToStringsSafe(review.Recommendations)
		if err != nil {
			return nil, "", fmt.Errorf("parse review recommendations: %w", err)
		}
		metadata["findings"] = findings
		metadata["recommendations"] = recommendations
		description = fmt.Sprintf("%s 最近一次评审摘要：%s。", description, review.Summary)
	}
	if note != "" {
		description = fmt.Sprintf("%s 人工说明：%s。", description, note)
	}
	description += "等待审批确认。"
	return metadata, description, nil
}

func shouldResumeFromPM(run *model.WorkflowRun) bool {
	if run == nil || len(run.Steps) == 0 {
		return false
	}
	for idx := len(run.Steps) - 1; idx >= 0; idx-- {
		step := run.Steps[idx]
		if step == nil {
			continue
		}
		return step.AgentRole == "pm" && step.Status == model.WorkflowStepFailed
	}
	return false
}

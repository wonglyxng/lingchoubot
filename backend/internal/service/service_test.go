package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

type fakeAuditRepo struct {
	entries []*model.AuditLog
}

func (r *fakeAuditRepo) Create(_ context.Context, a *model.AuditLog) error {
	copyEntry := *a
	r.entries = append(r.entries, &copyEntry)
	return nil
}

func (r *fakeAuditRepo) List(_ context.Context, _ repository.AuditListParams) ([]*model.AuditLog, int, error) {
	return nil, 0, nil
}

func (r *fakeAuditRepo) ProjectTimeline(_ context.Context, _ string, _, _ int) ([]*model.AuditLog, int, error) {
	return nil, 0, nil
}

func (r *fakeAuditRepo) TaskTimeline(_ context.Context, _ string, _, _ int) ([]*model.AuditLog, int, error) {
	return nil, 0, nil
}

type fakeTaskRepo struct {
	tasks             map[string]*model.Task
	updateStatusCalls int
	lastUpdatedTaskID string
	lastUpdatedStatus model.TaskStatus
	deleteCalls       int
	updateCalls       int
	createCalls       int
}

func (r *fakeTaskRepo) Create(_ context.Context, t *model.Task) error {
	r.createCalls++
	copyTask := *t
	if copyTask.ID == "" {
		copyTask.ID = fmt.Sprintf("task-%d", len(r.tasks)+1)
	}
	if r.tasks == nil {
		r.tasks = map[string]*model.Task{}
	}
	r.tasks[copyTask.ID] = &copyTask
	t.ID = copyTask.ID
	return nil
}

func (r *fakeTaskRepo) GetByID(_ context.Context, id string) (*model.Task, error) {
	if r.tasks == nil {
		return nil, nil
	}
	task := r.tasks[id]
	if task == nil {
		return nil, nil
	}
	copyTask := *task
	return &copyTask, nil
}

func (r *fakeTaskRepo) List(_ context.Context, _ repository.TaskListParams) ([]*model.Task, int, error) {
	items := make([]*model.Task, 0, len(r.tasks))
	for _, task := range r.tasks {
		copyTask := *task
		items = append(items, &copyTask)
	}
	return items, len(items), nil
}

func (r *fakeTaskRepo) Update(_ context.Context, t *model.Task) error {
	r.updateCalls++
	if r.tasks == nil {
		r.tasks = map[string]*model.Task{}
	}
	copyTask := *t
	r.tasks[t.ID] = &copyTask
	return nil
}

func (r *fakeTaskRepo) UpdateStatus(_ context.Context, id string, status model.TaskStatus) error {
	r.updateStatusCalls++
	r.lastUpdatedTaskID = id
	r.lastUpdatedStatus = status
	if task := r.tasks[id]; task != nil {
		task.Status = status
	}
	return nil
}

func (r *fakeTaskRepo) Delete(_ context.Context, id string) error {
	r.deleteCalls++
	delete(r.tasks, id)
	return nil
}

type fakeApprovalRepo struct {
	approvals   map[string]*model.ApprovalRequest
	decideCalls int
	lastStatus  model.ApprovalStatus
	lastNote    string
	lastID      string
}

func (r *fakeApprovalRepo) Create(_ context.Context, a *model.ApprovalRequest) error {
	copyApproval := *a
	if copyApproval.ID == "" {
		copyApproval.ID = fmt.Sprintf("approval-%d", len(r.approvals)+1)
	}
	if r.approvals == nil {
		r.approvals = map[string]*model.ApprovalRequest{}
	}
	r.approvals[copyApproval.ID] = &copyApproval
	a.ID = copyApproval.ID
	return nil
}

func (r *fakeApprovalRepo) GetByID(_ context.Context, id string) (*model.ApprovalRequest, error) {
	if r.approvals == nil {
		return nil, nil
	}
	approval := r.approvals[id]
	if approval == nil {
		return nil, nil
	}
	copyApproval := *approval
	return &copyApproval, nil
}

func (r *fakeApprovalRepo) List(_ context.Context, _ repository.ApprovalListParams) ([]*model.ApprovalRequest, int, error) {
	items := make([]*model.ApprovalRequest, 0, len(r.approvals))
	for _, approval := range r.approvals {
		copyApproval := *approval
		items = append(items, &copyApproval)
	}
	return items, len(items), nil
}

func (r *fakeApprovalRepo) Decide(_ context.Context, id string, status model.ApprovalStatus, note string) error {
	r.decideCalls++
	r.lastID = id
	r.lastStatus = status
	r.lastNote = note
	if approval := r.approvals[id]; approval != nil {
		approval.Status = status
		approval.DecisionNote = note
		now := time.Now()
		approval.DecidedAt = &now
	}
	return nil
}

type fakeWorkflowRunRepo struct {
	runs              map[string]*model.WorkflowRun
	createCalls       int
	updateStatusCalls int
	lastUpdatedRun    *model.WorkflowRun
}

type fakeReviewRepo struct {
	reviews map[string]*model.ReviewReport
}

func (r *fakeReviewRepo) Create(_ context.Context, rr *model.ReviewReport) error {
	if rr.ID == "" {
		rr.ID = fmt.Sprintf("review-%d", len(r.reviews)+1)
	}
	if r.reviews == nil {
		r.reviews = map[string]*model.ReviewReport{}
	}
	copyRR := *rr
	r.reviews[copyRR.ID] = &copyRR
	return nil
}

func (r *fakeReviewRepo) GetByID(_ context.Context, id string) (*model.ReviewReport, error) {
	if r.reviews == nil {
		return nil, nil
	}
	rr := r.reviews[id]
	if rr == nil {
		return nil, nil
	}
	copyRR := *rr
	return &copyRR, nil
}

func (r *fakeReviewRepo) List(_ context.Context, _ repository.ReviewListParams) ([]*model.ReviewReport, int, error) {
	items := make([]*model.ReviewReport, 0, len(r.reviews))
	for _, rr := range r.reviews {
		copyRR := *rr
		items = append(items, &copyRR)
	}
	return items, len(items), nil
}

func (r *fakeWorkflowRunRepo) Create(_ context.Context, run *model.WorkflowRun) error {
	r.createCalls++
	copyRun := *run
	if copyRun.ID == "" {
		copyRun.ID = fmt.Sprintf("run-%d", len(r.runs)+1)
	}
	if r.runs == nil {
		r.runs = map[string]*model.WorkflowRun{}
	}
	r.runs[copyRun.ID] = &copyRun
	run.ID = copyRun.ID
	return nil
}

func (r *fakeWorkflowRunRepo) GetByID(_ context.Context, id string) (*model.WorkflowRun, error) {
	if r.runs == nil {
		return nil, nil
	}
	run := r.runs[id]
	if run == nil {
		return nil, nil
	}
	copyRun := *run
	return &copyRun, nil
}

func (r *fakeWorkflowRunRepo) UpdateStatus(_ context.Context, run *model.WorkflowRun) error {
	r.updateStatusCalls++
	copyRun := *run
	r.lastUpdatedRun = &copyRun
	if r.runs == nil {
		r.runs = map[string]*model.WorkflowRun{}
	}
	r.runs[run.ID] = &copyRun
	return nil
}

func (r *fakeWorkflowRunRepo) List(_ context.Context, _ repository.WorkflowRunListParams) ([]*model.WorkflowRun, int, error) {
	items := make([]*model.WorkflowRun, 0, len(r.runs))
	for _, run := range r.runs {
		copyRun := *run
		items = append(items, &copyRun)
	}
	return items, len(items), nil
}

type fakeWorkflowStepRepo struct {
	steps             map[string]*model.WorkflowStep
	createCalls       int
	updateStatusCalls int
	lastUpdatedStep   *model.WorkflowStep
}

func (r *fakeWorkflowStepRepo) Create(_ context.Context, step *model.WorkflowStep) error {
	r.createCalls++
	copyStep := *step
	if copyStep.ID == "" {
		copyStep.ID = fmt.Sprintf("step-%d", len(r.steps)+1)
	}
	if r.steps == nil {
		r.steps = map[string]*model.WorkflowStep{}
	}
	r.steps[copyStep.ID] = &copyStep
	step.ID = copyStep.ID
	return nil
}

func (r *fakeWorkflowStepRepo) UpdateStatus(_ context.Context, step *model.WorkflowStep) error {
	r.updateStatusCalls++
	copyStep := *step
	r.lastUpdatedStep = &copyStep
	if r.steps == nil {
		r.steps = map[string]*model.WorkflowStep{}
	}
	r.steps[step.ID] = &copyStep
	return nil
}

func (r *fakeWorkflowStepRepo) ListByRunID(_ context.Context, runID string) ([]*model.WorkflowStep, error) {
	items := make([]*model.WorkflowStep, 0)
	for _, step := range r.steps {
		if step.RunID != runID {
			continue
		}
		copyStep := *step
		items = append(items, &copyStep)
	}
	return items, nil
}

func newTestAuditService() (*AuditService, *fakeAuditRepo) {
	repo := &fakeAuditRepo{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return &AuditService{repo: repo, logger: logger}, repo
}

func TestTaskServiceTransitionStatus(t *testing.T) {
	ctx := context.Background()
	auditSvc, auditRepo := newTestAuditService()
	taskRepo := &fakeTaskRepo{
		tasks: map[string]*model.Task{
			"task-1": {
				ID:        "task-1",
				Title:     "API 设计",
				Status:    model.TaskStatusPending,
				ProjectID: "proj-1",
			},
		},
	}
	svc := &TaskService{repo: taskRepo, audit: auditSvc}

	if err := svc.TransitionStatus(ctx, "task-1", model.TaskStatusAssigned); err != nil {
		t.Fatalf("TransitionStatus returned error: %v", err)
	}

	if taskRepo.updateStatusCalls != 1 {
		t.Fatalf("expected 1 UpdateStatus call, got %d", taskRepo.updateStatusCalls)
	}
	if taskRepo.lastUpdatedStatus != model.TaskStatusAssigned {
		t.Fatalf("expected status %q, got %q", model.TaskStatusAssigned, taskRepo.lastUpdatedStatus)
	}
	if got := taskRepo.tasks["task-1"].Status; got != model.TaskStatusAssigned {
		t.Fatalf("expected stored task status %q, got %q", model.TaskStatusAssigned, got)
	}
	if len(auditRepo.entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(auditRepo.entries))
	}
	if auditRepo.entries[0].EventType != "task.status_changed" {
		t.Fatalf("expected audit event task.status_changed, got %q", auditRepo.entries[0].EventType)
	}
}

func TestTaskServiceTransitionStatusRejectsInvalidTransition(t *testing.T) {
	ctx := context.Background()
	auditSvc, auditRepo := newTestAuditService()
	taskRepo := &fakeTaskRepo{
		tasks: map[string]*model.Task{
			"task-1": {
				ID:        "task-1",
				Title:     "API 设计",
				Status:    model.TaskStatusPending,
				ProjectID: "proj-1",
			},
		},
	}
	svc := &TaskService{repo: taskRepo, audit: auditSvc}

	err := svc.TransitionStatus(ctx, "task-1", model.TaskStatusCompleted)
	if err == nil {
		t.Fatal("expected invalid transition error, got nil")
	}
	if taskRepo.updateStatusCalls != 0 {
		t.Fatalf("expected 0 UpdateStatus calls, got %d", taskRepo.updateStatusCalls)
	}
	if len(auditRepo.entries) != 0 {
		t.Fatalf("expected 0 audit entries, got %d", len(auditRepo.entries))
	}
}

func TestApprovalRequestServiceDecideTransitionsTask(t *testing.T) {
	tests := []struct {
		name       string
		decision   model.ApprovalStatus
		wantStatus model.TaskStatus
	}{
		{name: "approved completes task", decision: model.ApprovalStatusApproved, wantStatus: model.TaskStatusCompleted},
		{name: "rejected sends task to revision", decision: model.ApprovalStatusRejected, wantStatus: model.TaskStatusRevisionRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			auditSvc, auditRepo := newTestAuditService()
			taskID := "task-1"
			taskRepo := &fakeTaskRepo{
				tasks: map[string]*model.Task{
					taskID: {
						ID:        taskID,
						Title:     "交付评审",
						Status:    model.TaskStatusPendingApproval,
						ProjectID: "proj-1",
					},
				},
			}
			approvalRepo := &fakeApprovalRepo{
				approvals: map[string]*model.ApprovalRequest{
					"approval-1": {
						ID:          "approval-1",
						ProjectID:   "proj-1",
						TaskID:      &taskID,
						RequestedBy: "reviewer-1",
						Title:       "审批交付物",
						Status:      model.ApprovalStatusPending,
						Metadata:    model.JSON("{}"),
					},
				},
			}
			taskSvc := &TaskService{repo: taskRepo, audit: auditSvc}
			svc := &ApprovalRequestService{repo: approvalRepo, taskSvc: taskSvc, audit: auditSvc}

			if err := svc.Decide(ctx, "approval-1", tt.decision, "ok"); err != nil {
				t.Fatalf("Decide returned error: %v", err)
			}

			if approvalRepo.decideCalls != 1 {
				t.Fatalf("expected 1 approval Decide call, got %d", approvalRepo.decideCalls)
			}
			if taskRepo.lastUpdatedStatus != tt.wantStatus {
				t.Fatalf("expected task status %q, got %q", tt.wantStatus, taskRepo.lastUpdatedStatus)
			}
			if got := taskRepo.tasks[taskID].Status; got != tt.wantStatus {
				t.Fatalf("expected stored task status %q, got %q", tt.wantStatus, got)
			}
			if len(auditRepo.entries) != 2 {
				t.Fatalf("expected 2 audit entries, got %d", len(auditRepo.entries))
			}
			if auditRepo.entries[0].EventType != "approval_request.decided" {
				t.Fatalf("expected first audit event approval_request.decided, got %q", auditRepo.entries[0].EventType)
			}
			if auditRepo.entries[1].EventType != "task.status_changed" {
				t.Fatalf("expected second audit event task.status_changed, got %q", auditRepo.entries[1].EventType)
			}
		})
	}
}

func TestReviewReportApprovedCreatesApprovalAndTransitionsTask(t *testing.T) {
	ctx := context.Background()
	auditSvc, _ := newTestAuditService()
	taskRepo := &fakeTaskRepo{
		tasks: map[string]*model.Task{
			"task-1": {
				ID:        "task-1",
				Title:     "API 设计",
				Status:    model.TaskStatusInReview,
				ProjectID: "proj-1",
			},
		},
	}
	approvalRepo := &fakeApprovalRepo{}
	reviewRepo := &fakeReviewRepo{}
	taskSvc := &TaskService{repo: taskRepo, audit: auditSvc}
	approvalSvc := &ApprovalRequestService{repo: approvalRepo, taskSvc: taskSvc, audit: auditSvc}
	reviewSvc := &ReviewReportService{repo: reviewRepo, taskSvc: taskSvc, approvalSvc: approvalSvc, audit: auditSvc}

	rr := &model.ReviewReport{
		TaskID:     "task-1",
		ReviewerID: "reviewer-1",
		Verdict:    model.ReviewVerdictApproved,
		Summary:    "looks good",
	}
	if err := reviewSvc.Create(ctx, rr); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	// Task should be in pending_approval
	if got := taskRepo.tasks["task-1"].Status; got != model.TaskStatusPendingApproval {
		t.Fatalf("expected task status %q, got %q", model.TaskStatusPendingApproval, got)
	}
	// An approval request should have been created
	if len(approvalRepo.approvals) != 1 {
		t.Fatalf("expected 1 approval request, got %d", len(approvalRepo.approvals))
	}
	for _, ar := range approvalRepo.approvals {
		if ar.TaskID == nil || *ar.TaskID != "task-1" {
			t.Fatalf("approval request task_id mismatch")
		}
		if ar.ProjectID != "proj-1" {
			t.Fatalf("approval request project_id = %q, want %q", ar.ProjectID, "proj-1")
		}
	}
}

func TestReviewReportRejectedDoesNotCreateApproval(t *testing.T) {
	ctx := context.Background()
	auditSvc, _ := newTestAuditService()
	taskRepo := &fakeTaskRepo{
		tasks: map[string]*model.Task{
			"task-1": {
				ID:        "task-1",
				Title:     "API 设计",
				Status:    model.TaskStatusInReview,
				ProjectID: "proj-1",
			},
		},
	}
	approvalRepo := &fakeApprovalRepo{}
	reviewRepo := &fakeReviewRepo{}
	taskSvc := &TaskService{repo: taskRepo, audit: auditSvc}
	approvalSvc := &ApprovalRequestService{repo: approvalRepo, taskSvc: taskSvc, audit: auditSvc}
	reviewSvc := &ReviewReportService{repo: reviewRepo, taskSvc: taskSvc, approvalSvc: approvalSvc, audit: auditSvc}

	rr := &model.ReviewReport{
		TaskID:     "task-1",
		ReviewerID: "reviewer-1",
		Verdict:    model.ReviewVerdictRejected,
		Summary:    "needs work",
	}
	if err := reviewSvc.Create(ctx, rr); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	// Task should be in revision_required
	if got := taskRepo.tasks["task-1"].Status; got != model.TaskStatusRevisionRequired {
		t.Fatalf("expected task status %q, got %q", model.TaskStatusRevisionRequired, got)
	}
	// No approval request should have been created
	if len(approvalRepo.approvals) != 0 {
		t.Fatalf("expected 0 approval requests, got %d", len(approvalRepo.approvals))
	}
}

func TestWorkflowServiceRunLifecycle(t *testing.T) {
	ctx := context.Background()
	runRepo := &fakeWorkflowRunRepo{}
	stepRepo := &fakeWorkflowStepRepo{}
	svc := &WorkflowService{runRepo: runRepo, stepRepo: stepRepo}

	run, err := svc.CreateRun(ctx, "proj-1")
	if err != nil {
		t.Fatalf("CreateRun returned error: %v", err)
	}
	if run.Status != model.WorkflowRunRunning {
		t.Fatalf("expected new run status %q, got %q", model.WorkflowRunRunning, run.Status)
	}
	if run.ID == "" {
		t.Fatal("expected run ID to be assigned")
	}

	if err := svc.CompleteRun(ctx, run, "done"); err != nil {
		t.Fatalf("CompleteRun returned error: %v", err)
	}
	if runRepo.lastUpdatedRun == nil || runRepo.lastUpdatedRun.Status != model.WorkflowRunCompleted {
		t.Fatalf("expected updated run status %q, got %#v", model.WorkflowRunCompleted, runRepo.lastUpdatedRun)
	}
	if runRepo.lastUpdatedRun.CompletedAt == nil {
		t.Fatal("expected completed_at to be set on completion")
	}

	run2, err := svc.CreateRun(ctx, "proj-2")
	if err != nil {
		t.Fatalf("CreateRun second run returned error: %v", err)
	}
	if err := svc.FailRun(ctx, run2, "boom"); err != nil {
		t.Fatalf("FailRun returned error: %v", err)
	}
	if runRepo.lastUpdatedRun == nil || runRepo.lastUpdatedRun.Status != model.WorkflowRunFailed {
		t.Fatalf("expected updated run status %q, got %#v", model.WorkflowRunFailed, runRepo.lastUpdatedRun)
	}
	if runRepo.lastUpdatedRun.Error != "boom" {
		t.Fatalf("expected run error %q, got %q", "boom", runRepo.lastUpdatedRun.Error)
	}
}

func TestWorkflowServiceStepLifecycle(t *testing.T) {
	ctx := context.Background()
	runRepo := &fakeWorkflowRunRepo{}
	stepRepo := &fakeWorkflowStepRepo{}
	svc := &WorkflowService{runRepo: runRepo, stepRepo: stepRepo}

	step, err := svc.AddStep(ctx, "run-1", "PM 分解", "pm", 1)
	if err != nil {
		t.Fatalf("AddStep returned error: %v", err)
	}
	if step.Status != model.WorkflowStepPending {
		t.Fatalf("expected new step status %q, got %q", model.WorkflowStepPending, step.Status)
	}

	if err := svc.StartStep(ctx, step); err != nil {
		t.Fatalf("StartStep returned error: %v", err)
	}
	if stepRepo.lastUpdatedStep == nil || stepRepo.lastUpdatedStep.Status != model.WorkflowStepRunning {
		t.Fatalf("expected updated step status %q, got %#v", model.WorkflowStepRunning, stepRepo.lastUpdatedStep)
	}
	if stepRepo.lastUpdatedStep.StartedAt == nil {
		t.Fatal("expected started_at to be set on start")
	}

	if err := svc.CompleteStep(ctx, step, "ok"); err != nil {
		t.Fatalf("CompleteStep returned error: %v", err)
	}
	if stepRepo.lastUpdatedStep == nil || stepRepo.lastUpdatedStep.Status != model.WorkflowStepCompleted {
		t.Fatalf("expected updated step status %q, got %#v", model.WorkflowStepCompleted, stepRepo.lastUpdatedStep)
	}
	if stepRepo.lastUpdatedStep.CompletedAt == nil {
		t.Fatal("expected completed_at to be set on completion")
	}
	if stepRepo.lastUpdatedStep.Summary != "ok" {
		t.Fatalf("expected step summary %q, got %q", "ok", stepRepo.lastUpdatedStep.Summary)
	}

	step2, err := svc.AddStep(ctx, "run-1", "Reviewer 评审", "reviewer", 2)
	if err != nil {
		t.Fatalf("AddStep second step returned error: %v", err)
	}
	if err := svc.FailStep(ctx, step2, "fail"); err != nil {
		t.Fatalf("FailStep returned error: %v", err)
	}
	if stepRepo.lastUpdatedStep == nil || stepRepo.lastUpdatedStep.Status != model.WorkflowStepFailed {
		t.Fatalf("expected updated step status %q, got %#v", model.WorkflowStepFailed, stepRepo.lastUpdatedStep)
	}
	if stepRepo.lastUpdatedStep.Error != "fail" {
		t.Fatalf("expected step error %q, got %q", "fail", stepRepo.lastUpdatedStep.Error)
	}
}

func TestWorkflowServiceGetRunLoadsSteps(t *testing.T) {
	ctx := context.Background()
	runRepo := &fakeWorkflowRunRepo{
		runs: map[string]*model.WorkflowRun{
			"run-1": {
				ID:        "run-1",
				ProjectID: "proj-1",
				Status:    model.WorkflowRunCompleted,
			},
		},
	}
	stepRepo := &fakeWorkflowStepRepo{
		steps: map[string]*model.WorkflowStep{
			"step-1": {ID: "step-1", RunID: "run-1", Name: "PM 分解", AgentRole: "pm", Status: model.WorkflowStepCompleted, SortOrder: 1},
			"step-2": {ID: "step-2", RunID: "run-1", Name: "Reviewer 评审", AgentRole: "reviewer", Status: model.WorkflowStepCompleted, SortOrder: 2},
		},
	}
	svc := &WorkflowService{runRepo: runRepo, stepRepo: stepRepo}

	run, err := svc.GetRun(ctx, "run-1")
	if err != nil {
		t.Fatalf("GetRun returned error: %v", err)
	}
	if run == nil {
		t.Fatal("expected run, got nil")
	}
	if len(run.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(run.Steps))
	}
	if run.Steps[0].RunID != "run-1" || run.Steps[1].RunID != "run-1" {
		t.Fatal("expected loaded steps to belong to run-1")
	}
}

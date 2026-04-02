package orchestrator_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/orchestrator"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
	"github.com/lingchou/lingchoubot/backend/internal/runtime"
	"github.com/lingchou/lingchoubot/backend/internal/testutil"
)

// ---------- Happy Path ----------

func TestIntegration_HappyPath(t *testing.T) {
	ctx := context.Background()
	f := testutil.NewFixture()

	// Seed standard agents (PM, DevSupervisor, QASupervisor, BackendWorker, FrontendWorker, QAWorker, Reviewer)
	if err := f.SeedStandardAgents(ctx); err != nil {
		t.Fatalf("seed agents: %v", err)
	}

	// Create a project
	proj := &model.Project{Name: "集成测试项目", Description: "端到端 Happy Path 测试"}
	if err := f.ProjectSvc.Create(ctx, proj); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// Run the engine
	run, err := f.Engine.Run(ctx, proj.ID)
	if err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}

	// ---- Verify WorkflowRun ----
	if run.Status != model.WorkflowRunWaitingApproval {
		t.Errorf("run status = %q, want %q", run.Status, model.WorkflowRunWaitingApproval)
	}

	// ---- Verify Phases ----
	// MockPM creates 4 phases: 需求分析, 方案设计, 开发实现, 测试验收
	phases, _ := f.PhaseSvc.ListByProject(ctx, proj.ID)
	if len(phases) != 4 {
		t.Fatalf("phases = %d, want 4", len(phases))
	}

	// ---- Verify Tasks ----
	// MockPM creates 9 tasks
	allTasks, _, _ := f.TaskSvc.List(ctx, repository.TaskListParams{ProjectID: proj.ID, Limit: 100})
	if len(allTasks) != 9 {
		t.Fatalf("tasks = %d, want 9", len(allTasks))
	}

	phaseByName := make(map[string]*model.Phase, len(phases))
	for _, phase := range phases {
		phaseByName[phase.Name] = phase
	}
	analysisPhase := phaseByName["需求分析"]
	if analysisPhase == nil {
		t.Fatal("需求分析阶段不存在")
	}
	designPhase := phaseByName["方案设计"]
	if designPhase == nil {
		t.Fatal("方案设计阶段不存在")
	}
	if analysisPhase.Status != model.PhaseStatusActive {
		t.Fatalf("analysis phase status = %q, want %q", analysisPhase.Status, model.PhaseStatusActive)
	}
	if designPhase.Status != model.PhaseStatusPending {
		t.Fatalf("design phase status = %q, want %q", designPhase.Status, model.PhaseStatusPending)
	}

	analysisTasks, _, _ := f.TaskSvc.List(ctx, repository.TaskListParams{PhaseID: analysisPhase.ID, Limit: 100})
	if len(analysisTasks) != 2 {
		t.Fatalf("analysis tasks = %d, want 2", len(analysisTasks))
	}
	for _, task := range analysisTasks {
		if task.Status != model.TaskStatusPendingApproval {
			t.Errorf("analysis task %q status = %q, want %q", task.Title, task.Status, model.TaskStatusPendingApproval)
		}
	}

	designTasks, _, _ := f.TaskSvc.List(ctx, repository.TaskListParams{PhaseID: designPhase.ID, Limit: 100})
	if len(designTasks) != 3 {
		t.Fatalf("design tasks = %d, want 3", len(designTasks))
	}
	for _, task := range designTasks {
		if task.Status != model.TaskStatusPending {
			t.Errorf("design task %q status = %q, want %q", task.Title, task.Status, model.TaskStatusPending)
		}
	}

	// Only the first phase should have been reviewed and advanced to pending_approval.
	for _, task := range allTasks {
		if task.PhaseID == nil || *task.PhaseID == analysisPhase.ID {
			continue
		}
		if task.Status != model.TaskStatusPending {
			t.Errorf("downstream task %q status = %q, want %q", task.Title, task.Status, model.TaskStatusPending)
		}
	}

	// ---- Verify Artifacts ----
	artCount := f.ArtifactRepo.CountByProject(proj.ID)
	if artCount != 2 {
		t.Errorf("artifacts = %d, want 2", artCount)
	}
	verCount := f.ArtifactVersionRepo.TotalCount()
	if verCount != 2 {
		t.Errorf("artifact versions = %d, want 2", verCount)
	}
	artifacts, _, err := f.ArtifactSvc.List(ctx, repository.ArtifactListParams{ProjectID: proj.ID, Limit: 100, Offset: 0})
	if err != nil {
		t.Fatalf("list artifacts: %v", err)
	}
	for _, artifact := range artifacts {
		versions, err := f.ArtifactSvc.ListVersions(ctx, artifact.ID)
		if err != nil {
			t.Fatalf("list artifact versions: %v", err)
		}
		if len(versions) == 0 {
			t.Fatalf("artifact %q has no versions", artifact.Name)
		}
		if versions[0].URI == "" {
			t.Fatalf("artifact %q latest version missing uri", artifact.Name)
		}
		var meta map[string]any
		if err := json.Unmarshal(versions[0].Metadata, &meta); err != nil {
			t.Fatalf("unmarshal artifact version metadata: %v", err)
		}
		if meta["inline_content"] == "" {
			t.Fatalf("artifact %q missing inline_content metadata", artifact.Name)
		}
	}

	// ---- Verify Reviews ----
	reviewCount := f.ReviewRepo.TotalCount()
	if reviewCount != 2 {
		t.Errorf("reviews = %d, want 2", reviewCount)
	}
	reviews, total, _ := f.ReviewRepo.List(ctx, repository.ReviewListParams{RunID: run.ID, Limit: 100, Offset: 0})
	if total != 2 {
		t.Errorf("reviews for run = %d, want 2", total)
	}
	for _, review := range reviews {
		if review.RunID == nil || *review.RunID != run.ID {
			t.Errorf("review %q missing run_id %q", review.ID, run.ID)
		}
		if strings.Contains(review.Summary, "0 个交付物") {
			t.Errorf("review %q summary = %q, want artifact count > 0", review.ID, review.Summary)
		}
		var meta struct {
			ArtifactCount int `json:"artifact_count"`
		}
		if err := json.Unmarshal(review.Metadata, &meta); err != nil {
			t.Fatalf("unmarshal review metadata: %v", err)
		}
		if meta.ArtifactCount <= 0 {
			t.Errorf("review %q artifact_count = %d, want > 0", review.ID, meta.ArtifactCount)
		}
	}

	// ---- Verify Workflow Steps ----
	// 1 PM step + 2*(supervisor+worker+reviewer) = 7
	steps := f.WorkflowStepRepo.StepsForRun(run.ID)
	if len(steps) != 7 {
		t.Errorf("workflow steps = %d, want 7", len(steps))
	}
	// All steps should be completed
	for _, step := range steps {
		if step.Status != model.WorkflowStepCompleted {
			t.Errorf("step %q status = %q, want %q", step.Name, step.Status, model.WorkflowStepCompleted)
		}
	}

	// ---- Verify Audit Log has entries ----
	entries := f.AuditRepo.Entries()
	if len(entries) < 10 {
		t.Errorf("audit entries = %d, expected at least 10", len(entries))
	}

	// Check for workflow.started and workflow.waiting_approval events
	hasStarted, hasWaiting, hasCompleted := false, false, false
	for _, e := range entries {
		if e.EventType == "workflow.started" {
			hasStarted = true
		}
		if e.EventType == "workflow.waiting_approval" {
			hasWaiting = true
		}
		if e.EventType == "workflow.completed" {
			hasCompleted = true
		}
	}
	if !hasStarted {
		t.Error("missing workflow.started audit event")
	}
	if !hasWaiting {
		t.Error("missing workflow.waiting_approval audit event")
	}
	if hasCompleted {
		t.Error("unexpected workflow.completed audit event")
	}
}

func TestIntegration_MissingReviewerPrecheckFails(t *testing.T) {
	ctx := context.Background()
	f := testutil.NewFixture()

	if err := f.SeedStandardAgents(ctx); err != nil {
		t.Fatalf("seed agents: %v", err)
	}
	agents, _, err := f.AgentSvc.List(ctx, 100, 0)
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	for _, agent := range agents {
		if agent.Role != model.AgentRoleReviewer {
			continue
		}
		agent.Status = model.AgentStatusInactive
		if err := f.AgentSvc.Update(ctx, agent); err != nil {
			t.Fatalf("disable reviewer: %v", err)
		}
	}

	proj := &model.Project{Name: "缺评审者项目", Description: "验证工作流启动前校验"}
	if err := f.ProjectSvc.Create(ctx, proj); err != nil {
		t.Fatalf("create project: %v", err)
	}

	_, err = f.Engine.Run(ctx, proj.ID)
	if !errors.Is(err, orchestrator.ErrWorkflowPrecheckFailed) {
		t.Fatalf("expected ErrWorkflowPrecheckFailed, got %v", err)
	}
	runs, total, err := f.WorkflowSvc.ListRuns(ctx, repository.WorkflowRunListParams{ProjectID: proj.ID, Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if total != 0 || len(runs) != 0 {
		t.Fatalf("expected no workflow runs created, got total=%d len=%d", total, len(runs))
	}
}

// ---------- Rework Path ----------

// reworkReviewer rejects each task's first review, then approves on the next.
type reworkReviewer struct {
	mu         sync.Mutex
	perTask    map[string]int32 // taskID → call count
	totalCalls atomic.Int32
}

func newReworkReviewer() *reworkReviewer {
	return &reworkReviewer{perTask: make(map[string]int32)}
}

func (r *reworkReviewer) Role() string { return "reviewer" }

func (r *reworkReviewer) Execute(input *runtime.AgentTaskInput) (*runtime.AgentTaskOutput, error) {
	r.totalCalls.Add(1)

	taskID := ""
	if input.Task != nil {
		taskID = input.Task.ID
	}

	r.mu.Lock()
	r.perTask[taskID]++
	n := r.perTask[taskID]
	r.mu.Unlock()

	if n <= 1 { // reject first review per task
		return &runtime.AgentTaskOutput{
			Status:  runtime.OutputStatusSuccess,
			Summary: "评审打回，需修改",
			Reviews: []runtime.ReviewAction{{
				Verdict:         "needs_revision",
				Summary:         "交付物不满足要求，需返工",
				Findings:        []string{"内容不完整"},
				Recommendations: []string{"请补充细节"},
			}},
		}, nil
	}
	return &runtime.AgentTaskOutput{
		Status:  runtime.OutputStatusSuccess,
		Summary: "评审通过",
		Reviews: []runtime.ReviewAction{{
			Verdict:         "approved",
			Summary:         "交付物已通过评审",
			Findings:        []string{"内容完整"},
			Recommendations: []string{},
		}},
	}, nil
}

// alwaysRejectReviewer always returns needs_revision.
type alwaysRejectReviewer struct{}

func (r *alwaysRejectReviewer) Role() string { return "reviewer" }

func (r *alwaysRejectReviewer) Execute(_ *runtime.AgentTaskInput) (*runtime.AgentTaskOutput, error) {
	return &runtime.AgentTaskOutput{
		Status:  runtime.OutputStatusSuccess,
		Summary: "评审打回",
		Reviews: []runtime.ReviewAction{{
			Verdict:         "needs_revision",
			Summary:         "不合格，需返工",
			Findings:        []string{"内容不完整"},
			Recommendations: []string{"请补充"},
		}},
	}, nil
}

type failOnceWorker struct {
	mu      sync.Mutex
	failed  bool
	fallback runtime.AgentRunner
}

func (w *failOnceWorker) Role() string { return "worker" }

func (w *failOnceWorker) Execute(input *runtime.AgentTaskInput) (*runtime.AgentTaskOutput, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.failed {
		w.failed = true
		return &runtime.AgentTaskOutput{Status: runtime.OutputStatusFailed, Error: "LLM call failed: upstream timeout"}, nil
	}
	return w.fallback.Execute(input)
}

func TestIntegration_LLMFailureWaitsForManualInterventionAndResume(t *testing.T) {
	ctx := context.Background()
	f := testutil.NewFixture()

	if err := f.SeedStandardAgents(ctx); err != nil {
		t.Fatalf("seed agents: %v", err)
	}

	baseWorker, err := f.Registry.Get("worker")
	if err != nil {
		t.Fatalf("get base worker: %v", err)
	}
	f.Registry.Register("worker", &failOnceWorker{fallback: baseWorker})

	proj := &model.Project{Name: "人工介入恢复项目", Description: "验证 LLM 失败挂起与恢复"}
	if err := f.ProjectSvc.Create(ctx, proj); err != nil {
		t.Fatalf("create project: %v", err)
	}

	run, err := f.Engine.Run(ctx, proj.ID)
	if err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}
	if run.Status != model.WorkflowRunWaitingManual {
		t.Fatalf("run status = %q, want %q", run.Status, model.WorkflowRunWaitingManual)
	}
	if run.Error == "" {
		t.Fatal("expected run error to capture LLM failure reason")
	}

	steps := f.WorkflowStepRepo.StepsForRun(run.ID)
	if len(steps) == 0 || steps[len(steps)-1].Status != model.WorkflowStepFailed {
		t.Fatal("expected last workflow step to be failed before manual resume")
	}

	hasWaitingManual := false
	for _, entry := range f.AuditRepo.Entries() {
		if entry.EventType == "workflow.waiting_manual_intervention" {
			hasWaitingManual = true
			break
		}
	}
	if !hasWaitingManual {
		t.Fatal("missing workflow.waiting_manual_intervention audit event")
	}

	if err := f.Engine.ResumeRun(ctx, run.ID); err != nil {
		t.Fatalf("ResumeRun: %v", err)
	}

	finalRun, err := f.WorkflowSvc.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun after resume: %v", err)
	}
	if finalRun == nil {
		t.Fatal("run not found after resume")
	}
	if finalRun.Status != model.WorkflowRunWaitingApproval {
		t.Fatalf("final run status = %q, want %q", finalRun.Status, model.WorkflowRunWaitingApproval)
	}

	hasResumed := false
	for _, entry := range f.AuditRepo.Entries() {
		if entry.EventType == "workflow.resumed" {
			hasResumed = true
			break
		}
	}
	if !hasResumed {
		t.Fatal("missing workflow.resumed audit event")
	}
}

func TestIntegration_ReworkPath(t *testing.T) {
	ctx := context.Background()
	f := testutil.NewFixture()

	// Replace the reviewer with one that rejects each task's first review.
	reviewer := newReworkReviewer()
	f.Registry.Register("reviewer", reviewer)

	if err := f.SeedStandardAgents(ctx); err != nil {
		t.Fatalf("seed agents: %v", err)
	}

	proj := &model.Project{Name: "返工测试项目", Description: "测试评审打回与返工路径"}
	if err := f.ProjectSvc.Create(ctx, proj); err != nil {
		t.Fatalf("create project: %v", err)
	}

	run, err := f.Engine.Run(ctx, proj.ID)
	if err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}

	if run.Status != model.WorkflowRunWaitingApproval {
		t.Errorf("run status = %q, want %q", run.Status, model.WorkflowRunWaitingApproval)
	}

	// First phase has 2 tasks; each task should have 2 reviews (1 reject + 1 approve)
	reviewCount := f.ReviewRepo.TotalCount()
	if reviewCount != 4 {
		t.Errorf("reviews = %d, want 4", reviewCount)
	}

	// Each executed task should have 2 artifacts (1 per worker execution)
	artCount := f.ArtifactRepo.CountByProject(proj.ID)
	if artCount != 4 {
		t.Errorf("artifacts = %d, want 4", artCount)
	}

	// Verify reviewer was called the expected number of times
	totalCalls := reviewer.totalCalls.Load()
	if totalCalls != 4 {
		t.Errorf("reviewer calls = %d, want 4", totalCalls)
	}

	// Check for rework audit events
	reworkCount := 0
	for _, e := range f.AuditRepo.Entries() {
		if e.EventType == "task.rework" {
			reworkCount++
		}
	}
	if reworkCount != 2 {
		t.Errorf("rework events = %d, want 2", reworkCount)
	}

	// Steps: PM(1) + 2 tasks * (supervisor + worker + reviewer + worker + reviewer) = 11
	steps := f.WorkflowStepRepo.StepsForRun(run.ID)
	if len(steps) != 11 {
		t.Errorf("workflow steps = %d, want 11", len(steps))
	}
}

// ---------- Max Rework Exceeded ----------

func TestIntegration_MaxReworkExceeded(t *testing.T) {
	ctx := context.Background()
	f := testutil.NewFixture()

	// Reviewer always rejects → triggers max rework error after maxReworkAttempts
	alwaysReject := &alwaysRejectReviewer{}
	f.Registry.Register("reviewer", alwaysReject)

	if err := f.SeedStandardAgents(ctx); err != nil {
		t.Fatalf("seed agents: %v", err)
	}

	proj := &model.Project{Name: "超限返工项目", Description: "验证最大返工次数限制"}
	if err := f.ProjectSvc.Create(ctx, proj); err != nil {
		t.Fatalf("create project: %v", err)
	}

	run, err := f.Engine.Run(ctx, proj.ID)
	if err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}

	// The first task to exceed max rework attempts should fail the whole run.
	if run.Status != model.WorkflowRunFailed {
		t.Errorf("run status = %q, want %q", run.Status, model.WorkflowRunFailed)
	}

	// Only the first failing task should consume the full rework budget before the run aborts.
	reviewCount := f.ReviewRepo.TotalCount()
	if reviewCount != 4 {
		t.Errorf("reviews = %d, want 4", reviewCount)
	}
	hasFailed, hasCompleted := false, false
	for _, entry := range f.AuditRepo.Entries() {
		if entry.EventType == "workflow.failed" {
			hasFailed = true
		}
		if entry.EventType == "workflow.completed" {
			hasCompleted = true
		}
	}
	if !hasFailed {
		t.Error("missing workflow.failed audit event")
	}
	if hasCompleted {
		t.Error("unexpected workflow.completed audit event")
	}
}

// ---------- Audit Log Verification ----------

func TestIntegration_AuditLogCoverage(t *testing.T) {
	ctx := context.Background()
	f := testutil.NewFixture()

	if err := f.SeedStandardAgents(ctx); err != nil {
		t.Fatalf("seed agents: %v", err)
	}
	proj := &model.Project{Name: "审计测试项目", Description: "验证审计日志全面性"}
	if err := f.ProjectSvc.Create(ctx, proj); err != nil {
		t.Fatalf("create project: %v", err)
	}

	_, err := f.Engine.Run(ctx, proj.ID)
	if err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}

	entries := f.AuditRepo.Entries()

	// Build a set of event types
	eventTypes := make(map[string]int)
	for _, e := range entries {
		eventTypes[e.EventType]++
	}

	// These event types should be present in a gated first-phase run
	required := []string{
		"workflow.started",
		"workflow.waiting_approval",
		"project.created",
		"phase.created",
		"task.created",
		"task.status_changed",
		"task_contract.created",
		"task_assignment.created",
		"artifact.created",
		"artifact_version.created",
		"handoff_snapshot.created",
		"review_report.created",
	}
	for _, ev := range required {
		if eventTypes[ev] == 0 {
			t.Errorf("missing audit event type: %s", ev)
		}
	}
}

// ---------- Artifact & Version Assertions ----------

func TestIntegration_ArtifactVersioning(t *testing.T) {
	ctx := context.Background()
	f := testutil.NewFixture()

	// Use rework reviewer so each task gets 2 worker executions → 2 artifacts + 2 versions each
	reviewer := newReworkReviewer()
	f.Registry.Register("reviewer", reviewer)

	if err := f.SeedStandardAgents(ctx); err != nil {
		t.Fatalf("seed agents: %v", err)
	}
	proj := &model.Project{Name: "版本测试项目", Description: "验证工件版本递增"}
	if err := f.ProjectSvc.Create(ctx, proj); err != nil {
		t.Fatalf("create project: %v", err)
	}

	_, err := f.Engine.Run(ctx, proj.ID)
	if err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}

	artCount := f.ArtifactRepo.CountByProject(proj.ID)
	verCount := f.ArtifactVersionRepo.TotalCount()

	// First phase has 2 tasks; each task runs worker twice = 4 artifacts, 4 versions
	if artCount != 4 {
		t.Errorf("artifacts = %d, want 4", artCount)
	}
	if verCount != 4 {
		t.Errorf("artifact versions = %d, want 4", verCount)
	}
}

// end of integration tests

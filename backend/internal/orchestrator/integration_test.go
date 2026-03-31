package orchestrator_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lingchou/lingchoubot/backend/internal/model"
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
	if run.Status != model.WorkflowRunCompleted {
		t.Errorf("run status = %q, want %q", run.Status, model.WorkflowRunCompleted)
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

	// Every task should have been reviewed (status in_review, since mock reviewer approves)
	for _, task := range allTasks {
		if task.Status != model.TaskStatusInReview {
			t.Errorf("task %q status = %q, want %q", task.Title, task.Status, model.TaskStatusInReview)
		}
	}

	// ---- Verify Artifacts ----
	artCount := f.ArtifactRepo.CountByProject(proj.ID)
	if artCount != 9 {
		t.Errorf("artifacts = %d, want 9", artCount)
	}
	verCount := f.ArtifactVersionRepo.TotalCount()
	if verCount != 9 {
		t.Errorf("artifact versions = %d, want 9", verCount)
	}

	// ---- Verify Reviews ----
	reviewCount := f.ReviewRepo.TotalCount()
	if reviewCount != 9 {
		t.Errorf("reviews = %d, want 9", reviewCount)
	}

	// ---- Verify Workflow Steps ----
	// 1 PM step + 9*(supervisor+worker+reviewer) = 1 + 27 = 28
	steps := f.WorkflowStepRepo.StepsForRun(run.ID)
	if len(steps) != 28 {
		t.Errorf("workflow steps = %d, want 28", len(steps))
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

	// Check for workflow.started and workflow.completed events
	hasStarted, hasCompleted := false, false
	for _, e := range entries {
		if e.EventType == "workflow.started" {
			hasStarted = true
		}
		if e.EventType == "workflow.completed" {
			hasCompleted = true
		}
	}
	if !hasStarted {
		t.Error("missing workflow.started audit event")
	}
	if !hasCompleted {
		t.Error("missing workflow.completed audit event")
	}
}

// ---------- Rework Path ----------

// reworkReviewer rejects each task's first review, then approves on the next.
type reworkReviewer struct {
	mu       sync.Mutex
	perTask  map[string]int32 // taskID → call count
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

	if run.Status != model.WorkflowRunCompleted {
		t.Errorf("run status = %q, want %q", run.Status, model.WorkflowRunCompleted)
	}

	// Each task should have 2 reviews (1 reject + 1 approve)
	reviewCount := f.ReviewRepo.TotalCount()
	if reviewCount != 18 { // 9 tasks * 2 reviews each
		t.Errorf("reviews = %d, want 18", reviewCount)
	}

	// Each task should have 2 artifacts (1 per worker execution)
	artCount := f.ArtifactRepo.CountByProject(proj.ID)
	if artCount != 18 { // 9 tasks * 2 worker runs
		t.Errorf("artifacts = %d, want 18", artCount)
	}

	// Verify reviewer was called the expected number of times
	totalCalls := reviewer.totalCalls.Load()
	if totalCalls != 18 { // 9 first-pass rejects + 9 second-pass approves
		t.Errorf("reviewer calls = %d, want 18", totalCalls)
	}

	// Check for rework audit events
	reworkCount := 0
	for _, e := range f.AuditRepo.Entries() {
		if e.EventType == "task.rework" {
			reworkCount++
		}
	}
	if reworkCount != 9 {
		t.Errorf("rework events = %d, want 9", reworkCount)
	}

	// Steps: PM(1) + 9 tasks * (supervisor + worker + reviewer + worker + reviewer) = 1 + 45 = 46
	steps := f.WorkflowStepRepo.StepsForRun(run.ID)
	if len(steps) != 46 {
		t.Errorf("workflow steps = %d, want 46", len(steps))
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

	// Run should still complete (errors in individual task chains are logged, not propagated to Run)
	if run.Status != model.WorkflowRunCompleted {
		t.Errorf("run status = %q, want %q", run.Status, model.WorkflowRunCompleted)
	}

	// Each task should have (maxReworkAttempts+1)=4 reviews (all needs_revision)
	reviewCount := f.ReviewRepo.TotalCount()
	// 9 tasks * 4 review rounds = 36
	if reviewCount != 36 {
		t.Errorf("reviews = %d, want 36", reviewCount)
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

	// These event types should be present in a happy-path run
	required := []string{
		"workflow.started",
		"workflow.completed",
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

	// 9 tasks * 2 worker runs = 18 artifacts, 18 versions
	if artCount != 18 {
		t.Errorf("artifacts = %d, want 18", artCount)
	}
	if verCount != 18 {
		t.Errorf("artifact versions = %d, want 18", verCount)
	}
}

// end of integration tests

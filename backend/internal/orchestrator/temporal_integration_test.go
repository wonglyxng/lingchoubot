package orchestrator_test

import (
	"context"
	"testing"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/orchestrator"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
	"github.com/lingchou/lingchoubot/backend/internal/testutil"
	"go.temporal.io/sdk/testsuite"
)

// TestTemporal_WorkflowHappyPath uses Temporal's test environment to exercise
// the ProjectWorkflow with real activities backed by in-memory fake repos.
func TestTemporal_WorkflowHappyPath(t *testing.T) {
	ctx := context.Background()
	f := testutil.NewFixture()

	if err := f.SeedStandardAgents(ctx); err != nil {
		t.Fatalf("seed agents: %v", err)
	}

	proj := &model.Project{Name: "Temporal集成测试项目", Description: "验证 Temporal 工作流的 Happy Path"}
	if err := f.ProjectSvc.Create(ctx, proj); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// Create a workflow run so activities can reference it
	run, err := f.WorkflowSvc.CreateRun(ctx, proj.ID)
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	// Build Activities struct backed by fakes
	activities := &orchestrator.Activities{
		Registry: f.Registry,
		Services: &orchestrator.Services{
			Project:    f.ProjectSvc,
			Phase:      f.PhaseSvc,
			Agent:      f.AgentSvc,
			Task:       f.TaskSvc,
			Contract:   f.ContractSvc,
			Assignment: f.AssignmentSvc,
			Artifact:   f.ArtifactSvc,
			Handoff:    f.HandoffSvc,
			Review:     f.ReviewSvc,
			Approval:   f.ApprovalSvc,
			Audit:      f.AuditSvc,
		},
		Workflow: f.WorkflowSvc,
		Logger:   f.Logger,
	}

	// Set up Temporal test environment
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	// Register activities
	env.RegisterActivity(activities.ActivityPM)
	env.RegisterActivity(activities.ActivityListPhaseTasks)
	env.RegisterActivity(activities.ActivitySupervisor)
	env.RegisterActivity(activities.ActivityWorker)
	env.RegisterActivity(activities.ActivityReviewer)
	env.RegisterActivity(activities.ActivityCheckRework)
	env.RegisterActivity(activities.ActivityCompleteRun)
	env.RegisterActivity(activities.ActivityFailRun)

	// Execute workflow
	input := orchestrator.ProjectWorkflowInput{
		RunID:     run.ID,
		ProjectID: proj.ID,
	}
	env.ExecuteWorkflow(orchestrator.ProjectWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error: %v", err)
	}

	// ---- Verify results ----

	// Phases: MockPM creates 4 phases
	phases, _ := f.PhaseSvc.ListByProject(ctx, proj.ID)
	if len(phases) != 4 {
		t.Errorf("phases = %d, want 4", len(phases))
	}

	// Tasks: MockPM creates 9 tasks
	tasks, _, _ := f.TaskSvc.List(ctx, repository.TaskListParams{ProjectID: proj.ID, Limit: 100})
	if len(tasks) != 9 {
		t.Errorf("tasks = %d, want 9", len(tasks))
	}

	// Artifacts: 9 tasks * 1 worker run each = 9
	artCount := f.ArtifactRepo.CountByProject(proj.ID)
	if artCount != 9 {
		t.Errorf("artifacts = %d, want 9", artCount)
	}

	// Reviews: 9
	reviewCount := f.ReviewRepo.TotalCount()
	if reviewCount != 9 {
		t.Errorf("reviews = %d, want 9", reviewCount)
	}
	reviews, total, _ := f.ReviewRepo.List(ctx, repository.ReviewListParams{RunID: run.ID, Limit: 100, Offset: 0})
	if total != 9 {
		t.Errorf("reviews for run = %d, want 9", total)
	}
	for _, review := range reviews {
		if review.RunID == nil || *review.RunID != run.ID {
			t.Errorf("review %q missing run_id %q", review.ID, run.ID)
		}
	}

	// Workflow run should be completed
	finalRun, _ := f.WorkflowSvc.GetRun(ctx, run.ID)
	if finalRun == nil {
		t.Fatal("run not found")
	}
	if finalRun.Status != model.WorkflowRunCompleted {
		t.Errorf("run status = %q, want %q", finalRun.Status, model.WorkflowRunCompleted)
	}

	// Workflow steps should exist
	steps := f.WorkflowStepRepo.StepsForRun(run.ID)
	if len(steps) < 20 {
		t.Errorf("workflow steps = %d, expected at least 20", len(steps))
	}

	// Audit entries
	entries := f.AuditRepo.Entries()
	if len(entries) < 10 {
		t.Errorf("audit entries = %d, expected at least 10", len(entries))
	}
}

func TestTemporal_WorkflowFailsWithoutReviewer(t *testing.T) {
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

	proj := &model.Project{Name: "Temporal缺评审者项目", Description: "验证 Temporal 失败收口"}
	if err := f.ProjectSvc.Create(ctx, proj); err != nil {
		t.Fatalf("create project: %v", err)
	}
	run, err := f.WorkflowSvc.CreateRun(ctx, proj.ID)
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	activities := &orchestrator.Activities{
		Registry: f.Registry,
		Services: &orchestrator.Services{
			Project:    f.ProjectSvc,
			Phase:      f.PhaseSvc,
			Agent:      f.AgentSvc,
			Task:       f.TaskSvc,
			Contract:   f.ContractSvc,
			Assignment: f.AssignmentSvc,
			Artifact:   f.ArtifactSvc,
			Handoff:    f.HandoffSvc,
			Review:     f.ReviewSvc,
			Approval:   f.ApprovalSvc,
			Audit:      f.AuditSvc,
		},
		Workflow: f.WorkflowSvc,
		Logger:   f.Logger,
	}

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivity(activities.ActivityPM)
	env.RegisterActivity(activities.ActivityListPhaseTasks)
	env.RegisterActivity(activities.ActivitySupervisor)
	env.RegisterActivity(activities.ActivityWorker)
	env.RegisterActivity(activities.ActivityReviewer)
	env.RegisterActivity(activities.ActivityCheckRework)
	env.RegisterActivity(activities.ActivityCompleteRun)
	env.RegisterActivity(activities.ActivityFailRun)
	env.ExecuteWorkflow(orchestrator.ProjectWorkflow, orchestrator.ProjectWorkflowInput{RunID: run.ID, ProjectID: proj.ID})

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err == nil {
		t.Fatal("expected workflow error, got nil")
	}
	finalRun, _ := f.WorkflowSvc.GetRun(ctx, run.ID)
	if finalRun == nil {
		t.Fatal("run not found")
	}
	if finalRun.Status != model.WorkflowRunFailed {
		t.Errorf("run status = %q, want %q", finalRun.Status, model.WorkflowRunFailed)
	}
}

func TestTemporal_WorkflowWaitsForManualInterventionOnLLMFailure(t *testing.T) {
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

	proj := &model.Project{Name: "Temporal人工介入项目", Description: "验证 Temporal LLM 失败挂起"}
	if err := f.ProjectSvc.Create(ctx, proj); err != nil {
		t.Fatalf("create project: %v", err)
	}
	run, err := f.WorkflowSvc.CreateRun(ctx, proj.ID)
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	activities := &orchestrator.Activities{
		Registry: f.Registry,
		Services: &orchestrator.Services{
			Project:    f.ProjectSvc,
			Phase:      f.PhaseSvc,
			Agent:      f.AgentSvc,
			Task:       f.TaskSvc,
			Contract:   f.ContractSvc,
			Assignment: f.AssignmentSvc,
			Artifact:   f.ArtifactSvc,
			Handoff:    f.HandoffSvc,
			Review:     f.ReviewSvc,
			Approval:   f.ApprovalSvc,
			Audit:      f.AuditSvc,
		},
		Workflow: f.WorkflowSvc,
		Logger:   f.Logger,
	}

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivity(activities.ActivityPM)
	env.RegisterActivity(activities.ActivityListPhaseTasks)
	env.RegisterActivity(activities.ActivitySupervisor)
	env.RegisterActivity(activities.ActivityWorker)
	env.RegisterActivity(activities.ActivityReviewer)
	env.RegisterActivity(activities.ActivityCheckRework)
	env.RegisterActivity(activities.ActivityCompleteRun)
	env.RegisterActivity(activities.ActivityFailRun)
	env.ExecuteWorkflow(orchestrator.ProjectWorkflow, orchestrator.ProjectWorkflowInput{RunID: run.ID, ProjectID: proj.ID})

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("expected no workflow error, got %v", err)
	}
	finalRun, _ := f.WorkflowSvc.GetRun(ctx, run.ID)
	if finalRun == nil {
		t.Fatal("run not found")
	}
	if finalRun.Status != model.WorkflowRunWaitingManual {
		t.Errorf("run status = %q, want %q", finalRun.Status, model.WorkflowRunWaitingManual)
	}
	if finalRun.Error == "" {
		t.Fatal("expected waiting manual run to keep error reason")
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
}

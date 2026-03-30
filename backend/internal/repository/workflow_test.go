package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lingchou/lingchoubot/backend/internal/model"
)

// --------------- WorkflowRunRepo ---------------

func TestWorkflowRunRepo_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewWorkflowRunRepo(db)
	now := time.Now().Truncate(time.Microsecond)

	run := &model.WorkflowRun{
		ProjectID: "proj-1",
		Status:    model.WorkflowRunRunning,
		Summary:   "initial",
		Error:     "",
		StartedAt: now,
	}

	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
		AddRow("run-uuid", now, now)

	mock.ExpectQuery(`INSERT INTO workflow_run`).
		WithArgs(run.ProjectID, run.Status, run.Summary, run.Error, run.StartedAt).
		WillReturnRows(rows)

	if err := repo.Create(context.Background(), run); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if run.ID != "run-uuid" {
		t.Fatalf("expected ID %q, got %q", "run-uuid", run.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestWorkflowRunRepo_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewWorkflowRunRepo(db)
	now := time.Now().Truncate(time.Microsecond)

	rows := sqlmock.NewRows([]string{
		"id", "project_id", "status", "summary", "error",
		"started_at", "completed_at", "created_at", "updated_at",
	}).AddRow("run-1", "proj-1", "running", "test", "", now, nil, now, now)

	mock.ExpectQuery(`SELECT .+ FROM workflow_run WHERE id = \$1`).
		WithArgs("run-1").
		WillReturnRows(rows)

	run, err := repo.GetByID(context.Background(), "run-1")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if run == nil {
		t.Fatal("expected run, got nil")
	}
	if run.ProjectID != "proj-1" {
		t.Fatalf("expected ProjectID %q, got %q", "proj-1", run.ProjectID)
	}
	if run.Status != model.WorkflowRunRunning {
		t.Fatalf("expected status %q, got %q", model.WorkflowRunRunning, run.Status)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestWorkflowRunRepo_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewWorkflowRunRepo(db)

	rows := sqlmock.NewRows([]string{
		"id", "project_id", "status", "summary", "error",
		"started_at", "completed_at", "created_at", "updated_at",
	})

	mock.ExpectQuery(`SELECT .+ FROM workflow_run WHERE id = \$1`).
		WithArgs("nonexistent").
		WillReturnRows(rows)

	run, err := repo.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if run != nil {
		t.Fatalf("expected nil, got %+v", run)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestWorkflowRunRepo_UpdateStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewWorkflowRunRepo(db)
	now := time.Now().Truncate(time.Microsecond)

	run := &model.WorkflowRun{
		ID:          "run-1",
		Status:      model.WorkflowRunCompleted,
		Summary:     "done",
		Error:       "",
		CompletedAt: &now,
	}

	mock.ExpectExec(`UPDATE workflow_run`).
		WithArgs(run.ID, run.Status, run.Summary, run.Error, run.CompletedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.UpdateStatus(context.Background(), run); err != nil {
		t.Fatalf("UpdateStatus returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestWorkflowRunRepo_UpdateStatus_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewWorkflowRunRepo(db)

	run := &model.WorkflowRun{
		ID:     "nonexistent",
		Status: model.WorkflowRunFailed,
		Error:  "boom",
	}

	mock.ExpectExec(`UPDATE workflow_run`).
		WithArgs(run.ID, run.Status, run.Summary, run.Error, run.CompletedAt).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := repo.UpdateStatus(context.Background(), run); err == nil {
		t.Fatal("expected error for not-found run, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestWorkflowRunRepo_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewWorkflowRunRepo(db)
	now := time.Now().Truncate(time.Microsecond)

	// count query
	mock.ExpectQuery(`SELECT count\(\*\) FROM workflow_run`).
		WithArgs("proj-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// list query
	listRows := sqlmock.NewRows([]string{
		"id", "project_id", "status", "summary", "error",
		"started_at", "completed_at", "created_at", "updated_at",
	}).
		AddRow("run-1", "proj-1", "completed", "s1", "", now, &now, now, now).
		AddRow("run-2", "proj-1", "running", "s2", "", now, nil, now, now)

	mock.ExpectQuery(`SELECT .+ FROM workflow_run`).
		WithArgs("proj-1", 10, 0).
		WillReturnRows(listRows)

	runs, total, err := repo.List(context.Background(), WorkflowRunListParams{
		ProjectID: "proj-1",
		Limit:     10,
		Offset:    0,
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(runs))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestWorkflowRunRepo_ListWithStatusFilter(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewWorkflowRunRepo(db)
	now := time.Now().Truncate(time.Microsecond)

	mock.ExpectQuery(`SELECT count\(\*\) FROM workflow_run`).
		WithArgs("proj-1", "running").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	listRows := sqlmock.NewRows([]string{
		"id", "project_id", "status", "summary", "error",
		"started_at", "completed_at", "created_at", "updated_at",
	}).AddRow("run-2", "proj-1", "running", "s2", "", now, nil, now, now)

	mock.ExpectQuery(`SELECT .+ FROM workflow_run`).
		WithArgs("proj-1", "running", 10, 0).
		WillReturnRows(listRows)

	runs, total, err := repo.List(context.Background(), WorkflowRunListParams{
		ProjectID: "proj-1",
		Status:    "running",
		Limit:     10,
		Offset:    0,
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// --------------- WorkflowStepRepo ---------------

func TestWorkflowStepRepo_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewWorkflowStepRepo(db)
	now := time.Now().Truncate(time.Microsecond)

	step := &model.WorkflowStep{
		RunID:     "run-1",
		Name:      "PM 分解",
		AgentRole: "pm",
		Status:    model.WorkflowStepPending,
		SortOrder: 1,
	}

	rows := sqlmock.NewRows([]string{"id", "created_at"}).
		AddRow("step-uuid", now)

	mock.ExpectQuery(`INSERT INTO workflow_step`).
		WithArgs(
			step.RunID, step.Name, step.AgentRole, step.AgentID, step.TaskID, step.PhaseID,
			step.Status, step.Summary, step.Error, step.SortOrder, step.StartedAt, step.CompletedAt,
		).
		WillReturnRows(rows)

	if err := repo.Create(context.Background(), step); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if step.ID != "step-uuid" {
		t.Fatalf("expected ID %q, got %q", "step-uuid", step.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestWorkflowStepRepo_UpdateStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewWorkflowStepRepo(db)
	now := time.Now().Truncate(time.Microsecond)

	step := &model.WorkflowStep{
		ID:          "step-1",
		Status:      model.WorkflowStepCompleted,
		Summary:     "done",
		Error:       "",
		AgentID:     strPtr("agent-1"),
		TaskID:      strPtr("task-1"),
		PhaseID:     strPtr("phase-1"),
		StartedAt:   &now,
		CompletedAt: &now,
	}

	mock.ExpectExec(`UPDATE workflow_step`).
		WithArgs(step.ID, step.Status, step.Summary, step.Error, step.AgentID,
			step.TaskID, step.PhaseID, step.StartedAt, step.CompletedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.UpdateStatus(context.Background(), step); err != nil {
		t.Fatalf("UpdateStatus returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestWorkflowStepRepo_UpdateStatus_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewWorkflowStepRepo(db)

	step := &model.WorkflowStep{
		ID:     "nonexistent",
		Status: model.WorkflowStepFailed,
		Error:  "something went wrong",
	}

	mock.ExpectExec(`UPDATE workflow_step`).
		WithArgs(step.ID, step.Status, step.Summary, step.Error, step.AgentID,
			step.TaskID, step.PhaseID, step.StartedAt, step.CompletedAt).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := repo.UpdateStatus(context.Background(), step); err == nil {
		t.Fatal("expected error for not-found step, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestWorkflowStepRepo_ListByRunID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewWorkflowStepRepo(db)
	now := time.Now().Truncate(time.Microsecond)

	agentID := "agent-1"
	rows := sqlmock.NewRows([]string{
		"id", "run_id", "name", "agent_role", "agent_id", "task_id", "phase_id",
		"status", "summary", "error", "sort_order", "started_at", "completed_at", "created_at",
	}).
		AddRow("step-1", "run-1", "PM 分解", "pm", &agentID, nil, nil,
			"completed", "ok", "", 1, &now, &now, now).
		AddRow("step-2", "run-1", "Reviewer 评审", "reviewer", nil, nil, nil,
			"pending", "", "", 2, nil, nil, now)

	mock.ExpectQuery(`SELECT .+ FROM workflow_step WHERE run_id = \$1`).
		WithArgs("run-1").
		WillReturnRows(rows)

	steps, err := repo.ListByRunID(context.Background(), "run-1")
	if err != nil {
		t.Fatalf("ListByRunID returned error: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if steps[0].Name != "PM 分解" {
		t.Fatalf("expected first step name %q, got %q", "PM 分解", steps[0].Name)
	}
	if steps[0].AgentID == nil || *steps[0].AgentID != "agent-1" {
		t.Fatal("expected first step agent_id to be agent-1")
	}
	if steps[1].AgentID != nil {
		t.Fatal("expected second step agent_id to be nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestWorkflowStepRepo_ListByRunID_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewWorkflowStepRepo(db)

	rows := sqlmock.NewRows([]string{
		"id", "run_id", "name", "agent_role", "agent_id", "task_id", "phase_id",
		"status", "summary", "error", "sort_order", "started_at", "completed_at", "created_at",
	})

	mock.ExpectQuery(`SELECT .+ FROM workflow_step WHERE run_id = \$1`).
		WithArgs("run-empty").
		WillReturnRows(rows)

	steps, err := repo.ListByRunID(context.Background(), "run-empty")
	if err != nil {
		t.Fatalf("ListByRunID returned error: %v", err)
	}
	if len(steps) != 0 {
		t.Fatalf("expected 0 steps, got %d", len(steps))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// strPtr is a test helper for *string fields.
func strPtr(s string) *string { return &s }

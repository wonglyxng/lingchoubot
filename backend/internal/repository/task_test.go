package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lingchou/lingchoubot/backend/internal/model"
)

func TestTaskRepo_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewTaskRepo(db)
	now := time.Now().Truncate(time.Microsecond)

	task := &model.Task{
		ProjectID:   "proj-1",
		PhaseID:     nil,
		Title:       "设计 API 接口",
		Description: "完成 REST API 设计",
		Status:      model.TaskStatusPending,
		Priority:    3,
		Metadata:    model.JSON("{}"),
	}

	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
		AddRow("task-uuid", now, now)

	mock.ExpectQuery(`INSERT INTO task`).
		WithArgs(task.ProjectID, task.PhaseID, task.ParentTaskID, task.Title, task.Description,
			task.Status, task.Priority, task.AssigneeID, task.ExecutionDomain, task.OwnerSupervisorID,
			task.InputContext, task.OutputSummary, task.Metadata).
		WillReturnRows(rows)

	if err := repo.Create(context.Background(), task); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if task.ID != "task-uuid" {
		t.Fatalf("expected ID %q, got %q", "task-uuid", task.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTaskRepo_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewTaskRepo(db)
	now := time.Now().Truncate(time.Microsecond)

	rows := sqlmock.NewRows([]string{
		"id", "project_id", "phase_id", "parent_task_id", "title", "description",
		"status", "priority", "assignee_id", "execution_domain", "owner_supervisor_id",
		"input_context", "output_summary", "metadata",
		"created_at", "updated_at",
	}).AddRow(
		"task-1", "proj-1", nil, nil, "设计 API", "描述",
		"pending", 3, nil, "general", nil,
		[]byte("{}"), []byte("{}"), []byte("{}"),
		now, now,
	)

	mock.ExpectQuery(`SELECT .+ FROM task WHERE id = \$1`).
		WithArgs("task-1").
		WillReturnRows(rows)

	task, err := repo.GetByID(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if task == nil {
		t.Fatal("expected task, got nil")
	}
	if task.Title != "设计 API" {
		t.Fatalf("expected title %q, got %q", "设计 API", task.Title)
	}
	if task.Status != model.TaskStatusPending {
		t.Fatalf("expected status %q, got %q", model.TaskStatusPending, task.Status)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTaskRepo_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewTaskRepo(db)

	rows := sqlmock.NewRows([]string{
		"id", "project_id", "phase_id", "parent_task_id", "title", "description",
		"status", "priority", "assignee_id", "execution_domain", "owner_supervisor_id",
		"input_context", "output_summary", "metadata",
		"created_at", "updated_at",
	})

	mock.ExpectQuery(`SELECT .+ FROM task WHERE id = \$1`).
		WithArgs("nonexistent").
		WillReturnRows(rows)

	task, err := repo.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if task != nil {
		t.Fatalf("expected nil, got %+v", task)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTaskRepo_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewTaskRepo(db)
	now := time.Now().Truncate(time.Microsecond)

	mock.ExpectQuery(`SELECT count\(\*\) FROM task`).
		WithArgs("proj-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	listRows := sqlmock.NewRows([]string{
		"id", "project_id", "phase_id", "parent_task_id", "title", "description",
		"status", "priority", "assignee_id", "execution_domain", "owner_supervisor_id",
		"input_context", "output_summary", "metadata",
		"created_at", "updated_at",
	}).
		AddRow("t-1", "proj-1", nil, nil, "任务1", "描述1",
			"pending", 3, nil, "general", nil, []byte("{}"), []byte("{}"), []byte("{}"), now, now).
		AddRow("t-2", "proj-1", nil, nil, "任务2", "描述2",
			"assigned", 2, nil, "development", nil, []byte("{}"), []byte("{}"), []byte("{}"), now, now)

	mock.ExpectQuery(`SELECT .+ FROM task`).
		WithArgs("proj-1", 10, 0).
		WillReturnRows(listRows)

	tasks, total, err := repo.List(context.Background(), TaskListParams{
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
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTaskRepo_ListWithMultipleFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewTaskRepo(db)
	now := time.Now().Truncate(time.Microsecond)

	mock.ExpectQuery(`SELECT count\(\*\) FROM task`).
		WithArgs("proj-1", "phase-1", "pending").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	listRows := sqlmock.NewRows([]string{
		"id", "project_id", "phase_id", "parent_task_id", "title", "description",
		"status", "priority", "assignee_id", "execution_domain", "owner_supervisor_id",
		"input_context", "output_summary", "metadata",
		"created_at", "updated_at",
	}).AddRow("t-1", "proj-1", strPtr("phase-1"), nil, "任务1", "描述1",
		"pending", 3, nil, "general", nil, []byte("{}"), []byte("{}"), []byte("{}"), now, now)

	mock.ExpectQuery(`SELECT .+ FROM task`).
		WithArgs("proj-1", "phase-1", "pending", 10, 0).
		WillReturnRows(listRows)

	tasks, total, err := repo.List(context.Background(), TaskListParams{
		ProjectID: "proj-1",
		PhaseID:   "phase-1",
		Status:    "pending",
		Limit:     10,
		Offset:    0,
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTaskRepo_UpdateStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewTaskRepo(db)

	mock.ExpectExec(`UPDATE task SET status`).
		WithArgs("task-1", model.TaskStatusAssigned).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.UpdateStatus(context.Background(), "task-1", model.TaskStatusAssigned); err != nil {
		t.Fatalf("UpdateStatus returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTaskRepo_UpdateStatus_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewTaskRepo(db)

	mock.ExpectExec(`UPDATE task SET status`).
		WithArgs("nonexistent", model.TaskStatusAssigned).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := repo.UpdateStatus(context.Background(), "nonexistent", model.TaskStatusAssigned); err == nil {
		t.Fatal("expected error for not-found task, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTaskRepo_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewTaskRepo(db)
	now := time.Now().Truncate(time.Microsecond)

	task := &model.Task{
		ID:          "task-1",
		Title:       "更新后的任务",
		Description: "更新后的描述",
		Status:      model.TaskStatusInProgress,
		Priority:    5,
		Metadata:    model.JSON("{}"),
	}

	rows := sqlmock.NewRows([]string{"updated_at"}).AddRow(now)

	mock.ExpectQuery(`UPDATE task`).
		WithArgs(task.ID, task.Title, task.Description, task.Status, task.Priority,
			task.PhaseID, task.ParentTaskID, task.AssigneeID,
			task.ExecutionDomain, task.OwnerSupervisorID,
			task.InputContext, task.OutputSummary, task.Metadata).
		WillReturnRows(rows)

	if err := repo.Update(context.Background(), task); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if task.UpdatedAt.IsZero() {
		t.Fatal("expected UpdatedAt to be set")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTaskRepo_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewTaskRepo(db)

	mock.ExpectExec(`DELETE FROM task WHERE id = \$1`).
		WithArgs("task-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.Delete(context.Background(), "task-1"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

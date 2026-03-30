package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lingchou/lingchoubot/backend/internal/model"
)

func TestApprovalRequestRepo_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewApprovalRequestRepo(db)
	now := time.Now().Truncate(time.Microsecond)
	taskID := "task-1"

	a := &model.ApprovalRequest{
		ProjectID:    "proj-1",
		TaskID:       &taskID,
		ArtifactID:   nil,
		RequestedBy:  "agent-pm",
		ApproverType: "agent",
		ApproverID:   "agent-reviewer",
		Title:        "审批交付物",
		Description:  "请评审此交付物",
		Status:       model.ApprovalStatusPending,
		Metadata:     model.JSON("{}"),
	}

	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
		AddRow("approval-uuid", now, now)

	mock.ExpectQuery(`INSERT INTO approval_request`).
		WithArgs(a.ProjectID, a.TaskID, a.ArtifactID, a.RequestedBy,
			a.ApproverType, a.ApproverID, a.Title, a.Description, a.Status, a.Metadata).
		WillReturnRows(rows)

	if err := repo.Create(context.Background(), a); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if a.ID != "approval-uuid" {
		t.Fatalf("expected ID %q, got %q", "approval-uuid", a.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestApprovalRequestRepo_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewApprovalRequestRepo(db)
	now := time.Now().Truncate(time.Microsecond)
	taskID := "task-1"

	rows := sqlmock.NewRows([]string{
		"id", "project_id", "task_id", "artifact_id", "requested_by",
		"approver_type", "approver_id", "title", "description", "status",
		"decision_note", "decided_at", "metadata", "created_at", "updated_at",
	}).AddRow(
		"a-1", "proj-1", &taskID, nil, "agent-pm",
		"agent", "agent-reviewer", "审批", "评审", "pending",
		"", nil, []byte("{}"), now, now,
	)

	mock.ExpectQuery(`SELECT .+ FROM approval_request WHERE id = \$1`).
		WithArgs("a-1").
		WillReturnRows(rows)

	a, err := repo.GetByID(context.Background(), "a-1")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if a == nil {
		t.Fatal("expected approval, got nil")
	}
	if a.ProjectID != "proj-1" {
		t.Fatalf("expected ProjectID %q, got %q", "proj-1", a.ProjectID)
	}
	if a.Status != model.ApprovalStatusPending {
		t.Fatalf("expected status %q, got %q", model.ApprovalStatusPending, a.Status)
	}
	if a.TaskID == nil || *a.TaskID != "task-1" {
		t.Fatal("expected TaskID to be task-1")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestApprovalRequestRepo_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewApprovalRequestRepo(db)

	rows := sqlmock.NewRows([]string{
		"id", "project_id", "task_id", "artifact_id", "requested_by",
		"approver_type", "approver_id", "title", "description", "status",
		"decision_note", "decided_at", "metadata", "created_at", "updated_at",
	})

	mock.ExpectQuery(`SELECT .+ FROM approval_request WHERE id = \$1`).
		WithArgs("nonexistent").
		WillReturnRows(rows)

	a, err := repo.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if a != nil {
		t.Fatalf("expected nil, got %+v", a)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestApprovalRequestRepo_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewApprovalRequestRepo(db)
	now := time.Now().Truncate(time.Microsecond)

	mock.ExpectQuery(`SELECT count\(\*\) FROM approval_request`).
		WithArgs("proj-1", "pending").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	listRows := sqlmock.NewRows([]string{
		"id", "project_id", "task_id", "artifact_id", "requested_by",
		"approver_type", "approver_id", "title", "description", "status",
		"decision_note", "decided_at", "metadata", "created_at", "updated_at",
	}).AddRow(
		"a-1", "proj-1", nil, nil, "pm",
		"agent", "reviewer", "审批", "评审", "pending",
		"", nil, []byte("{}"), now, now,
	)

	mock.ExpectQuery(`SELECT .+ FROM approval_request`).
		WithArgs("proj-1", "pending", 20, 0).
		WillReturnRows(listRows)

	approvals, total, err := repo.List(context.Background(), ApprovalListParams{
		ProjectID: "proj-1",
		Status:    "pending",
		Limit:     20,
		Offset:    0,
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(approvals) != 1 {
		t.Fatalf("expected 1 approval, got %d", len(approvals))
	}
	if approvals[0].Title != "审批" {
		t.Fatalf("expected title %q, got %q", "审批", approvals[0].Title)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestApprovalRequestRepo_Decide(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewApprovalRequestRepo(db)

	mock.ExpectExec(`UPDATE approval_request`).
		WithArgs("a-1", model.ApprovalStatusApproved, "LGTM").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.Decide(context.Background(), "a-1", model.ApprovalStatusApproved, "LGTM"); err != nil {
		t.Fatalf("Decide returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestApprovalRequestRepo_Decide_AlreadyDecided(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewApprovalRequestRepo(db)

	mock.ExpectExec(`UPDATE approval_request`).
		WithArgs("a-1", model.ApprovalStatusRejected, "bad").
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := repo.Decide(context.Background(), "a-1", model.ApprovalStatusRejected, "bad"); err == nil {
		t.Fatal("expected error for already-decided approval, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

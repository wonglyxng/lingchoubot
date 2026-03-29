package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
)

type ApprovalRequestRepo struct {
	db *sql.DB
}

func NewApprovalRequestRepo(db *sql.DB) *ApprovalRequestRepo {
	return &ApprovalRequestRepo{db: db}
}

func (r *ApprovalRequestRepo) Create(ctx context.Context, a *model.ApprovalRequest) error {
	const q = `
		INSERT INTO approval_request (project_id, task_id, artifact_id, requested_by,
		                               approver_type, approver_id, title, description, status, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, q,
		a.ProjectID, a.TaskID, a.ArtifactID, a.RequestedBy,
		a.ApproverType, a.ApproverID, a.Title, a.Description, a.Status, a.Metadata,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (r *ApprovalRequestRepo) GetByID(ctx context.Context, id string) (*model.ApprovalRequest, error) {
	const q = `
		SELECT id, project_id, task_id, artifact_id, requested_by,
		       approver_type, approver_id, title, description, status,
		       decision_note, decided_at, metadata, created_at, updated_at
		FROM approval_request WHERE id = $1`
	a := &model.ApprovalRequest{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&a.ID, &a.ProjectID, &a.TaskID, &a.ArtifactID, &a.RequestedBy,
		&a.ApproverType, &a.ApproverID, &a.Title, &a.Description, &a.Status,
		&a.DecisionNote, &a.DecidedAt, &a.Metadata, &a.CreatedAt, &a.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("approvalRequest.GetByID: %w", err)
	}
	return a, nil
}

type ApprovalListParams struct {
	ProjectID string
	TaskID    string
	Status    string
	Limit     int
	Offset    int
}

func (r *ApprovalRequestRepo) List(ctx context.Context, p ApprovalListParams) ([]*model.ApprovalRequest, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if p.ProjectID != "" {
		where += fmt.Sprintf(" AND project_id = $%d", idx)
		args = append(args, p.ProjectID)
		idx++
	}
	if p.TaskID != "" {
		where += fmt.Sprintf(" AND task_id = $%d", idx)
		args = append(args, p.TaskID)
		idx++
	}
	if p.Status != "" {
		where += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, p.Status)
		idx++
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT count(*) FROM approval_request "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("approvalRequest.List count: %w", err)
	}

	q := fmt.Sprintf(`
		SELECT id, project_id, task_id, artifact_id, requested_by,
		       approver_type, approver_id, title, description, status,
		       decision_note, decided_at, metadata, created_at, updated_at
		FROM approval_request %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, idx, idx+1)
	args = append(args, p.Limit, p.Offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("approvalRequest.List: %w", err)
	}
	defer rows.Close()

	var list []*model.ApprovalRequest
	for rows.Next() {
		a := &model.ApprovalRequest{}
		if err := rows.Scan(
			&a.ID, &a.ProjectID, &a.TaskID, &a.ArtifactID, &a.RequestedBy,
			&a.ApproverType, &a.ApproverID, &a.Title, &a.Description, &a.Status,
			&a.DecisionNote, &a.DecidedAt, &a.Metadata, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("approvalRequest.List scan: %w", err)
		}
		list = append(list, a)
	}
	return list, total, rows.Err()
}

func (r *ApprovalRequestRepo) Decide(ctx context.Context, id string, status model.ApprovalStatus, note string) error {
	const q = `
		UPDATE approval_request
		SET status = $2, decision_note = $3, decided_at = now(), updated_at = now()
		WHERE id = $1 AND status = 'pending'`
	res, err := r.db.ExecContext(ctx, q, id, status, note)
	if err != nil {
		return fmt.Errorf("approvalRequest.Decide: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("approval not found or already decided")
	}
	return nil
}

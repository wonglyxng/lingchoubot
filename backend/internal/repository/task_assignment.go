package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
)

type TaskAssignmentRepo struct {
	db *sql.DB
}

func NewTaskAssignmentRepo(db *sql.DB) *TaskAssignmentRepo {
	return &TaskAssignmentRepo{db: db}
}

func (r *TaskAssignmentRepo) Create(ctx context.Context, a *model.TaskAssignment) error {
	const q = `
		INSERT INTO task_assignment (task_id, agent_id, assigned_by, role, status, note, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, q,
		a.TaskID, a.AgentID, a.AssignedBy, a.Role, a.Status, a.Note, a.Metadata,
	).Scan(&a.ID, &a.CreatedAt)
}

func (r *TaskAssignmentRepo) GetByID(ctx context.Context, id string) (*model.TaskAssignment, error) {
	const q = `
		SELECT id, task_id, agent_id, assigned_by, role, status, note, metadata,
		       created_at, completed_at
		FROM task_assignment WHERE id = $1`
	a := &model.TaskAssignment{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&a.ID, &a.TaskID, &a.AgentID, &a.AssignedBy, &a.Role, &a.Status,
		&a.Note, &a.Metadata, &a.CreatedAt, &a.CompletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("task_assignment.GetByID: %w", err)
	}
	return a, nil
}

type AssignmentListParams struct {
	TaskID  string
	AgentID string
	Status  string
	Limit   int
	Offset  int
}

func (r *TaskAssignmentRepo) List(ctx context.Context, p AssignmentListParams) ([]*model.TaskAssignment, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if p.TaskID != "" {
		where += fmt.Sprintf(" AND task_id = $%d", idx)
		args = append(args, p.TaskID)
		idx++
	}
	if p.AgentID != "" {
		where += fmt.Sprintf(" AND agent_id = $%d", idx)
		args = append(args, p.AgentID)
		idx++
	}
	if p.Status != "" {
		where += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, p.Status)
		idx++
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT count(*) FROM task_assignment "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("task_assignment.List count: %w", err)
	}

	q := fmt.Sprintf(`
		SELECT id, task_id, agent_id, assigned_by, role, status, note, metadata,
		       created_at, completed_at
		FROM task_assignment %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, idx, idx+1)
	args = append(args, p.Limit, p.Offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("task_assignment.List: %w", err)
	}
	defer rows.Close()

	var list []*model.TaskAssignment
	for rows.Next() {
		a := &model.TaskAssignment{}
		if err := rows.Scan(
			&a.ID, &a.TaskID, &a.AgentID, &a.AssignedBy, &a.Role, &a.Status,
			&a.Note, &a.Metadata, &a.CreatedAt, &a.CompletedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("task_assignment.List scan: %w", err)
		}
		list = append(list, a)
	}
	return list, total, rows.Err()
}

func (r *TaskAssignmentRepo) UpdateStatus(ctx context.Context, id string, status model.AssignmentStatus) error {
	q := `UPDATE task_assignment SET status = $2`
	if status == model.AssignmentStatusCompleted {
		q += `, completed_at = now()`
	}
	q += ` WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q, id, status)
	if err != nil {
		return fmt.Errorf("task_assignment.UpdateStatus: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("task_assignment.UpdateStatus: not found")
	}
	return nil
}

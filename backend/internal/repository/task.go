package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
)

type TaskRepo struct {
	db *sql.DB
}

func NewTaskRepo(db *sql.DB) *TaskRepo {
	return &TaskRepo{db: db}
}

func (r *TaskRepo) Create(ctx context.Context, t *model.Task) error {
	const q = `
		INSERT INTO task (project_id, phase_id, parent_task_id, title, description,
		                   status, priority, assignee_id, input_context, output_summary, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, q,
		t.ProjectID, t.PhaseID, t.ParentTaskID, t.Title, t.Description,
		t.Status, t.Priority, t.AssigneeID, t.InputContext, t.OutputSummary, t.Metadata,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *TaskRepo) GetByID(ctx context.Context, id string) (*model.Task, error) {
	const q = `
		SELECT id, project_id, phase_id, parent_task_id, title, description,
		       status, priority, assignee_id, input_context, output_summary, metadata,
		       created_at, updated_at
		FROM task WHERE id = $1`
	t := &model.Task{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&t.ID, &t.ProjectID, &t.PhaseID, &t.ParentTaskID, &t.Title, &t.Description,
		&t.Status, &t.Priority, &t.AssigneeID, &t.InputContext, &t.OutputSummary, &t.Metadata,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("task.GetByID: %w", err)
	}
	return t, nil
}

type TaskListParams struct {
	ProjectID string
	PhaseID   string
	Status    string
	Limit     int
	Offset    int
}

func (r *TaskRepo) List(ctx context.Context, p TaskListParams) ([]*model.Task, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if p.ProjectID != "" {
		where += fmt.Sprintf(" AND project_id = $%d", idx)
		args = append(args, p.ProjectID)
		idx++
	}
	if p.PhaseID != "" {
		where += fmt.Sprintf(" AND phase_id = $%d", idx)
		args = append(args, p.PhaseID)
		idx++
	}
	if p.Status != "" {
		where += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, p.Status)
		idx++
	}

	var total int
	countQ := "SELECT count(*) FROM task " + where
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("task.List count: %w", err)
	}

	q := fmt.Sprintf(`
		SELECT id, project_id, phase_id, parent_task_id, title, description,
		       status, priority, assignee_id, input_context, output_summary, metadata,
		       created_at, updated_at
		FROM task %s ORDER BY priority DESC, created_at DESC LIMIT $%d OFFSET $%d`,
		where, idx, idx+1)
	args = append(args, p.Limit, p.Offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("task.List: %w", err)
	}
	defer rows.Close()

	var list []*model.Task
	for rows.Next() {
		t := &model.Task{}
		if err := rows.Scan(
			&t.ID, &t.ProjectID, &t.PhaseID, &t.ParentTaskID, &t.Title, &t.Description,
			&t.Status, &t.Priority, &t.AssigneeID, &t.InputContext, &t.OutputSummary, &t.Metadata,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("task.List scan: %w", err)
		}
		list = append(list, t)
	}
	return list, total, rows.Err()
}

func (r *TaskRepo) UpdateStatus(ctx context.Context, id string, status model.TaskStatus) error {
	const q = `UPDATE task SET status = $2, updated_at = now() WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q, id, status)
	if err != nil {
		return fmt.Errorf("task.UpdateStatus: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("task.UpdateStatus: not found")
	}
	return nil
}

func (r *TaskRepo) Update(ctx context.Context, t *model.Task) error {
	const q = `
		UPDATE task
		SET title = $2, description = $3, status = $4, priority = $5,
		    phase_id = $6, parent_task_id = $7, assignee_id = $8,
		    input_context = $9, output_summary = $10, metadata = $11, updated_at = now()
		WHERE id = $1
		RETURNING updated_at`
	return r.db.QueryRowContext(ctx, q,
		t.ID, t.Title, t.Description, t.Status, t.Priority,
		t.PhaseID, t.ParentTaskID, t.AssigneeID,
		t.InputContext, t.OutputSummary, t.Metadata,
	).Scan(&t.UpdatedAt)
}

func (r *TaskRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM task WHERE id = $1`, id)
	return err
}

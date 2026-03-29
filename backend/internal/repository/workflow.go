package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
)

type WorkflowRunRepo struct {
	db *sql.DB
}

func NewWorkflowRunRepo(db *sql.DB) *WorkflowRunRepo {
	return &WorkflowRunRepo{db: db}
}

func (r *WorkflowRunRepo) Create(ctx context.Context, run *model.WorkflowRun) error {
	const q = `
		INSERT INTO workflow_run (project_id, status, summary, error, started_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, q,
		run.ProjectID, run.Status, run.Summary, run.Error, run.StartedAt,
	).Scan(&run.ID, &run.CreatedAt, &run.UpdatedAt)
}

func (r *WorkflowRunRepo) GetByID(ctx context.Context, id string) (*model.WorkflowRun, error) {
	const q = `
		SELECT id, project_id, status, summary, error,
		       started_at, completed_at, created_at, updated_at
		FROM workflow_run WHERE id = $1`
	run := &model.WorkflowRun{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&run.ID, &run.ProjectID, &run.Status, &run.Summary, &run.Error,
		&run.StartedAt, &run.CompletedAt, &run.CreatedAt, &run.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("workflow_run.GetByID: %w", err)
	}
	return run, nil
}

// UpdateStatus updates the run's status, summary, error and completed_at.
func (r *WorkflowRunRepo) UpdateStatus(ctx context.Context, run *model.WorkflowRun) error {
	const q = `
		UPDATE workflow_run
		SET status = $2, summary = $3, error = $4, completed_at = $5, updated_at = now()
		WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q,
		run.ID, run.Status, run.Summary, run.Error, run.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("workflow_run.UpdateStatus: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("workflow_run %s not found", run.ID)
	}
	return nil
}

type WorkflowRunListParams struct {
	ProjectID string
	Status    string
	Limit     int
	Offset    int
}

func (r *WorkflowRunRepo) List(ctx context.Context, p WorkflowRunListParams) ([]*model.WorkflowRun, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if p.ProjectID != "" {
		where += fmt.Sprintf(" AND project_id = $%d", idx)
		args = append(args, p.ProjectID)
		idx++
	}
	if p.Status != "" {
		where += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, p.Status)
		idx++
	}

	var total int
	countQ := "SELECT count(*) FROM workflow_run " + where
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("workflow_run.List count: %w", err)
	}

	q := fmt.Sprintf(`
		SELECT id, project_id, status, summary, error,
		       started_at, completed_at, created_at, updated_at
		FROM workflow_run %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, idx, idx+1)
	args = append(args, p.Limit, p.Offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("workflow_run.List: %w", err)
	}
	defer rows.Close()

	var list []*model.WorkflowRun
	for rows.Next() {
		run := &model.WorkflowRun{}
		if err := rows.Scan(
			&run.ID, &run.ProjectID, &run.Status, &run.Summary, &run.Error,
			&run.StartedAt, &run.CompletedAt, &run.CreatedAt, &run.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("workflow_run.List scan: %w", err)
		}
		list = append(list, run)
	}
	return list, total, rows.Err()
}

// --- WorkflowStepRepo ---

type WorkflowStepRepo struct {
	db *sql.DB
}

func NewWorkflowStepRepo(db *sql.DB) *WorkflowStepRepo {
	return &WorkflowStepRepo{db: db}
}

func (r *WorkflowStepRepo) Create(ctx context.Context, s *model.WorkflowStep) error {
	const q = `
		INSERT INTO workflow_step (run_id, name, agent_role, agent_id, task_id, phase_id,
		                           status, summary, error, sort_order, started_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, q,
		s.RunID, s.Name, s.AgentRole, s.AgentID, s.TaskID, s.PhaseID,
		s.Status, s.Summary, s.Error, s.SortOrder, s.StartedAt, s.CompletedAt,
	).Scan(&s.ID, &s.CreatedAt)
}

// UpdateStatus updates the step's mutable fields.
func (r *WorkflowStepRepo) UpdateStatus(ctx context.Context, s *model.WorkflowStep) error {
	const q = `
		UPDATE workflow_step
		SET status = $2, summary = $3, error = $4, agent_id = $5,
		    task_id = $6, phase_id = $7, started_at = $8, completed_at = $9
		WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q,
		s.ID, s.Status, s.Summary, s.Error, s.AgentID,
		s.TaskID, s.PhaseID, s.StartedAt, s.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("workflow_step.UpdateStatus: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("workflow_step %s not found", s.ID)
	}
	return nil
}

// ListByRunID returns all steps for a given run, ordered by sort_order.
func (r *WorkflowStepRepo) ListByRunID(ctx context.Context, runID string) ([]*model.WorkflowStep, error) {
	const q = `
		SELECT id, run_id, name, agent_role, agent_id, task_id, phase_id,
		       status, summary, error, sort_order, started_at, completed_at, created_at
		FROM workflow_step WHERE run_id = $1 ORDER BY sort_order, created_at`
	rows, err := r.db.QueryContext(ctx, q, runID)
	if err != nil {
		return nil, fmt.Errorf("workflow_step.ListByRunID: %w", err)
	}
	defer rows.Close()

	var list []*model.WorkflowStep
	for rows.Next() {
		s := &model.WorkflowStep{}
		if err := rows.Scan(
			&s.ID, &s.RunID, &s.Name, &s.AgentRole, &s.AgentID, &s.TaskID, &s.PhaseID,
			&s.Status, &s.Summary, &s.Error, &s.SortOrder, &s.StartedAt, &s.CompletedAt, &s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("workflow_step.ListByRunID scan: %w", err)
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

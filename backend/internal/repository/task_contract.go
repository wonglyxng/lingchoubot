package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
)

type TaskContractRepo struct {
	db *sql.DB
}

func NewTaskContractRepo(db *sql.DB) *TaskContractRepo {
	return &TaskContractRepo{db: db}
}

func (r *TaskContractRepo) Create(ctx context.Context, c *model.TaskContract) error {
	const q = `
		INSERT INTO task_contract (task_id, version, scope, non_goals, done_definition,
		                           verification_plan, acceptance_criteria, tool_permissions,
		                           escalation_policy, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, q,
		c.TaskID, c.Version, c.Scope, c.NonGoals, c.DoneDefinition,
		c.VerificationPlan, c.AcceptanceCriteria, c.ToolPermissions,
		c.EscalationPolicy, c.Metadata,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *TaskContractRepo) GetByID(ctx context.Context, id string) (*model.TaskContract, error) {
	const q = `
		SELECT id, task_id, version, scope, non_goals, done_definition,
		       verification_plan, acceptance_criteria, tool_permissions,
		       escalation_policy, metadata, created_at, updated_at
		FROM task_contract WHERE id = $1`
	c := &model.TaskContract{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&c.ID, &c.TaskID, &c.Version, &c.Scope, &c.NonGoals, &c.DoneDefinition,
		&c.VerificationPlan, &c.AcceptanceCriteria, &c.ToolPermissions,
		&c.EscalationPolicy, &c.Metadata, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("task_contract.GetByID: %w", err)
	}
	return c, nil
}

// GetLatestByTaskID returns the highest-version contract for a task.
func (r *TaskContractRepo) GetLatestByTaskID(ctx context.Context, taskID string) (*model.TaskContract, error) {
	const q = `
		SELECT id, task_id, version, scope, non_goals, done_definition,
		       verification_plan, acceptance_criteria, tool_permissions,
		       escalation_policy, metadata, created_at, updated_at
		FROM task_contract WHERE task_id = $1
		ORDER BY version DESC LIMIT 1`
	c := &model.TaskContract{}
	err := r.db.QueryRowContext(ctx, q, taskID).Scan(
		&c.ID, &c.TaskID, &c.Version, &c.Scope, &c.NonGoals, &c.DoneDefinition,
		&c.VerificationPlan, &c.AcceptanceCriteria, &c.ToolPermissions,
		&c.EscalationPolicy, &c.Metadata, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("task_contract.GetLatestByTaskID: %w", err)
	}
	return c, nil
}

func (r *TaskContractRepo) ListByTaskID(ctx context.Context, taskID string) ([]*model.TaskContract, error) {
	const q = `
		SELECT id, task_id, version, scope, non_goals, done_definition,
		       verification_plan, acceptance_criteria, tool_permissions,
		       escalation_policy, metadata, created_at, updated_at
		FROM task_contract WHERE task_id = $1
		ORDER BY version ASC`
	rows, err := r.db.QueryContext(ctx, q, taskID)
	if err != nil {
		return nil, fmt.Errorf("task_contract.ListByTaskID: %w", err)
	}
	defer rows.Close()

	var list []*model.TaskContract
	for rows.Next() {
		c := &model.TaskContract{}
		if err := rows.Scan(
			&c.ID, &c.TaskID, &c.Version, &c.Scope, &c.NonGoals, &c.DoneDefinition,
			&c.VerificationPlan, &c.AcceptanceCriteria, &c.ToolPermissions,
			&c.EscalationPolicy, &c.Metadata, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("task_contract.ListByTaskID scan: %w", err)
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// NextVersion returns the next available version number for a task's contract.
func (r *TaskContractRepo) NextVersion(ctx context.Context, taskID string) (int, error) {
	var maxVer sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT max(version) FROM task_contract WHERE task_id = $1`, taskID,
	).Scan(&maxVer)
	if err != nil {
		return 0, fmt.Errorf("task_contract.NextVersion: %w", err)
	}
	if !maxVer.Valid {
		return 1, nil
	}
	return int(maxVer.Int64) + 1, nil
}

func (r *TaskContractRepo) Update(ctx context.Context, c *model.TaskContract) error {
	const q = `
		UPDATE task_contract
		SET scope = $2, non_goals = $3, done_definition = $4, verification_plan = $5,
		    acceptance_criteria = $6, tool_permissions = $7, escalation_policy = $8,
		    metadata = $9, updated_at = now()
		WHERE id = $1
		RETURNING updated_at`
	return r.db.QueryRowContext(ctx, q,
		c.ID, c.Scope, c.NonGoals, c.DoneDefinition, c.VerificationPlan,
		c.AcceptanceCriteria, c.ToolPermissions, c.EscalationPolicy, c.Metadata,
	).Scan(&c.UpdatedAt)
}

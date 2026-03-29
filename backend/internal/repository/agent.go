package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
)

type AgentRepo struct {
	db *sql.DB
}

func NewAgentRepo(db *sql.DB) *AgentRepo {
	return &AgentRepo{db: db}
}

func (r *AgentRepo) Create(ctx context.Context, a *model.Agent) error {
	const q = `
		INSERT INTO agent (name, role, description, reports_to, status, capabilities, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, q,
		a.Name, a.Role, a.Description, a.ReportsTo, a.Status, a.Capabilities, a.Metadata,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (r *AgentRepo) GetByID(ctx context.Context, id string) (*model.Agent, error) {
	const q = `
		SELECT id, name, role, description, reports_to, status, capabilities, metadata, created_at, updated_at
		FROM agent WHERE id = $1`
	a := &model.Agent{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&a.ID, &a.Name, &a.Role, &a.Description, &a.ReportsTo,
		&a.Status, &a.Capabilities, &a.Metadata, &a.CreatedAt, &a.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("agent.GetByID: %w", err)
	}
	return a, nil
}

func (r *AgentRepo) List(ctx context.Context, limit, offset int) ([]*model.Agent, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM agent`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("agent.List count: %w", err)
	}

	const q = `
		SELECT id, name, role, description, reports_to, status, capabilities, metadata, created_at, updated_at
		FROM agent ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("agent.List: %w", err)
	}
	defer rows.Close()

	var list []*model.Agent
	for rows.Next() {
		a := &model.Agent{}
		if err := rows.Scan(
			&a.ID, &a.Name, &a.Role, &a.Description, &a.ReportsTo,
			&a.Status, &a.Capabilities, &a.Metadata, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("agent.List scan: %w", err)
		}
		list = append(list, a)
	}
	return list, total, rows.Err()
}

func (r *AgentRepo) Update(ctx context.Context, a *model.Agent) error {
	const q = `
		UPDATE agent
		SET name = $2, role = $3, description = $4, reports_to = $5, status = $6,
		    capabilities = $7, metadata = $8, updated_at = now()
		WHERE id = $1
		RETURNING updated_at`
	return r.db.QueryRowContext(ctx, q,
		a.ID, a.Name, a.Role, a.Description, a.ReportsTo,
		a.Status, a.Capabilities, a.Metadata,
	).Scan(&a.UpdatedAt)
}

func (r *AgentRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM agent WHERE id = $1`, id)
	return err
}

// GetSubordinates returns all agents that directly report to the given agent.
func (r *AgentRepo) GetSubordinates(ctx context.Context, agentID string) ([]*model.Agent, error) {
	const q = `
		SELECT id, name, role, description, reports_to, status, capabilities, metadata, created_at, updated_at
		FROM agent WHERE reports_to = $1 ORDER BY created_at`
	rows, err := r.db.QueryContext(ctx, q, agentID)
	if err != nil {
		return nil, fmt.Errorf("agent.GetSubordinates: %w", err)
	}
	defer rows.Close()

	var list []*model.Agent
	for rows.Next() {
		a := &model.Agent{}
		if err := rows.Scan(
			&a.ID, &a.Name, &a.Role, &a.Description, &a.ReportsTo,
			&a.Status, &a.Capabilities, &a.Metadata, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("agent.GetSubordinates scan: %w", err)
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// GetOrgTree returns a flat list of all agents under the given root (recursive via CTE).
// If rootID is empty, returns the full org tree.
func (r *AgentRepo) GetOrgTree(ctx context.Context, rootID string) ([]*model.Agent, error) {
	var q string
	var args []interface{}

	if rootID != "" {
		q = `
			WITH RECURSIVE tree AS (
				SELECT id, name, role, description, reports_to, status, capabilities, metadata, created_at, updated_at, 0 AS depth
				FROM agent WHERE id = $1
				UNION ALL
				SELECT a.id, a.name, a.role, a.description, a.reports_to, a.status, a.capabilities, a.metadata, a.created_at, a.updated_at, t.depth + 1
				FROM agent a INNER JOIN tree t ON a.reports_to = t.id
			)
			SELECT id, name, role, description, reports_to, status, capabilities, metadata, created_at, updated_at
			FROM tree ORDER BY depth, created_at`
		args = []interface{}{rootID}
	} else {
		q = `
			WITH RECURSIVE tree AS (
				SELECT id, name, role, description, reports_to, status, capabilities, metadata, created_at, updated_at, 0 AS depth
				FROM agent WHERE reports_to IS NULL
				UNION ALL
				SELECT a.id, a.name, a.role, a.description, a.reports_to, a.status, a.capabilities, a.metadata, a.created_at, a.updated_at, t.depth + 1
				FROM agent a INNER JOIN tree t ON a.reports_to = t.id
			)
			SELECT id, name, role, description, reports_to, status, capabilities, metadata, created_at, updated_at
			FROM tree ORDER BY depth, created_at`
	}

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("agent.GetOrgTree: %w", err)
	}
	defer rows.Close()

	var list []*model.Agent
	for rows.Next() {
		a := &model.Agent{}
		if err := rows.Scan(
			&a.ID, &a.Name, &a.Role, &a.Description, &a.ReportsTo,
			&a.Status, &a.Capabilities, &a.Metadata, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("agent.GetOrgTree scan: %w", err)
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

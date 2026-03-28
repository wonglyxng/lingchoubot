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

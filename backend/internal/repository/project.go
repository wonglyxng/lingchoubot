package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
)

type ProjectRepo struct {
	db *sql.DB
}

func NewProjectRepo(db *sql.DB) *ProjectRepo {
	return &ProjectRepo{db: db}
}

func (r *ProjectRepo) Create(ctx context.Context, p *model.Project) error {
	const q = `
		INSERT INTO project (name, description, status, owner_agent_id, metadata)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, q,
		p.Name, p.Description, p.Status, p.OwnerAgentID, p.Metadata,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *ProjectRepo) GetByID(ctx context.Context, id string) (*model.Project, error) {
	const q = `
		SELECT id, name, description, status, owner_agent_id, metadata, created_at, updated_at
		FROM project WHERE id = $1`
	p := &model.Project{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.Status, &p.OwnerAgentID,
		&p.Metadata, &p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("project.GetByID: %w", err)
	}
	return p, nil
}

func (r *ProjectRepo) List(ctx context.Context, limit, offset int) ([]*model.Project, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM project`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("project.List count: %w", err)
	}

	const q = `
		SELECT id, name, description, status, owner_agent_id, metadata, created_at, updated_at
		FROM project ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("project.List: %w", err)
	}
	defer rows.Close()

	var list []*model.Project
	for rows.Next() {
		p := &model.Project{}
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Status, &p.OwnerAgentID,
			&p.Metadata, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("project.List scan: %w", err)
		}
		list = append(list, p)
	}
	return list, total, rows.Err()
}

func (r *ProjectRepo) Update(ctx context.Context, p *model.Project) error {
	const q = `
		UPDATE project
		SET name = $2, description = $3, status = $4, owner_agent_id = $5, metadata = $6, updated_at = now()
		WHERE id = $1
		RETURNING updated_at`
	return r.db.QueryRowContext(ctx, q,
		p.ID, p.Name, p.Description, p.Status, p.OwnerAgentID, p.Metadata,
	).Scan(&p.UpdatedAt)
}

func (r *ProjectRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM project WHERE id = $1`, id)
	return err
}

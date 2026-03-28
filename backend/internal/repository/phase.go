package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
)

type PhaseRepo struct {
	db *sql.DB
}

func NewPhaseRepo(db *sql.DB) *PhaseRepo {
	return &PhaseRepo{db: db}
}

func (r *PhaseRepo) Create(ctx context.Context, p *model.Phase) error {
	const q = `
		INSERT INTO project_phase (project_id, name, description, status, sort_order, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, q,
		p.ProjectID, p.Name, p.Description, p.Status, p.SortOrder, p.Metadata,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *PhaseRepo) GetByID(ctx context.Context, id string) (*model.Phase, error) {
	const q = `
		SELECT id, project_id, name, description, status, sort_order, metadata, created_at, updated_at
		FROM project_phase WHERE id = $1`
	p := &model.Phase{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&p.ID, &p.ProjectID, &p.Name, &p.Description, &p.Status,
		&p.SortOrder, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("phase.GetByID: %w", err)
	}
	return p, nil
}

func (r *PhaseRepo) ListByProject(ctx context.Context, projectID string) ([]*model.Phase, error) {
	const q = `
		SELECT id, project_id, name, description, status, sort_order, metadata, created_at, updated_at
		FROM project_phase WHERE project_id = $1 ORDER BY sort_order, created_at`
	rows, err := r.db.QueryContext(ctx, q, projectID)
	if err != nil {
		return nil, fmt.Errorf("phase.ListByProject: %w", err)
	}
	defer rows.Close()

	var list []*model.Phase
	for rows.Next() {
		p := &model.Phase{}
		if err := rows.Scan(
			&p.ID, &p.ProjectID, &p.Name, &p.Description, &p.Status,
			&p.SortOrder, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("phase.ListByProject scan: %w", err)
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *PhaseRepo) Update(ctx context.Context, p *model.Phase) error {
	const q = `
		UPDATE project_phase
		SET name = $2, description = $3, status = $4, sort_order = $5, metadata = $6, updated_at = now()
		WHERE id = $1
		RETURNING updated_at`
	return r.db.QueryRowContext(ctx, q,
		p.ID, p.Name, p.Description, p.Status, p.SortOrder, p.Metadata,
	).Scan(&p.UpdatedAt)
}

func (r *PhaseRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM project_phase WHERE id = $1`, id)
	return err
}

package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
)

type HandoffSnapshotRepo struct {
	db *sql.DB
}

func NewHandoffSnapshotRepo(db *sql.DB) *HandoffSnapshotRepo {
	return &HandoffSnapshotRepo{db: db}
}

func (r *HandoffSnapshotRepo) Create(ctx context.Context, s *model.HandoffSnapshot) error {
	const q = `
		INSERT INTO handoff_snapshot (task_id, agent_id, summary, completed_items,
		                               pending_items, risks, next_steps, artifact_refs, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, q,
		s.TaskID, s.AgentID, s.Summary, s.CompletedItems,
		s.PendingItems, s.Risks, s.NextSteps, s.ArtifactRefs, s.Metadata,
	).Scan(&s.ID, &s.CreatedAt)
}

func (r *HandoffSnapshotRepo) GetByID(ctx context.Context, id string) (*model.HandoffSnapshot, error) {
	const q = `
		SELECT id, task_id, agent_id, summary, completed_items, pending_items,
		       risks, next_steps, artifact_refs, metadata, created_at
		FROM handoff_snapshot WHERE id = $1`
	s := &model.HandoffSnapshot{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&s.ID, &s.TaskID, &s.AgentID, &s.Summary, &s.CompletedItems, &s.PendingItems,
		&s.Risks, &s.NextSteps, &s.ArtifactRefs, &s.Metadata, &s.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("handoff_snapshot.GetByID: %w", err)
	}
	return s, nil
}

type HandoffListParams struct {
	TaskID  string
	AgentID string
	Limit   int
	Offset  int
}

func (r *HandoffSnapshotRepo) List(ctx context.Context, p HandoffListParams) ([]*model.HandoffSnapshot, int, error) {
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

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT count(*) FROM handoff_snapshot "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("handoff_snapshot.List count: %w", err)
	}

	q := fmt.Sprintf(`
		SELECT id, task_id, agent_id, summary, completed_items, pending_items,
		       risks, next_steps, artifact_refs, metadata, created_at
		FROM handoff_snapshot %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, idx, idx+1)
	args = append(args, p.Limit, p.Offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("handoff_snapshot.List: %w", err)
	}
	defer rows.Close()

	var list []*model.HandoffSnapshot
	for rows.Next() {
		s := &model.HandoffSnapshot{}
		if err := rows.Scan(
			&s.ID, &s.TaskID, &s.AgentID, &s.Summary, &s.CompletedItems, &s.PendingItems,
			&s.Risks, &s.NextSteps, &s.ArtifactRefs, &s.Metadata, &s.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("handoff_snapshot.List scan: %w", err)
		}
		list = append(list, s)
	}
	return list, total, rows.Err()
}

// GetLatestByTaskID returns the most recent snapshot for a given task.
func (r *HandoffSnapshotRepo) GetLatestByTaskID(ctx context.Context, taskID string) (*model.HandoffSnapshot, error) {
	const q = `
		SELECT id, task_id, agent_id, summary, completed_items, pending_items,
		       risks, next_steps, artifact_refs, metadata, created_at
		FROM handoff_snapshot WHERE task_id = $1
		ORDER BY created_at DESC LIMIT 1`
	s := &model.HandoffSnapshot{}
	err := r.db.QueryRowContext(ctx, q, taskID).Scan(
		&s.ID, &s.TaskID, &s.AgentID, &s.Summary, &s.CompletedItems, &s.PendingItems,
		&s.Risks, &s.NextSteps, &s.ArtifactRefs, &s.Metadata, &s.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("handoff_snapshot.GetLatestByTaskID: %w", err)
	}
	return s, nil
}

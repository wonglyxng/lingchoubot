package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
)

type AuditRepo struct {
	db *sql.DB
}

func NewAuditRepo(db *sql.DB) *AuditRepo {
	return &AuditRepo{db: db}
}

func (r *AuditRepo) Create(ctx context.Context, a *model.AuditLog) error {
	const q = `
		INSERT INTO audit_log (actor_type, actor_id, event_type, event_summary,
		                        target_type, target_id, before_state, after_state, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, q,
		a.ActorType, a.ActorID, a.EventType, a.EventSummary,
		a.TargetType, a.TargetID, a.BeforeState, a.AfterState, a.Metadata,
	).Scan(&a.ID, &a.CreatedAt)
}

type AuditListParams struct {
	TargetType string
	TargetID   string
	EventType  string
	Limit      int
	Offset     int
}

func (r *AuditRepo) List(ctx context.Context, p AuditListParams) ([]*model.AuditLog, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if p.TargetType != "" {
		where += fmt.Sprintf(" AND target_type = $%d", idx)
		args = append(args, p.TargetType)
		idx++
	}
	if p.TargetID != "" {
		where += fmt.Sprintf(" AND target_id = $%d", idx)
		args = append(args, p.TargetID)
		idx++
	}
	if p.EventType != "" {
		where += fmt.Sprintf(" AND event_type = $%d", idx)
		args = append(args, p.EventType)
		idx++
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT count(*) FROM audit_log "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("audit.List count: %w", err)
	}

	q := fmt.Sprintf(`
		SELECT id, actor_type, actor_id, event_type, event_summary,
		       target_type, target_id, before_state, after_state, metadata, created_at
		FROM audit_log %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, idx, idx+1)
	args = append(args, p.Limit, p.Offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("audit.List: %w", err)
	}
	defer rows.Close()

	var list []*model.AuditLog
	for rows.Next() {
		a := &model.AuditLog{}
		if err := rows.Scan(
			&a.ID, &a.ActorType, &a.ActorID, &a.EventType, &a.EventSummary,
			&a.TargetType, &a.TargetID, &a.BeforeState, &a.AfterState, &a.Metadata, &a.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("audit.List scan: %w", err)
		}
		list = append(list, a)
	}
	return list, total, rows.Err()
}

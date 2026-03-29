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

// ProjectTimeline returns all audit events related to a project and its
// child entities (tasks, artifacts, contracts, assignments, reviews, approvals, etc.).
func (r *AuditRepo) ProjectTimeline(ctx context.Context, projectID string, limit, offset int) ([]*model.AuditLog, int, error) {
	cte := `
	WITH related_ids AS (
		SELECT 'project' AS t, id::text AS tid FROM project WHERE id = $1
		UNION ALL
		SELECT 'project_phase', id::text FROM project_phase WHERE project_id = $1
		UNION ALL
		SELECT 'task', id::text FROM task WHERE project_id = $1
		UNION ALL
		SELECT 'task_contract', tc.id::text FROM task_contract tc
		  JOIN task t ON tc.task_id = t.id WHERE t.project_id = $1
		UNION ALL
		SELECT 'task_assignment', ta.id::text FROM task_assignment ta
		  JOIN task t ON ta.task_id = t.id WHERE t.project_id = $1
		UNION ALL
		SELECT 'handoff_snapshot', hs.id::text FROM handoff_snapshot hs
		  JOIN task t ON hs.task_id = t.id WHERE t.project_id = $1
		UNION ALL
		SELECT 'artifact', id::text FROM artifact WHERE project_id = $1
		UNION ALL
		SELECT 'artifact_version', av.id::text FROM artifact_version av
		  JOIN artifact a ON av.artifact_id = a.id WHERE a.project_id = $1
		UNION ALL
		SELECT 'review_report', rr.id::text FROM review_report rr
		  JOIN task t ON rr.task_id = t.id WHERE t.project_id = $1
		UNION ALL
		SELECT 'approval_request', id::text FROM approval_request WHERE project_id = $1
	)`

	countQ := cte + `
	SELECT count(*) FROM audit_log al
	JOIN related_ids ri ON al.target_type = ri.t AND al.target_id = ri.tid`

	var total int
	if err := r.db.QueryRowContext(ctx, countQ, projectID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("audit.ProjectTimeline count: %w", err)
	}

	q := cte + `
	SELECT al.id, al.actor_type, al.actor_id, al.event_type, al.event_summary,
	       al.target_type, al.target_id, al.before_state, al.after_state, al.metadata, al.created_at
	FROM audit_log al
	JOIN related_ids ri ON al.target_type = ri.t AND al.target_id = ri.tid
	ORDER BY al.created_at DESC
	LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, q, projectID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("audit.ProjectTimeline: %w", err)
	}
	defer rows.Close()

	var list []*model.AuditLog
	for rows.Next() {
		a := &model.AuditLog{}
		if err := rows.Scan(
			&a.ID, &a.ActorType, &a.ActorID, &a.EventType, &a.EventSummary,
			&a.TargetType, &a.TargetID, &a.BeforeState, &a.AfterState, &a.Metadata, &a.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("audit.ProjectTimeline scan: %w", err)
		}
		list = append(list, a)
	}
	return list, total, rows.Err()
}

// TaskTimeline returns all audit events directly targeting a given task or
// its child entities (contracts, assignments, handoffs, reviews).
func (r *AuditRepo) TaskTimeline(ctx context.Context, taskID string, limit, offset int) ([]*model.AuditLog, int, error) {
	cte := `
	WITH related_ids AS (
		SELECT 'task' AS t, $1::text AS tid
		UNION ALL
		SELECT 'task_contract', id::text FROM task_contract WHERE task_id = $1::uuid
		UNION ALL
		SELECT 'task_assignment', id::text FROM task_assignment WHERE task_id = $1::uuid
		UNION ALL
		SELECT 'handoff_snapshot', id::text FROM handoff_snapshot WHERE task_id = $1::uuid
		UNION ALL
		SELECT 'review_report', id::text FROM review_report WHERE task_id = $1::uuid
		UNION ALL
		SELECT 'approval_request', id::text FROM approval_request WHERE task_id = $1::uuid
	)`

	countQ := cte + `
	SELECT count(*) FROM audit_log al
	JOIN related_ids ri ON al.target_type = ri.t AND al.target_id = ri.tid`

	var total int
	if err := r.db.QueryRowContext(ctx, countQ, taskID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("audit.TaskTimeline count: %w", err)
	}

	q := cte + `
	SELECT al.id, al.actor_type, al.actor_id, al.event_type, al.event_summary,
	       al.target_type, al.target_id, al.before_state, al.after_state, al.metadata, al.created_at
	FROM audit_log al
	JOIN related_ids ri ON al.target_type = ri.t AND al.target_id = ri.tid
	ORDER BY al.created_at DESC
	LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, q, taskID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("audit.TaskTimeline: %w", err)
	}
	defer rows.Close()

	var list []*model.AuditLog
	for rows.Next() {
		a := &model.AuditLog{}
		if err := rows.Scan(
			&a.ID, &a.ActorType, &a.ActorID, &a.EventType, &a.EventSummary,
			&a.TargetType, &a.TargetID, &a.BeforeState, &a.AfterState, &a.Metadata, &a.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("audit.TaskTimeline scan: %w", err)
		}
		list = append(list, a)
	}
	return list, total, rows.Err()
}

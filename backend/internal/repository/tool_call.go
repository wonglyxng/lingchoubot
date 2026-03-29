package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
)

type ToolCallRepo struct {
	db *sql.DB
}

func NewToolCallRepo(db *sql.DB) *ToolCallRepo {
	return &ToolCallRepo{db: db}
}

func (r *ToolCallRepo) Create(ctx context.Context, tc *model.ToolCall) error {
	const q = `
		INSERT INTO tool_call (task_id, agent_id, tool_name, input, output, status, error_message, duration_ms, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, q,
		tc.TaskID, tc.AgentID, tc.ToolName, tc.Input, tc.Output,
		tc.Status, tc.ErrorMessage, tc.DurationMs, tc.Metadata,
	).Scan(&tc.ID, &tc.CreatedAt)
}

func (r *ToolCallRepo) GetByID(ctx context.Context, id string) (*model.ToolCall, error) {
	const q = `
		SELECT id, task_id, agent_id, tool_name, input, output,
		       status, error_message, duration_ms, metadata, created_at, completed_at
		FROM tool_call WHERE id = $1`
	tc := &model.ToolCall{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&tc.ID, &tc.TaskID, &tc.AgentID, &tc.ToolName, &tc.Input, &tc.Output,
		&tc.Status, &tc.ErrorMessage, &tc.DurationMs, &tc.Metadata, &tc.CreatedAt, &tc.CompletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("tool_call.GetByID: %w", err)
	}
	return tc, nil
}

type ToolCallListParams struct {
	TaskID   string
	AgentID  string
	ToolName string
	Status   string
	Limit    int
	Offset   int
}

func (r *ToolCallRepo) List(ctx context.Context, p ToolCallListParams) ([]*model.ToolCall, int, error) {
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
	if p.ToolName != "" {
		where += fmt.Sprintf(" AND tool_name = $%d", idx)
		args = append(args, p.ToolName)
		idx++
	}
	if p.Status != "" {
		where += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, p.Status)
		idx++
	}

	var total int
	countQ := "SELECT count(*) FROM tool_call " + where
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("tool_call.List count: %w", err)
	}

	q := fmt.Sprintf(`
		SELECT id, task_id, agent_id, tool_name, input, output,
		       status, error_message, duration_ms, metadata, created_at, completed_at
		FROM tool_call %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, idx, idx+1)
	args = append(args, p.Limit, p.Offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("tool_call.List: %w", err)
	}
	defer rows.Close()

	var list []*model.ToolCall
	for rows.Next() {
		tc := &model.ToolCall{}
		if err := rows.Scan(
			&tc.ID, &tc.TaskID, &tc.AgentID, &tc.ToolName, &tc.Input, &tc.Output,
			&tc.Status, &tc.ErrorMessage, &tc.DurationMs, &tc.Metadata, &tc.CreatedAt, &tc.CompletedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("tool_call.List scan: %w", err)
		}
		list = append(list, tc)
	}
	return list, total, rows.Err()
}

// Complete updates a tool call with its execution result.
func (r *ToolCallRepo) Complete(ctx context.Context, id string, status model.ToolCallStatus, output model.JSON, errMsg string, durationMs int) error {
	const q = `
		UPDATE tool_call
		SET status = $2, output = $3, error_message = $4, duration_ms = $5, completed_at = now()
		WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q, id, status, output, errMsg, durationMs)
	if err != nil {
		return fmt.Errorf("tool_call.Complete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("tool_call.Complete: not found")
	}
	return nil
}

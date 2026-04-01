package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
)

type ReviewReportRepo struct {
	db *sql.DB
}

func NewReviewReportRepo(db *sql.DB) *ReviewReportRepo {
	return &ReviewReportRepo{db: db}
}

func (r *ReviewReportRepo) Create(ctx context.Context, rr *model.ReviewReport) error {
	const q = `
		INSERT INTO review_report (run_id, task_id, reviewer_id, artifact_version_id, verdict,
		                            summary, findings, recommendations, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, q,
		rr.RunID, rr.TaskID, rr.ReviewerID, rr.ArtifactVersionID, rr.Verdict,
		rr.Summary, rr.Findings, rr.Recommendations, rr.Metadata,
	).Scan(&rr.ID, &rr.CreatedAt)
}

func (r *ReviewReportRepo) GetByID(ctx context.Context, id string) (*model.ReviewReport, error) {
	const q = `
		SELECT id, run_id, task_id, reviewer_id, artifact_version_id, verdict,
		       summary, findings, recommendations, metadata, created_at
		FROM review_report WHERE id = $1`
	rr := &model.ReviewReport{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&rr.ID, &rr.RunID, &rr.TaskID, &rr.ReviewerID, &rr.ArtifactVersionID, &rr.Verdict,
		&rr.Summary, &rr.Findings, &rr.Recommendations, &rr.Metadata, &rr.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reviewReport.GetByID: %w", err)
	}
	return rr, nil
}

type ReviewListParams struct {
	RunID      string
	TaskID     string
	ReviewerID string
	Verdict    string
	Limit      int
	Offset     int
}

func (r *ReviewReportRepo) List(ctx context.Context, p ReviewListParams) ([]*model.ReviewReport, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if p.RunID != "" {
		where += fmt.Sprintf(" AND run_id = $%d", idx)
		args = append(args, p.RunID)
		idx++
	}
	if p.TaskID != "" {
		where += fmt.Sprintf(" AND task_id = $%d", idx)
		args = append(args, p.TaskID)
		idx++
	}
	if p.ReviewerID != "" {
		where += fmt.Sprintf(" AND reviewer_id = $%d", idx)
		args = append(args, p.ReviewerID)
		idx++
	}
	if p.Verdict != "" {
		where += fmt.Sprintf(" AND verdict = $%d", idx)
		args = append(args, p.Verdict)
		idx++
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT count(*) FROM review_report "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("reviewReport.List count: %w", err)
	}

	q := fmt.Sprintf(`
		SELECT id, run_id, task_id, reviewer_id, artifact_version_id, verdict,
		       summary, findings, recommendations, metadata, created_at
		FROM review_report %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, idx, idx+1)
	args = append(args, p.Limit, p.Offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("reviewReport.List: %w", err)
	}
	defer rows.Close()

	var list []*model.ReviewReport
	for rows.Next() {
		rr := &model.ReviewReport{}
		if err := rows.Scan(
			&rr.ID, &rr.RunID, &rr.TaskID, &rr.ReviewerID, &rr.ArtifactVersionID, &rr.Verdict,
			&rr.Summary, &rr.Findings, &rr.Recommendations, &rr.Metadata, &rr.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("reviewReport.List scan: %w", err)
		}
		list = append(list, rr)
	}
	return list, total, rows.Err()
}

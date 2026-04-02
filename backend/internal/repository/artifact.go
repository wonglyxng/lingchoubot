package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
)

// ---------- ArtifactRepo ----------

type ArtifactRepo struct {
	db *sql.DB
}

func NewArtifactRepo(db *sql.DB) *ArtifactRepo {
	return &ArtifactRepo{db: db}
}

func (r *ArtifactRepo) Create(ctx context.Context, a *model.Artifact) error {
	const q = `
		INSERT INTO artifact (project_id, task_id, name, artifact_type, description, created_by, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, q,
		a.ProjectID, a.TaskID, a.Name, a.ArtifactType, a.Description, a.CreatedBy, a.Metadata,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (r *ArtifactRepo) Delete(ctx context.Context, id string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM artifact WHERE id = $1`, id); err != nil {
		return fmt.Errorf("artifact.Delete: %w", err)
	}
	return nil
}

func (r *ArtifactRepo) GetByID(ctx context.Context, id string) (*model.Artifact, error) {
	const q = `
		SELECT id, project_id, task_id, name, artifact_type, description, created_by, metadata,
		       created_at, updated_at
		FROM artifact WHERE id = $1`
	a := &model.Artifact{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&a.ID, &a.ProjectID, &a.TaskID, &a.Name, &a.ArtifactType,
		&a.Description, &a.CreatedBy, &a.Metadata, &a.CreatedAt, &a.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("artifact.GetByID: %w", err)
	}
	return a, nil
}

type ArtifactListParams struct {
	ProjectID    string
	TaskID       string
	ArtifactType string
	Limit        int
	Offset       int
}

func (r *ArtifactRepo) List(ctx context.Context, p ArtifactListParams) ([]*model.Artifact, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if p.ProjectID != "" {
		where += fmt.Sprintf(" AND project_id = $%d", idx)
		args = append(args, p.ProjectID)
		idx++
	}
	if p.TaskID != "" {
		where += fmt.Sprintf(" AND task_id = $%d", idx)
		args = append(args, p.TaskID)
		idx++
	}
	if p.ArtifactType != "" {
		where += fmt.Sprintf(" AND artifact_type = $%d", idx)
		args = append(args, p.ArtifactType)
		idx++
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT count(*) FROM artifact "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("artifact.List count: %w", err)
	}

	q := fmt.Sprintf(`
		SELECT id, project_id, task_id, name, artifact_type, description, created_by, metadata,
		       created_at, updated_at
		FROM artifact %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, idx, idx+1)
	args = append(args, p.Limit, p.Offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("artifact.List: %w", err)
	}
	defer rows.Close()

	var list []*model.Artifact
	for rows.Next() {
		a := &model.Artifact{}
		if err := rows.Scan(
			&a.ID, &a.ProjectID, &a.TaskID, &a.Name, &a.ArtifactType,
			&a.Description, &a.CreatedBy, &a.Metadata, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("artifact.List scan: %w", err)
		}
		list = append(list, a)
	}
	return list, total, rows.Err()
}

// ---------- ArtifactVersionRepo ----------

type ArtifactVersionRepo struct {
	db *sql.DB
}

func NewArtifactVersionRepo(db *sql.DB) *ArtifactVersionRepo {
	return &ArtifactVersionRepo{db: db}
}

func (r *ArtifactVersionRepo) NextVersion(ctx context.Context, artifactID string) (int, error) {
	var maxVer sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT max(version) FROM artifact_version WHERE artifact_id = $1`, artifactID,
	).Scan(&maxVer)
	if err != nil {
		return 0, fmt.Errorf("artifactVersion.NextVersion: %w", err)
	}
	if !maxVer.Valid {
		return 1, nil
	}
	return int(maxVer.Int64) + 1, nil
}

func (r *ArtifactVersionRepo) Create(ctx context.Context, v *model.ArtifactVersion) error {
	const q = `
		INSERT INTO artifact_version (artifact_id, version, uri, content_type, size_bytes, checksum,
		                               change_summary, created_by, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, q,
		v.ArtifactID, v.Version, v.URI, v.ContentType, v.SizeBytes, v.Checksum,
		v.ChangeSummary, v.CreatedBy, v.Metadata,
	).Scan(&v.ID, &v.CreatedAt)
}

func (r *ArtifactVersionRepo) GetByID(ctx context.Context, id string) (*model.ArtifactVersion, error) {
	const q = `
		SELECT id, artifact_id, version, uri, content_type, size_bytes, checksum,
		       change_summary, created_by, metadata, created_at
		FROM artifact_version WHERE id = $1`
	v := &model.ArtifactVersion{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&v.ID, &v.ArtifactID, &v.Version, &v.URI, &v.ContentType, &v.SizeBytes,
		&v.Checksum, &v.ChangeSummary, &v.CreatedBy, &v.Metadata, &v.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("artifactVersion.GetByID: %w", err)
	}
	return v, nil
}

func (r *ArtifactVersionRepo) ListByArtifact(ctx context.Context, artifactID string) ([]*model.ArtifactVersion, error) {
	const q = `
		SELECT id, artifact_id, version, uri, content_type, size_bytes, checksum,
		       change_summary, created_by, metadata, created_at
		FROM artifact_version WHERE artifact_id = $1 ORDER BY version DESC`
	rows, err := r.db.QueryContext(ctx, q, artifactID)
	if err != nil {
		return nil, fmt.Errorf("artifactVersion.ListByArtifact: %w", err)
	}
	defer rows.Close()

	var list []*model.ArtifactVersion
	for rows.Next() {
		v := &model.ArtifactVersion{}
		if err := rows.Scan(
			&v.ID, &v.ArtifactID, &v.Version, &v.URI, &v.ContentType, &v.SizeBytes,
			&v.Checksum, &v.ChangeSummary, &v.CreatedBy, &v.Metadata, &v.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("artifactVersion.ListByArtifact scan: %w", err)
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

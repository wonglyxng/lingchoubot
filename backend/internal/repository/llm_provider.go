package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
)

type LLMProviderRepo struct {
	db *sql.DB
}

func NewLLMProviderRepo(db *sql.DB) *LLMProviderRepo {
	return &LLMProviderRepo{db: db}
}

const llmProviderColumns = `id, key, name, base_url, api_key, is_builtin, is_enabled, sort_order, metadata, created_at, updated_at`

func scanLLMProvider(s interface{ Scan(dest ...any) error }) (*model.LLMProvider, error) {
	p := &model.LLMProvider{}
	err := s.Scan(&p.ID, &p.Key, &p.Name, &p.BaseURL, &p.APIKey, &p.IsBuiltin, &p.IsEnabled,
		&p.SortOrder, &p.Metadata, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (r *LLMProviderRepo) Create(ctx context.Context, p *model.LLMProvider) error {
	const q = `
		INSERT INTO llm_provider (key, name, base_url, api_key, is_builtin, is_enabled, sort_order, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, q,
		p.Key, p.Name, p.BaseURL, p.APIKey, p.IsBuiltin, p.IsEnabled, p.SortOrder, p.Metadata,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *LLMProviderRepo) GetByID(ctx context.Context, id string) (*model.LLMProvider, error) {
	q := `SELECT ` + llmProviderColumns + ` FROM llm_provider WHERE id = $1`
	p, err := scanLLMProvider(r.db.QueryRowContext(ctx, q, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("llm_provider.GetByID: %w", err)
	}
	return p, nil
}

func (r *LLMProviderRepo) GetByKey(ctx context.Context, key string) (*model.LLMProvider, error) {
	q := `SELECT ` + llmProviderColumns + ` FROM llm_provider WHERE key = $1`
	p, err := scanLLMProvider(r.db.QueryRowContext(ctx, q, key))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("llm_provider.GetByKey: %w", err)
	}
	return p, nil
}

func (r *LLMProviderRepo) List(ctx context.Context, enabledOnly bool) ([]*model.LLMProvider, error) {
	q := `SELECT ` + llmProviderColumns + ` FROM llm_provider`
	if enabledOnly {
		q += ` WHERE is_enabled = true`
	}
	q += ` ORDER BY sort_order, created_at`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("llm_provider.List: %w", err)
	}
	defer rows.Close()

	var list []*model.LLMProvider
	for rows.Next() {
		p, err := scanLLMProvider(rows)
		if err != nil {
			return nil, fmt.Errorf("llm_provider.List scan: %w", err)
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *LLMProviderRepo) Update(ctx context.Context, p *model.LLMProvider) error {
	const q = `
		UPDATE llm_provider
		SET key = $2, name = $3, base_url = $4, api_key = $5, is_enabled = $6,
		    sort_order = $7, metadata = $8, updated_at = now()
		WHERE id = $1
		RETURNING updated_at`
	return r.db.QueryRowContext(ctx, q,
		p.ID, p.Key, p.Name, p.BaseURL, p.APIKey, p.IsEnabled, p.SortOrder, p.Metadata,
	).Scan(&p.UpdatedAt)
}

func (r *LLMProviderRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM llm_provider WHERE id = $1`, id)
	return err
}

// --- LLM Model ---

const llmModelColumns = `id, provider_id, model_id, name, is_default, sort_order, metadata, created_at, updated_at`

func scanLLMModel(s interface{ Scan(dest ...any) error }) (*model.LLMModel, error) {
	m := &model.LLMModel{}
	err := s.Scan(&m.ID, &m.ProviderID, &m.ModelID, &m.Name, &m.IsDefault,
		&m.SortOrder, &m.Metadata, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}

func (r *LLMProviderRepo) CreateModel(ctx context.Context, m *model.LLMModel) error {
	const q = `
		INSERT INTO llm_model (provider_id, model_id, name, is_default, sort_order, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, q,
		m.ProviderID, m.ModelID, m.Name, m.IsDefault, m.SortOrder, m.Metadata,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

func (r *LLMProviderRepo) GetModelByID(ctx context.Context, id string) (*model.LLMModel, error) {
	q := `SELECT ` + llmModelColumns + ` FROM llm_model WHERE id = $1`
	m, err := scanLLMModel(r.db.QueryRowContext(ctx, q, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("llm_model.GetByID: %w", err)
	}
	return m, nil
}

func (r *LLMProviderRepo) ListModelsByProvider(ctx context.Context, providerID string) ([]*model.LLMModel, error) {
	q := `SELECT ` + llmModelColumns + ` FROM llm_model WHERE provider_id = $1 ORDER BY sort_order, created_at`
	rows, err := r.db.QueryContext(ctx, q, providerID)
	if err != nil {
		return nil, fmt.Errorf("llm_model.ListByProvider: %w", err)
	}
	defer rows.Close()

	var list []*model.LLMModel
	for rows.Next() {
		m, err := scanLLMModel(rows)
		if err != nil {
			return nil, fmt.Errorf("llm_model.ListByProvider scan: %w", err)
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (r *LLMProviderRepo) UpdateModel(ctx context.Context, m *model.LLMModel) error {
	const q = `
		UPDATE llm_model
		SET model_id = $2, name = $3, is_default = $4, sort_order = $5, metadata = $6, updated_at = now()
		WHERE id = $1
		RETURNING updated_at`
	return r.db.QueryRowContext(ctx, q,
		m.ID, m.ModelID, m.Name, m.IsDefault, m.SortOrder, m.Metadata,
	).Scan(&m.UpdatedAt)
}

func (r *LLMProviderRepo) DeleteModel(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM llm_model WHERE id = $1`, id)
	return err
}

// ListAllModels returns all models ordered by provider and sort_order.
func (r *LLMProviderRepo) ListAllModels(ctx context.Context) ([]*model.LLMModel, error) {
	q := `SELECT ` + llmModelColumns + ` FROM llm_model ORDER BY provider_id, sort_order, created_at`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("llm_model.ListAll: %w", err)
	}
	defer rows.Close()

	var list []*model.LLMModel
	for rows.Next() {
		m, err := scanLLMModel(rows)
		if err != nil {
			return nil, fmt.Errorf("llm_model.ListAll scan: %w", err)
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

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
		INSERT INTO agent (name, role, role_code, agent_type, specialization, description, reports_to, status,
		                   managed_roles, allowed_tools, risk_level, capabilities, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, q,
		a.Name, a.Role, a.RoleCode, a.AgentType, a.Specialization, a.Description, a.ReportsTo, a.Status,
		a.ManagedRoles, a.AllowedTools, a.RiskLevel, a.Capabilities, a.Metadata,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

// agentColumns is the canonical SELECT column list for agent queries.
const agentColumns = `id, name, role, role_code, agent_type, specialization, description, reports_to, status,
		managed_roles, allowed_tools, risk_level, capabilities, metadata, created_at, updated_at`

func scanAgent(s interface{ Scan(dest ...any) error }) (*model.Agent, error) {
	a := &model.Agent{}
	err := s.Scan(
		&a.ID, &a.Name, &a.Role, &a.RoleCode, &a.AgentType, &a.Specialization, &a.Description, &a.ReportsTo,
		&a.Status, &a.ManagedRoles, &a.AllowedTools, &a.RiskLevel, &a.Capabilities, &a.Metadata,
		&a.CreatedAt, &a.UpdatedAt,
	)
	return a, err
}

func (r *AgentRepo) GetByID(ctx context.Context, id string) (*model.Agent, error) {
	q := `SELECT ` + agentColumns + ` FROM agent WHERE id = $1`
	a, err := scanAgent(r.db.QueryRowContext(ctx, q, id))
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

	q := `SELECT ` + agentColumns + ` FROM agent ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("agent.List: %w", err)
	}
	defer rows.Close()

	var list []*model.Agent
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("agent.List scan: %w", err)
		}
		list = append(list, a)
	}
	return list, total, rows.Err()
}

func (r *AgentRepo) Update(ctx context.Context, a *model.Agent) error {
	const q = `
		UPDATE agent
		SET name = $2, role = $3, role_code = $4, agent_type = $5, specialization = $6, description = $7,
		    reports_to = $8, status = $9, managed_roles = $10, allowed_tools = $11, risk_level = $12,
		    capabilities = $13, metadata = $14, updated_at = now()
		WHERE id = $1
		RETURNING updated_at`
	return r.db.QueryRowContext(ctx, q,
		a.ID, a.Name, a.Role, a.RoleCode, a.AgentType, a.Specialization, a.Description,
		a.ReportsTo, a.Status, a.ManagedRoles, a.AllowedTools, a.RiskLevel,
		a.Capabilities, a.Metadata,
	).Scan(&a.UpdatedAt)
}

func (r *AgentRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM agent WHERE id = $1`, id)
	return err
}

// GetSubordinates returns all agents that directly report to the given agent.
func (r *AgentRepo) GetSubordinates(ctx context.Context, agentID string) ([]*model.Agent, error) {
	q := `SELECT ` + agentColumns + ` FROM agent WHERE reports_to = $1 ORDER BY created_at`
	rows, err := r.db.QueryContext(ctx, q, agentID)
	if err != nil {
		return nil, fmt.Errorf("agent.GetSubordinates: %w", err)
	}
	defer rows.Close()

	var list []*model.Agent
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, fmt.Errorf("agent.GetSubordinates scan: %w", err)
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// GetOrgTree returns a flat list of all agents under the given root (recursive via CTE).
// If rootID is empty, returns the full org tree.
func (r *AgentRepo) GetOrgTree(ctx context.Context, rootID string) ([]*model.Agent, error) {
	var q string
	var args []interface{}

	if rootID != "" {
		q = `
			WITH RECURSIVE tree AS (
				SELECT ` + agentColumns + `, 0 AS depth
				FROM agent WHERE id = $1
				UNION ALL
				SELECT a.id, a.name, a.role, a.role_code, a.agent_type, a.specialization, a.description, a.reports_to, a.status,
				       a.managed_roles, a.allowed_tools, a.risk_level, a.capabilities, a.metadata, a.created_at, a.updated_at,
				       t.depth + 1
				FROM agent a INNER JOIN tree t ON a.reports_to = t.id
			)
			SELECT ` + agentColumns + ` FROM tree ORDER BY depth, created_at`
		args = []interface{}{rootID}
	} else {
		q = `
			WITH RECURSIVE tree AS (
				SELECT ` + agentColumns + `, 0 AS depth
				FROM agent WHERE reports_to IS NULL
				UNION ALL
				SELECT a.id, a.name, a.role, a.role_code, a.agent_type, a.specialization, a.description, a.reports_to, a.status,
				       a.managed_roles, a.allowed_tools, a.risk_level, a.capabilities, a.metadata, a.created_at, a.updated_at,
				       t.depth + 1
				FROM agent a INNER JOIN tree t ON a.reports_to = t.id
			)
			SELECT ` + agentColumns + ` FROM tree ORDER BY depth, created_at`
	}

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("agent.GetOrgTree: %w", err)
	}
	defer rows.Close()

	var list []*model.Agent
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, fmt.Errorf("agent.GetOrgTree scan: %w", err)
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// FindByRoleAndSpec finds the first active agent matching role and specialization.
// If no exact specialization match is found, falls back to a "general" specialization agent.
func (r *AgentRepo) FindByRoleAndSpec(ctx context.Context, role model.AgentRole, spec model.AgentSpecialization) (*model.Agent, error) {
	q := `SELECT ` + agentColumns + `
		FROM agent
		WHERE role = $1 AND status = 'active' AND specialization IN ($2, 'general')
		ORDER BY CASE WHEN specialization = $2 THEN 0 ELSE 1 END, created_at
		LIMIT 1`
	a, err := scanAgent(r.db.QueryRowContext(ctx, q, role, spec))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("agent.FindByRoleAndSpec: %w", err)
	}
	return a, nil
}

// FindByRoleCode finds the first active agent matching a specific role_code.
func (r *AgentRepo) FindByRoleCode(ctx context.Context, roleCode model.RoleCode) (*model.Agent, error) {
	q := `SELECT ` + agentColumns + `
		FROM agent
		WHERE role_code = $1 AND status = 'active'
		ORDER BY created_at
		LIMIT 1`
	a, err := scanAgent(r.db.QueryRowContext(ctx, q, roleCode))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("agent.FindByRoleCode: %w", err)
	}
	return a, nil
}

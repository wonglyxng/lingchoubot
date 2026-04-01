DROP INDEX IF EXISTS uq_agent_role_code_non_empty;

CREATE INDEX IF NOT EXISTS idx_agent_role_code ON agent(role_code);
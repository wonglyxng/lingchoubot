-- 000008 rollback: remove agent capability columns

DROP INDEX IF EXISTS idx_agent_status_role;
DROP INDEX IF EXISTS idx_agent_role_specialization;
ALTER TABLE agent DROP COLUMN IF EXISTS specialization;
ALTER TABLE agent DROP COLUMN IF EXISTS agent_type;

-- 000010 rollback: remove org role matrix columns

DROP INDEX IF EXISTS idx_task_owner_supervisor;
DROP INDEX IF EXISTS idx_task_execution_domain;
ALTER TABLE task DROP COLUMN IF EXISTS owner_supervisor_id;
ALTER TABLE task DROP COLUMN IF EXISTS execution_domain;

DROP INDEX IF EXISTS idx_agent_role_code;
ALTER TABLE agent DROP COLUMN IF EXISTS risk_level;
ALTER TABLE agent DROP COLUMN IF EXISTS allowed_tools;
ALTER TABLE agent DROP COLUMN IF EXISTS managed_roles;
ALTER TABLE agent DROP COLUMN IF EXISTS role_code;

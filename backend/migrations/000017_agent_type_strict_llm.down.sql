-- 000017 rollback: 恢复 agent_type 对 mock 的兼容

ALTER TABLE agent
    ALTER COLUMN agent_type SET DEFAULT 'mock';

ALTER TABLE agent DROP CONSTRAINT IF EXISTS chk_agent_agent_type;

ALTER TABLE agent
    ADD CONSTRAINT chk_agent_agent_type
    CHECK (agent_type IN ('mock', 'llm', 'human'));
-- 000017: 收紧 agent_type 到严格 llm/human 模式

UPDATE agent
SET agent_type = 'llm'
WHERE agent_type = 'mock';

ALTER TABLE agent
    ALTER COLUMN agent_type SET DEFAULT 'llm';

ALTER TABLE agent DROP CONSTRAINT IF EXISTS chk_agent_agent_type;

DO $$
DECLARE legacy_constraint text;
BEGIN
    SELECT conname INTO legacy_constraint
    FROM pg_constraint
    WHERE conrelid = 'agent'::regclass
      AND contype = 'c'
      AND conname <> 'chk_agent_agent_type'
      AND pg_get_constraintdef(oid) LIKE '%agent_type%'
    LIMIT 1;

    IF legacy_constraint IS NOT NULL THEN
        EXECUTE format('ALTER TABLE agent DROP CONSTRAINT %I', legacy_constraint);
    END IF;
END $$;

ALTER TABLE agent
    ADD CONSTRAINT chk_agent_agent_type
    CHECK (agent_type IN ('llm', 'human'));
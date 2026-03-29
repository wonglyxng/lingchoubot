-- 000006: 第7周 — tool_call (工具调用记录)

CREATE TABLE IF NOT EXISTS tool_call (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id         UUID REFERENCES task(id) ON DELETE SET NULL,
    agent_id        UUID NOT NULL REFERENCES agent(id),
    tool_name       TEXT NOT NULL,
    input           JSONB NOT NULL DEFAULT '{}',
    output          JSONB NOT NULL DEFAULT '{}',
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'running', 'success', 'failed', 'denied')),
    error_message   TEXT NOT NULL DEFAULT '',
    duration_ms     INTEGER NOT NULL DEFAULT 0,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_tool_call_task    ON tool_call(task_id);
CREATE INDEX IF NOT EXISTS idx_tool_call_agent   ON tool_call(agent_id);
CREATE INDEX IF NOT EXISTS idx_tool_call_tool    ON tool_call(tool_name);
CREATE INDEX IF NOT EXISTS idx_tool_call_status  ON tool_call(status);

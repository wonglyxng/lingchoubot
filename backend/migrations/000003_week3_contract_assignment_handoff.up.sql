-- 000003: 第3周 — task_contract, task_assignment, handoff_snapshot

-- ============================================================
-- task_contract — 任务契约（与 task 一对多，按版本递增）
-- ============================================================
CREATE TABLE IF NOT EXISTS task_contract (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id             UUID NOT NULL REFERENCES task(id) ON DELETE CASCADE,
    version             INT NOT NULL DEFAULT 1,
    scope               TEXT NOT NULL DEFAULT '',
    non_goals           JSONB NOT NULL DEFAULT '[]',
    done_definition     JSONB NOT NULL DEFAULT '[]',
    verification_plan   JSONB NOT NULL DEFAULT '[]',
    acceptance_criteria JSONB NOT NULL DEFAULT '[]',
    tool_permissions    JSONB NOT NULL DEFAULT '[]',
    escalation_policy   JSONB NOT NULL DEFAULT '{}',
    metadata            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(task_id, version)
);

CREATE INDEX IF NOT EXISTS idx_task_contract_task ON task_contract(task_id);

-- ============================================================
-- task_assignment — 任务分派记录（保留完整历史）
-- ============================================================
CREATE TABLE IF NOT EXISTS task_assignment (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id         UUID NOT NULL REFERENCES task(id) ON DELETE CASCADE,
    agent_id        UUID NOT NULL REFERENCES agent(id) ON DELETE CASCADE,
    assigned_by     UUID REFERENCES agent(id),
    role            TEXT NOT NULL DEFAULT 'executor'
                    CHECK (role IN ('executor', 'reviewer')),
    status          TEXT NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active', 'completed', 'revoked')),
    note            TEXT NOT NULL DEFAULT '',
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_task_assignment_task  ON task_assignment(task_id);
CREATE INDEX IF NOT EXISTS idx_task_assignment_agent ON task_assignment(agent_id);

-- ============================================================
-- handoff_snapshot — 交接快照
-- ============================================================
CREATE TABLE IF NOT EXISTS handoff_snapshot (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id         UUID NOT NULL REFERENCES task(id) ON DELETE CASCADE,
    agent_id        UUID NOT NULL REFERENCES agent(id),
    summary         TEXT NOT NULL DEFAULT '',
    completed_items JSONB NOT NULL DEFAULT '[]',
    pending_items   JSONB NOT NULL DEFAULT '[]',
    risks           JSONB NOT NULL DEFAULT '[]',
    next_steps      JSONB NOT NULL DEFAULT '[]',
    artifact_refs   JSONB NOT NULL DEFAULT '[]',
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_handoff_snapshot_task  ON handoff_snapshot(task_id);
CREATE INDEX IF NOT EXISTS idx_handoff_snapshot_agent ON handoff_snapshot(agent_id);

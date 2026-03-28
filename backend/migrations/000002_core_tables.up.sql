-- 000002: 核心业务表 — project, project_phase, agent, task, audit_log

-- ============================================================
-- project
-- ============================================================
CREATE TABLE IF NOT EXISTS project (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'planning'
                CHECK (status IN ('planning','active','paused','completed','cancelled')),
    owner_agent_id UUID,                       -- 项目负责人 Agent（可后续关联）
    metadata    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- project_phase
-- ============================================================
CREATE TABLE IF NOT EXISTS project_phase (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id  UUID NOT NULL REFERENCES project(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending','active','completed','skipped')),
    sort_order  INT NOT NULL DEFAULT 0,
    metadata    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_project_phase_project ON project_phase(project_id);

-- ============================================================
-- agent
-- ============================================================
CREATE TABLE IF NOT EXISTS agent (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL,
    role        TEXT NOT NULL
                CHECK (role IN ('pm','supervisor','worker','reviewer')),
    description TEXT NOT NULL DEFAULT '',
    reports_to  UUID REFERENCES agent(id),      -- 上级 Agent
    status      TEXT NOT NULL DEFAULT 'active'
                CHECK (status IN ('active','inactive')),
    capabilities JSONB NOT NULL DEFAULT '[]',
    metadata    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_reports_to ON agent(reports_to);

-- ============================================================
-- task
-- ============================================================
CREATE TABLE IF NOT EXISTS task (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id      UUID NOT NULL REFERENCES project(id) ON DELETE CASCADE,
    phase_id        UUID REFERENCES project_phase(id) ON DELETE SET NULL,
    parent_task_id  UUID REFERENCES task(id) ON DELETE SET NULL,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN (
                        'pending','assigned','in_progress',
                        'in_review','revision_required',
                        'completed','failed','cancelled','blocked'
                    )),
    priority        INT NOT NULL DEFAULT 0,
    assignee_id     UUID REFERENCES agent(id) ON DELETE SET NULL,
    input_context   JSONB NOT NULL DEFAULT '{}',
    output_summary  JSONB NOT NULL DEFAULT '{}',
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_task_project   ON task(project_id);
CREATE INDEX IF NOT EXISTS idx_task_phase     ON task(phase_id);
CREATE INDEX IF NOT EXISTS idx_task_assignee  ON task(assignee_id);
CREATE INDEX IF NOT EXISTS idx_task_status    ON task(status);
CREATE INDEX IF NOT EXISTS idx_task_parent    ON task(parent_task_id);

-- ============================================================
-- audit_log
-- ============================================================
CREATE TABLE IF NOT EXISTS audit_log (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    actor_type    TEXT NOT NULL,                 -- 'user' | 'agent' | 'system'
    actor_id      TEXT NOT NULL DEFAULT '',
    event_type    TEXT NOT NULL,                 -- 如 'project.created', 'task.status_changed'
    event_summary TEXT NOT NULL DEFAULT '',
    target_type   TEXT NOT NULL DEFAULT '',      -- 'project' | 'task' | 'agent' 等
    target_id     TEXT NOT NULL DEFAULT '',
    before_state  JSONB,
    after_state   JSONB,
    metadata      JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_target   ON audit_log(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_event    ON audit_log(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_log_created  ON audit_log(created_at);

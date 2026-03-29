-- 工作流运行态持久化表
-- WP2-01: 将内存 RunStore 升级为 DB 持久化

CREATE TABLE workflow_run (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES project(id),
    status      TEXT NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    summary     TEXT NOT NULL DEFAULT '',
    error       TEXT NOT NULL DEFAULT '',
    started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_workflow_run_project ON workflow_run(project_id);
CREATE INDEX idx_workflow_run_status ON workflow_run(status);

CREATE TABLE workflow_step (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id      UUID NOT NULL REFERENCES workflow_run(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    agent_role  TEXT NOT NULL DEFAULT '',
    agent_id    UUID,
    task_id     UUID,
    phase_id    UUID,
    status      TEXT NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending', 'running', 'completed', 'failed', 'skipped')),
    summary     TEXT NOT NULL DEFAULT '',
    error       TEXT NOT NULL DEFAULT '',
    sort_order  INT NOT NULL DEFAULT 0,
    started_at  TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_workflow_step_run ON workflow_step(run_id);
CREATE INDEX idx_workflow_step_task ON workflow_step(task_id);

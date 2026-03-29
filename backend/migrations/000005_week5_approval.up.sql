-- 000005: 第5周 — approval_request

CREATE TABLE IF NOT EXISTS approval_request (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id      UUID NOT NULL REFERENCES project(id) ON DELETE CASCADE,
    task_id         UUID REFERENCES task(id) ON DELETE SET NULL,
    artifact_id     UUID REFERENCES artifact(id) ON DELETE SET NULL,
    requested_by    UUID NOT NULL REFERENCES agent(id),
    approver_type   TEXT NOT NULL DEFAULT 'user'
                    CHECK (approver_type IN ('user', 'agent')),
    approver_id     TEXT NOT NULL DEFAULT '',
    title           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'approved', 'rejected')),
    decision_note   TEXT NOT NULL DEFAULT '',
    decided_at      TIMESTAMPTZ,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_approval_request_project ON approval_request(project_id);
CREATE INDEX IF NOT EXISTS idx_approval_request_task    ON approval_request(task_id);
CREATE INDEX IF NOT EXISTS idx_approval_request_status  ON approval_request(status);

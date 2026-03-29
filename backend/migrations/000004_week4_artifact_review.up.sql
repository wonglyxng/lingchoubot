-- 000004: 第4周 — artifact, artifact_version, review_report

-- ============================================================
-- artifact — 工件（任务或阶段产生的可验证交付物）
-- ============================================================
CREATE TABLE IF NOT EXISTS artifact (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id      UUID NOT NULL REFERENCES project(id) ON DELETE CASCADE,
    task_id         UUID REFERENCES task(id) ON DELETE SET NULL,
    name            TEXT NOT NULL,
    artifact_type   TEXT NOT NULL
                    CHECK (artifact_type IN (
                        'prd','design','api_spec','schema_sql',
                        'source_code','test_report','deployment_plan',
                        'release_note','other'
                    )),
    description     TEXT NOT NULL DEFAULT '',
    created_by      UUID REFERENCES agent(id) ON DELETE SET NULL,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_artifact_project  ON artifact(project_id);
CREATE INDEX IF NOT EXISTS idx_artifact_task     ON artifact(task_id);
CREATE INDEX IF NOT EXISTS idx_artifact_type     ON artifact(artifact_type);

-- ============================================================
-- artifact_version — 工件版本（同一工件按版本号递增）
-- ============================================================
CREATE TABLE IF NOT EXISTS artifact_version (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    artifact_id     UUID NOT NULL REFERENCES artifact(id) ON DELETE CASCADE,
    version         INT NOT NULL DEFAULT 1,
    uri             TEXT NOT NULL DEFAULT '',
    content_type    TEXT NOT NULL DEFAULT '',
    size_bytes      BIGINT NOT NULL DEFAULT 0,
    checksum        TEXT NOT NULL DEFAULT '',
    change_summary  TEXT NOT NULL DEFAULT '',
    created_by      UUID REFERENCES agent(id) ON DELETE SET NULL,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(artifact_id, version)
);

CREATE INDEX IF NOT EXISTS idx_artifact_version_artifact ON artifact_version(artifact_id);

-- ============================================================
-- review_report — 独立评审报告
-- ============================================================
CREATE TABLE IF NOT EXISTS review_report (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id             UUID NOT NULL REFERENCES task(id) ON DELETE CASCADE,
    reviewer_id         UUID NOT NULL REFERENCES agent(id),
    artifact_version_id UUID REFERENCES artifact_version(id) ON DELETE SET NULL,
    verdict             TEXT NOT NULL
                        CHECK (verdict IN ('approved','rejected','needs_revision')),
    summary             TEXT NOT NULL DEFAULT '',
    findings            JSONB NOT NULL DEFAULT '[]',
    recommendations     JSONB NOT NULL DEFAULT '[]',
    metadata            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_review_report_task     ON review_report(task_id);
CREATE INDEX IF NOT EXISTS idx_review_report_reviewer ON review_report(reviewer_id);

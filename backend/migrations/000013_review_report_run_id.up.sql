-- 000013: bind review_report to workflow_run for per-run tracing and filtering

ALTER TABLE review_report
    ADD COLUMN IF NOT EXISTS run_id UUID REFERENCES workflow_run(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_review_report_run ON review_report(run_id);
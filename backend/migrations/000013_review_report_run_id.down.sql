-- Revert: remove review_report.run_id tracing column

DROP INDEX IF EXISTS idx_review_report_run;

ALTER TABLE review_report
    DROP COLUMN IF EXISTS run_id;
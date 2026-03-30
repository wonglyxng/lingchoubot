-- WP2-08: Revert 'cancelled' status from workflow_run

ALTER TABLE workflow_run DROP CONSTRAINT IF EXISTS workflow_run_status_check;
ALTER TABLE workflow_run ADD CONSTRAINT workflow_run_status_check
    CHECK (status IN ('pending', 'running', 'completed', 'failed'));

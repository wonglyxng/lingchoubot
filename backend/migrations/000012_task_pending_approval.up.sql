-- 000012: Add pending_approval to task status CHECK constraint
-- This status sits between in_review and completed: review approved -> pending_approval -> (approval decides) -> completed/revision_required

ALTER TABLE task DROP CONSTRAINT IF EXISTS task_status_check;

ALTER TABLE task ADD CONSTRAINT task_status_check
    CHECK (status IN (
        'pending','assigned','in_progress',
        'in_review','pending_approval','revision_required',
        'completed','failed','cancelled','blocked'
    ));

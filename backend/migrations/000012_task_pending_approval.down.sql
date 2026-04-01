-- Revert: remove pending_approval from task status CHECK constraint

ALTER TABLE task DROP CONSTRAINT IF EXISTS task_status_check;

ALTER TABLE task ADD CONSTRAINT task_status_check
    CHECK (status IN (
        'pending','assigned','in_progress',
        'in_review','revision_required',
        'completed','failed','cancelled','blocked'
    ));

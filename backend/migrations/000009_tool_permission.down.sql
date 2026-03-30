-- 000009 down: 回滚工具权限细化

DROP INDEX IF EXISTS idx_tool_call_agent_tool_status;

ALTER TABLE tool_call DROP COLUMN IF EXISTS denied_reason;
ALTER TABLE tool_call DROP COLUMN IF EXISTS action;

ALTER TABLE tool_call DROP CONSTRAINT IF EXISTS tool_call_status_check;
ALTER TABLE tool_call ADD CONSTRAINT tool_call_status_check
    CHECK (status IN ('pending', 'running', 'success', 'failed', 'denied'));

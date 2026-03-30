-- 000009: WP2-05 — 工具权限细化

-- tool_call 新增 escalated 状态
ALTER TABLE tool_call DROP CONSTRAINT IF EXISTS tool_call_status_check;
ALTER TABLE tool_call ADD CONSTRAINT tool_call_status_check
    CHECK (status IN ('pending', 'running', 'success', 'failed', 'denied', 'escalated'));

-- tool_call 添加 action 和 denied_reason 字段
ALTER TABLE tool_call ADD COLUMN IF NOT EXISTS action TEXT NOT NULL DEFAULT '';
ALTER TABLE tool_call ADD COLUMN IF NOT EXISTS denied_reason TEXT NOT NULL DEFAULT '';

-- 复合索引：按 agent + tool + status 查询
CREATE INDEX IF NOT EXISTS idx_tool_call_agent_tool_status ON tool_call(agent_id, tool_name, status);

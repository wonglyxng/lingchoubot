-- 000010: WP2-07 — 组织角色矩阵与主管路由
-- Agent 新增 role_code / managed_roles / allowed_tools / risk_level
-- Task 新增 execution_domain / owner_supervisor_id

-- ============================================================
-- agent 表扩展
-- ============================================================

-- 业务职责码，细粒度角色标识
ALTER TABLE agent ADD COLUMN IF NOT EXISTS role_code TEXT NOT NULL DEFAULT '';

-- 主管可管理的下级 role_code 列表
ALTER TABLE agent ADD COLUMN IF NOT EXISTS managed_roles JSONB NOT NULL DEFAULT '[]';

-- 默认允许的工具模板
ALTER TABLE agent ADD COLUMN IF NOT EXISTS allowed_tools JSONB NOT NULL DEFAULT '[]';

-- 风险级别
ALTER TABLE agent ADD COLUMN IF NOT EXISTS risk_level TEXT NOT NULL DEFAULT 'medium'
    CHECK (risk_level IN ('low', 'medium', 'high', 'critical'));

-- 索引
CREATE INDEX IF NOT EXISTS idx_agent_role_code ON agent(role_code);

-- ============================================================
-- task 表扩展
-- ============================================================

-- 任务执行域：development / qa / general
ALTER TABLE task ADD COLUMN IF NOT EXISTS execution_domain TEXT NOT NULL DEFAULT 'general'
    CHECK (execution_domain IN ('development', 'qa', 'general'));

-- 责任主管 ID
ALTER TABLE task ADD COLUMN IF NOT EXISTS owner_supervisor_id UUID REFERENCES agent(id) ON DELETE SET NULL;

-- 索引
CREATE INDEX IF NOT EXISTS idx_task_execution_domain ON task(execution_domain);
CREATE INDEX IF NOT EXISTS idx_task_owner_supervisor ON task(owner_supervisor_id);

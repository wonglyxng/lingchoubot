-- 000008: Agent 能力模型扩展
-- 新增 agent_type 和 specialization 列，扩展 role CHECK 约束

-- 1. 新增 agent_type 列（执行类型：mock/llm/human）
ALTER TABLE agent ADD COLUMN IF NOT EXISTS agent_type TEXT NOT NULL DEFAULT 'mock'
    CHECK (agent_type IN ('mock', 'llm', 'human'));

-- 2. 新增 specialization 列（专业方向，worker 角色可细分）
ALTER TABLE agent ADD COLUMN IF NOT EXISTS specialization TEXT NOT NULL DEFAULT 'general'
    CHECK (specialization IN ('general', 'backend', 'frontend', 'qa', 'release', 'devops', 'design'));

-- 3. 新增索引
CREATE INDEX IF NOT EXISTS idx_agent_role_specialization ON agent(role, specialization);
CREATE INDEX IF NOT EXISTS idx_agent_status_role ON agent(status, role);

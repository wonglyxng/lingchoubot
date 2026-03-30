# WP2-04: Agent 能力模型扩展与专业化执行 Agent

## 变更摘要

本次实现了 Agent 能力模型的扩展，为 Agent 增加了类型（agent_type）和专业化方向（specialization）两个维度，并基于此实现了任务到专业化 Agent 的智能路由。新增 3 个专业化 mock worker（backend/frontend/qa），Registry 升级为组合键路由，Engine 新增关键词推断 + 专业化路由逻辑。

## 文件清单

### 新增文件
| 文件 | 说明 |
|------|------|
| `backend/migrations/000008_agent_capability.up.sql` | 迁移：agent 表增加 agent_type 和 specialization 列 |
| `backend/migrations/000008_agent_capability.down.sql` | 回滚迁移 |
| `backend/internal/runtime/mock_backend_worker.go` | Backend 专业化 mock worker，生成 Go handler 代码 |
| `backend/internal/runtime/mock_frontend_worker.go` | Frontend 专业化 mock worker，生成 React/TypeScript 组件 |
| `backend/internal/runtime/mock_qa_worker.go` | QA 专业化 mock worker，生成测试报告 |
| `backend/internal/model/agent_test.go` | AgentType/AgentSpecialization 常量与匹配逻辑测试 |
| `backend/internal/runtime/registry_test.go` | Registry 专业化路由与回退机制测试 |
| `backend/internal/orchestrator/engine_test.go` | inferSpecialization 关键词推断测试 |

### 修改文件
| 文件 | 说明 |
|------|------|
| `backend/internal/model/agent.go` | 新增 AgentType、AgentSpecialization 类型常量与 MatchesSpecialization 方法 |
| `backend/internal/repository/agent.go` | 所有查询适配新字段，新增 FindByRoleAndSpec 方法 |
| `backend/internal/service/agent.go` | 创建时默认值处理，新增 FindByRoleAndSpec |
| `backend/internal/runtime/protocol.go` | 新增 SpecializedRunner 接口 |
| `backend/internal/runtime/registry.go` | 升级为 role:specialization 组合键路由，支持回退 |
| `backend/internal/orchestrator/engine.go` | 新增 inferSpecialization、findAgentWithSpec、专业化路由逻辑 |

## 关键实现说明

### 1. Agent 能力模型扩展

- **AgentType**（mock / llm / human）：区分 Agent 的执行引擎类型，为后续接入 LLM Agent 做准备
- **AgentSpecialization**（general / backend / frontend / qa / release / devops / design）：Agent 的专业化方向
- **MatchesSpecialization** 方法：general 可匹配任何方向，实现通用/专业双层 Agent 体系

### 2. Registry 组合键路由

- runners 映射键从 `role` 升级为 `role:specialization` 格式
- `GetForSpec(role, spec)` 先查精确键 `role:spec`，未命中则回退到基础键 `role`
- `Get(role)` 保持向后兼容

### 3. 任务专业化推断

`inferSpecialization(task)` 通过任务标题和描述的关键词匹配确定专业方向：
- backend: API、接口、后端、数据库、handler、repository、service
- frontend: 前端、页面、组件、UI、React、Next.js
- qa: 测试、验证、QA、test
- release: 发布、部署
- devops: CI/CD、Docker、Kubernetes

### 4. 专业化 Agent 路由

Engine 的 `runWorker()` 流程：
1. `inferSpecialization(task)` → 推断专业方向
2. `findAgentWithSpec(ctx, "worker", spec)` → 数据库查找匹配的 Agent（精确匹配优先，general 回退）
3. `registry.GetForSpec("worker", spec)` → 获取对应的执行 Runner
4. Step 名称包含专业化标识

### 5. 专业化 Mock Worker 产出

| Worker | 产出类型 | 内容类型 | 说明 |
|--------|---------|----------|------|
| Backend | source_code | text/x-go | Go HTTP handler 代码 |
| Frontend | source_code | text/typescript | React 组件代码 |
| QA | test_report | text/markdown | 测试报告（含测试用例表格） |

## 运行方式

```bash
# 运行数据库迁移（包含 000008）
docker-compose up migrate

# 启动 API 服务
docker-compose up api

# 或本地开发
cd backend && go run cmd/api/main.go
```

## 测试方式

```bash
cd backend

# 运行所有测试
go test ./...

# 运行新增测试（详细输出）
go test -v ./internal/model/ -run TestAgentType
go test -v ./internal/model/ -run TestMatchesSpecialization
go test -v ./internal/runtime/ -run TestRegistryGetForSpec
go test -v ./internal/runtime/ -run TestRegistrySpecializedRunnerOutput
go test -v ./internal/orchestrator/ -run TestInferSpecialization
```

## 已知限制

1. **关键词推断为静态规则**：inferSpecialization 基于硬编码关键词列表，后续可替换为 LLM 推断或任务标签
2. **专业化方向固定**：CHECK 约束限定了 7 个方向，新增方向需要走 migration
3. **Agent 数据未预置**：000008 仅加列，未 INSERT 专业化 Agent 数据，需要通过 API 或后续 migration 创建
4. **仅 worker 角色使用专业化路由**：pm/supervisor/reviewer 暂未走专业化逻辑
5. **Mock Worker 产出为固定模板**：实际代码生成需接入 LLM Agent 后替换

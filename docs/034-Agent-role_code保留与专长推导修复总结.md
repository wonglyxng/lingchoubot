# 034 - Agent role_code 保留与专长推导修复总结

## 变更摘要

修复 Agent 编辑时可能出现的 role_code 冲突问题。问题根因是前端更新请求未携带 role_code，后端在 role_code 为空时又仅按 role 进行默认推导，导致前端 Worker 在编辑后被错误回填为 BACKEND_DEV_WORKER，进而触发唯一约束冲突。

本次修复从前后端两侧同时收口：
- 后端更新 Agent 时，若请求未显式传入 role_code，则保留数据库中的既有值，避免编辑动作误改组织职责码。
- 后端默认 role_code 推导改为同时参考 role 和 specialization，前端、QA、QA 主管等场景不再统一落到后端默认编码。
- 前端 Agent 表单在创建和编辑时显式携带 role_code，并在角色或专长变化时同步推导对应职责码。

## 文件清单

### 修改文件
| 文件 | 说明 |
|------|------|
| `backend/internal/service/agent.go` | 更新时保留已有 role_code；默认 role_code 推导改为基于 role + specialization |
| `backend/internal/handler/agent_test.go` | 新增「更新时省略 role_code 仍保留原值」回归测试 |
| `frontend/src/app/agents/page.tsx` | Agent 表单新增隐藏的 role_code 状态，并在 role/spec 变化时同步推导 |

### 新增文件
| 文件 | 说明 |
|------|------|
| `backend/internal/service/agent_role_code_test.go` | 新增服务层测试，覆盖专长驱动默认编码与更新保留编码 |

## 关键实现说明

### 1. 后端更新语义修正
- `AgentService.Update` 在加载旧数据后，若本次请求未传 `role_code`，则先回填旧值，再进入统一的写入标准化流程。
- 这样可以兼容当前前端表单未显式展示 role_code 的交互方式，避免一次普通编辑把 Agent 的职责码错误重置。

### 2. 默认 role_code 推导规则细化
- PM -> `PM_SUPERVISOR`
- Supervisor + QA -> `QA_SUPERVISOR`
- Supervisor + 其他 -> `DEVELOPMENT_SUPERVISOR`
- Worker + Frontend -> `FRONTEND_DEV_WORKER`
- Worker + QA -> `QA_WORKER`
- Worker + 其他 -> `BACKEND_DEV_WORKER`
- Reviewer -> `REVIEWER_WORKER`

### 3. 前端提交链路补全
- 创建和编辑 Agent 时，payload 现在会显式带上 `role_code`。
- 当用户修改「角色」或「专长」时，前端会同步更新待提交的 role_code，保证前后端推导规则一致。

## 运行方式

```bash
# 1. 启动后端
cd backend && go run cmd/api/main.go

# 2. 启动前端
cd frontend && npm run dev

# 3. 进入 Agent 页面验证编辑
http://localhost:3000/agents
```

## 测试方式

```bash
# 后端全量测试
cd backend && go test ./... -count=1 -short

# 前端构建验证
cd frontend && npm run build
```

## 已知限制

1. 当前前端仍未直接暴露 role_code 编辑入口，用户只能通过角色与专长的组合来间接决定默认职责码。
2. 若后续需要支持自定义 role_code，建议为 Agent 表单增加显式字段，并在后端区分“自动推导”和“手工指定”两种模式。
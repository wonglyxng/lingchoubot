# 018 - 真实 LLM Agent 接入与 Temporal 适配层实现总结

## 1. 变更摘要

本次实现了两个核心能力：

1. **真实 LLM Agent 接入**：系统可通过 OpenAI 兼容 API 调用真实大语言模型驱动 PM / Supervisor / Worker / Reviewer 角色，替代 Mock Agent。
2. **Temporal 适配层**：引入 WorkflowEngine 接口抽象，实现了基于 Temporal 的工作流引擎，每个编排步骤（PM / Supervisor / Worker / Reviewer）作为独立 Activity 执行，支持崩溃恢复和自动重试。

两者均为**可选启用**，通过环境变量控制，默认行为与原有系统完全一致（Mock Agent + 本地 goroutine 编排）。

## 2. 文件清单

### 新增文件（9 个）

| 文件 | 说明 |
|------|------|
| `backend/internal/runtime/llm_client.go` | OpenAI 兼容 HTTP 客户端，支持 JSON 模式返回 |
| `backend/internal/runtime/llm_prompts.go` | 各角色系统提示词模板（PM / Supervisor / Worker / Reviewer） |
| `backend/internal/runtime/llm_runner.go` | LLM 驱动的 AgentRunner 实现 + RegisterLLMRunners 注册器 |
| `backend/internal/orchestrator/workflow_engine.go` | WorkflowEngine 接口定义（RunAsync / GetRun / ListRuns） |
| `backend/internal/orchestrator/temporal_engine.go` | Temporal 工作流引擎实现 |
| `backend/internal/orchestrator/temporal_workflow.go` | Temporal 工作流定义与活动输入类型 |
| `backend/internal/orchestrator/temporal_activities.go` | Temporal 活动实现（PM / Supervisor / Worker / Reviewer / 辅助） |
| `backend/internal/orchestrator/temporal_worker.go` | Temporal Worker 启动与 slog 日志适配器 |
| `docs/018-真实LLM接入与Temporal适配层实现总结.md` | 本文档 |

### 修改文件（4 个）

| 文件 | 说明 |
|------|------|
| `backend/internal/config/config.go` | 新增 LLMConfig 和 TemporalConfig 结构体及环境变量加载 |
| `backend/internal/handler/orchestrator.go` | Handler 改用 WorkflowEngine 接口而非 *Engine 具体类型 |
| `backend/cmd/api/main.go` | 条件注册 LLM Runner、条件选择 Temporal/本地引擎 |
| `docker-compose.yml` | 添加 LLM/Temporal 环境变量；新增 temporal + temporal-ui 服务（profile 隔离） |

### 依赖变更

| 依赖 | 版本 |
|------|------|
| `go.temporal.io/sdk` | v1.41.1 |

## 3. 关键实现说明

### 3.1 LLM Agent 架构

```
LLMClientConfig → LLMClient → LLMAgentRunner (implements AgentRunner)
                                    ↓
                            buildSystemPrompt(role, spec)
                            buildUserPrompt(input)
                                    ↓
                            OpenAI /chat/completions (JSON mode)
                                    ↓
                            json.Unmarshal → AgentTaskOutput
```

- **兼容性**：使用 OpenAI Chat Completions API 格式，兼容 OpenAI / DeepSeek / Azure / vLLM / Ollama 等
- **JSON 强制**：通过 `response_format: {"type": "json_object"}` 确保 LLM 返回有效 JSON
- **优雅降级**：LLM 调用失败或 JSON 解析失败时返回 `OutputStatusFailed`，不抛 Go error，编排引擎可继续处理
- **提示词设计**：每个角色有专用系统提示词，明确定义期望的 JSON 输出结构；Worker 根据专长领域（backend/frontend/qa）使用不同提示

### 3.2 Temporal 适配层架构

```
WorkflowEngine (interface)
├── Engine (本地 goroutine，默认)
└── TemporalEngine (Temporal client)

Temporal Workflow: ProjectWorkflow
├── ActivityPM          → 项目分解
├── ActivityListPhaseTasks → 获取阶段任务列表
├── ActivitySupervisor  → 任务规划与契约
├── ActivityWorker      → 任务执行与工件生成
├── ActivityReviewer    → 独立评审
├── ActivityCompleteRun → 完成标记
└── ActivityFailRun     → 失败标记
```

- **接口抽象**：`WorkflowEngine` 接口（RunAsync / GetRun / ListRuns），Engine 和 TemporalEngine 均实现
- **Activity 粒度**：以角色步骤为粒度划分 Activity，崩溃后从当前 Activity 重试而非整个工作流重来
- **重试策略**：每个 Activity 最多重试 3 次，指数退避（1s → 2s → 4s...），最大间隔 1 分钟
- **代码复用**：Activity 通过构造临时 Engine 实例调用已有 processXxxActions 方法，避免重复代码
- **Profile 隔离**：Temporal 服务使用 Docker Compose profile，不影响默认启动流程

### 3.3 配置模型

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `LLM_ENABLED` | `false` | 启用 LLM Agent |
| `LLM_BASE_URL` | `https://api.openai.com/v1` | LLM API 基础 URL |
| `LLM_API_KEY` | （空） | LLM API 密钥 |
| `LLM_MODEL` | `gpt-4o-mini` | 模型名称 |
| `TEMPORAL_ENABLED` | `false` | 启用 Temporal 编排 |
| `TEMPORAL_HOST_PORT` | `localhost:7233` | Temporal gRPC 地址 |
| `TEMPORAL_NAMESPACE` | `default` | Temporal 命名空间 |
| `TEMPORAL_TASK_QUEUE` | `lingchou-orchestrator` | Temporal 任务队列 |

## 4. 运行方式

### 默认模式（Mock Agent + 本地编排）

```bash
docker compose up -d
# 与之前完全相同，无需任何额外配置
```

### 启用 LLM Agent

```bash
export LLM_ENABLED=true
export LLM_API_KEY=sk-your-key-here
export LLM_MODEL=gpt-4o-mini      # 或 deepseek-chat 等
export LLM_BASE_URL=https://api.openai.com/v1  # 或其他兼容端点
docker compose up -d
```

### 启用 Temporal 编排

```bash
# 先启动 Temporal 基础设施
docker compose --profile temporal up -d

# 再启动带 Temporal 的 API 服务
export TEMPORAL_ENABLED=true
docker compose up -d api
```

### LLM + Temporal 同时启用

```bash
export LLM_ENABLED=true
export LLM_API_KEY=sk-xxx
export TEMPORAL_ENABLED=true
docker compose --profile temporal up -d
```

## 5. 测试方式

### 编译验证

```bash
cd backend
go build ./...
```

### 单元测试

```bash
cd backend
go test ./internal/... -count=1
```

### LLM 集成测试（手动）

```bash
# 启动带 LLM 的服务后
curl -X POST http://localhost:8080/api/v1/orchestrator/runs \
  -H "Content-Type: application/json" \
  -d '{"project_id": "<your-project-id>"}'

# 轮询查看运行状态
curl http://localhost:8080/api/v1/orchestrator/runs/<run-id>
```

### Temporal 集成测试（手动）

```bash
# 启动 Temporal 后，访问 Temporal UI 查看工作流执行
open http://localhost:8088
```

## 6. 已知限制

1. **LLM 提示词需迭代**：当前提示词是初版，实际接入 LLM 后可能需要根据具体模型的输出特点调整
2. **无自动集成测试**：LLM 和 Temporal 的集成测试需要外部依赖，当前仅验证编译和已有单元测试通过
3. **Activity 无 Workflow Step 持久化**：Temporal 模式下 Activities 直接操作业务数据，但未像本地 Engine 那样写入 WorkflowStep 记录（后续迭代可补充）
4. **Temporal DB 共用 PostgreSQL**：auto-setup 镜像的 Temporal 后端使用同一个 PostgreSQL 实例，生产环境建议分离
5. **Token 预算管控**：当前未对 LLM 调用做 token 成本追踪和限制
6. **工作流重入**：Temporal 工作流定义目前不支持部分重入（如只重跑失败的 task chain）

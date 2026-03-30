# WP2-08 Temporal 编排收口与恢复执行 — 实现总结

## 1. 变更摘要

本次实现 WP2-08 工作包，目标是让 Temporal 编排路径与本地 Engine 达到功能对齐，主要包含以下能力：

1. **Activities 步骤持久化** — 所有 Temporal Activity 均写入 `workflow_step` 记录，实现可审计的步骤追踪
2. **域路由对齐** — Temporal Activities 对齐 WP2-07 的组织角色矩阵路由逻辑（`inferExecutionDomain` + `findSupervisorByDomain`）
3. **返工循环** — `ProjectWorkflow` 新增最多 3 次返工循环（Reviewer → CheckRework → Worker → Reviewer）
4. **取消能力** — `CancelRun` 接口贯穿 interface → service → engine → handler → migration 全链路
5. **Worker 进程分离** — 新增 `cmd/worker/main.go` 独立进程，API 进程可选仅 dial client 而不启动 worker
6. **Cancel API 端点** — `POST /api/v1/orchestrator/runs/{id}/cancel`

## 2. 文件清单

### 新增文件
| 文件 | 说明 |
|------|------|
| `backend/cmd/worker/main.go` | 独立 Temporal Worker 进程入口 |
| `backend/migrations/000011_workflow_cancel.up.sql` | 添加 `cancelled` 状态到 workflow_run CHECK 约束 |
| `backend/migrations/000011_workflow_cancel.down.sql` | 回滚 migration |

### 修改文件
| 文件 | 说明 |
|------|------|
| `backend/internal/orchestrator/temporal_activities.go` | 新增 `stepTracker` 辅助结构，所有 Activity 写 workflow_step，域路由对齐，新增 `ActivityCheckRework` |
| `backend/internal/orchestrator/temporal_workflow.go` | `ProjectWorkflow` 添加步骤计数、返工循环（最多 3 次），新增 `CheckReworkInput`/`TaskChainInput.SortOffset` |
| `backend/internal/orchestrator/temporal_worker.go` | 注册 `ActivityCheckRework`，新增 `DialTemporal` 函数 |
| `backend/internal/orchestrator/temporal_engine.go` | 实现 `CancelRun`（Temporal cancel + DB 更新） |
| `backend/internal/orchestrator/workflow_engine.go` | `WorkflowEngine` 接口新增 `CancelRun` 方法 |
| `backend/internal/orchestrator/engine.go` | 本地 Engine 实现 `CancelRun`（仅 DB） |
| `backend/internal/orchestrator/engine_test.go` | 新增 4 个测试用例 |
| `backend/internal/model/workflow.go` | 新增 `WorkflowRunCancelled` 状态常量 |
| `backend/internal/service/workflow.go` | 新增 `CancelRun` 业务方法 |
| `backend/internal/config/config.go` | `TemporalConfig` 新增 `WorkerEmbedded` 字段 |
| `backend/cmd/api/main.go` | 条件启动嵌入式/外置 worker |
| `backend/internal/handler/orchestrator.go` | 新增 cancel 路由与 handler |
| `backend/internal/handler/orchestrator_test.go` | 新增 2 个 cancel handler 测试 |

## 3. 关键实现说明

### 3.1 stepTracker — 步骤生命周期管理

```go
type stepTracker struct {
    svc    *service.WorkflowService
    runID  string
    stepID string
    sort   int
}
```

每个 Activity 在入口处调用 `newStep(name, agentType)`，自动创建 `workflow_step` 记录（状态 running）。
执行成功后调用 `complete(output)`，失败调用 `fail(err)`。
辅助方法 `setAgent/setTask/setPhase` 用于关联上下文实体。

### 3.2 域路由对齐

`ActivitySupervisor` 中使用 `inferExecutionDomain(task.Title)` 推断任务域，
然后通过 `findSupervisorByDomain` 查找对应域的 Supervisor Agent，
与本地 Engine 的 WP2-07 逻辑保持一致。

### 3.3 返工循环

```
ProjectWorkflow:
  PM → for each task:
    Supervisor → Worker → Reviewer → CheckRework?
                 ↑_________yes (max 3)___|
```

`ActivityCheckRework` 检查任务状态：
- `revision_required` → 返回 true，工作流回到 Worker
- 其他状态 → 返回 false，继续下一任务
- 超过 3 次 → 强制跳出

### 3.4 CancelRun 全链路

```
Handler.CancelRun
  → Engine.CancelRun (interface)
    → TemporalEngine: client.CancelWorkflow + DB update
    → LocalEngine: DB update only
  → WorkflowService.CancelRun: set status=cancelled, error="cancelled by user"
```

TemporalEngine 的 CancelRun 具备容错：即使 Temporal cancel 失败，仍会尝试更新 DB。

### 3.5 Worker 进程分离

- `TEMPORAL_WORKER_EMBEDDED=true`（默认）：API 进程内嵌 worker（开发模式）
- `TEMPORAL_WORKER_EMBEDDED=false`：API 只 dial Temporal 作为 client，worker 通过 `cmd/worker/main.go` 独立运行
- `DialTemporal(cfg)` 只建立 client 连接，不启动 worker

### 3.6 Migration 000011

```sql
ALTER TABLE workflow_run
  DROP CONSTRAINT workflow_run_status_check,
  ADD CONSTRAINT workflow_run_status_check
    CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled'));
```

## 4. 运行方式

### 开发模式（嵌入式 worker）
```bash
# 默认 TEMPORAL_WORKER_EMBEDDED=true
docker-compose up -d postgres
cd backend && go run cmd/api/main.go
```

### 生产模式（独立 worker）
```bash
docker-compose --profile temporal up -d

# API 进程
TEMPORAL_ENABLED=true TEMPORAL_WORKER_EMBEDDED=false go run cmd/api/main.go

# Worker 进程（单独终端）
TEMPORAL_ENABLED=true go run cmd/worker/main.go
```

### 执行 migration
```bash
go run cmd/migrate/main.go
```

### 取消运行
```bash
curl -X POST http://localhost:8080/api/v1/orchestrator/runs/{runID}/cancel
```

## 5. 测试方式

```bash
cd backend
go test ./... -count=1
```

新增测试用例：
- `TestWorkflowRunStatusConstants` — 验证 5 种状态常量（含 cancelled）
- `TestWorkflowStepStatusConstants` — 验证 5 种步骤状态常量
- `TestCheckReworkInputType` — 验证 CheckReworkInput 类型字段
- `TestTaskChainInputSortOffset` — 验证 SortOffset 字段传递
- `TestOrchestratorHandler_CancelRun_Success` — cancel handler 成功路径
- `TestOrchestratorHandler_CancelRun_Error` — cancel handler 错误路径

## 6. 已知限制

1. **审批等待点未实现** — Temporal workflow 暂未接入 Signal 等待审批决策，需后续 WP 补充
2. **Retry/Timeout 策略为默认** — Activity 未配置自定义 RetryPolicy 和超时，使用 Temporal 默认值
3. **返工循环硬编码 3 次** — 最大返工次数写死在 workflow 中，未做成配置项
4. **stepTracker 依赖 WorkflowService** — 如果 service 调用失败，步骤记录可能不完整（但不影响主流程）
5. **独立 worker 进程未容器化** — `cmd/worker/main.go` 暂无独立 Dockerfile
6. **Temporal 集成测试需要真实 Temporal Server** — 当前测试为单元测试，不含 Temporal e2e 测试

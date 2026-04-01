# 030 基础 Agent 启动自举实现总结

## 1. 变更摘要

本次实现了基础 Agent 的启动自举能力，后端 API 服务启动时会自动检查并补齐一组 MVP 最小可运行角色，避免每次手动创建 PM、主管、执行与评审 Agent。

本次改动目标是：

- 让新环境启动后立即具备可运行的最小组织树
- 保证重复启动不会重复造角色
- 让测试夹具与生产启动共用同一套基线 Agent 模板，避免角色定义漂移

## 2. 文件清单

新增文件：

- `backend/internal/service/agent_bootstrap.go`
- `backend/internal/service/agent_bootstrap_test.go`
- `docs/030-基础Agent启动自举实现总结.md`

修改文件：

- `backend/cmd/api/main.go`
- `backend/internal/testutil/testutil.go`
- `docs/002-灵筹 Agent 组织树与职责说明.md`

## 3. 关键实现说明

### 3.1 统一维护基线 Agent 模板

在 `backend/internal/service/agent_bootstrap.go` 中新增 `BaselineAgentSpecs()`，集中定义系统默认基线角色：

- PM Agent
- Development Supervisor
- QA Supervisor
- Backend Worker
- Frontend Worker
- QA Worker
- Reviewer Agent

该模板同时包含：

- `role`
- `role_code`
- `specialization`
- `managed_roles`
- `allowed_tools`
- `reports_to` 所依赖的父级 `role_code`

这样生产启动和测试夹具都可以复用同一份角色基线，不再各自维护硬编码列表。

### 3.2 启动时幂等补齐缺失角色

新增 `AgentService.EnsureBaselineAgents(ctx)`：

- 先读取已有 Agent 列表
- 以 `role_code` 为幂等判断键
- 已存在则跳过
- 不存在则创建
- 创建子角色时自动挂接父级 `reports_to`

这样即使服务多次重启，也只会补齐缺失项，不会重复创建整套基线 Agent。

### 3.3 API 启动链路接入自举

在 `backend/cmd/api/main.go` 中，服务初始化完成后会调用 `agentSvc.EnsureBaselineAgents(...)`：

- 成功且有新增时输出创建数量日志
- 失败时记录错误日志
- 当前策略是不阻断 API 启动，避免单次自举异常直接影响服务可用性

### 3.4 测试夹具复用同一套基线模板

`backend/internal/testutil/testutil.go` 中的 `SeedStandardAgents` 已改为复用 `BaselineAgentSpecs()`，不再维护另一套独立 Agent 列表。

这样可以保证：

- 测试环境与生产环境角色定义一致
- 后续新增或调整基线角色时只需改一处

### 3.5 覆盖幂等性与层级关系测试

新增 `backend/internal/service/agent_bootstrap_test.go`，覆盖以下场景：

- 首次调用会创建完整基线角色集
- 重复调用不会重复创建
- 已存在部分角色时仅补齐缺失项
- Development Supervisor / QA Supervisor 正确挂到 PM 下
- Reviewer Agent 正确挂到 QA Supervisor 下

## 4. 运行方式

后端启动后会自动执行基础 Agent 自举，无需额外手工脚本。

本地启动示例：

```bash
cd backend
go run ./cmd/api
```

或使用 Docker Compose：

```bash
docker compose up --build
```

启动完成后，可在 Agent 列表页或通过 Agent API 查询，确认系统已自动补齐基础角色。

## 5. 测试方式

本次已执行：

```bash
cd backend
go test ./...
go build ./cmd/api
```

验证点：

1. 后端所有测试通过
2. API 启动入口编译通过
3. 新增自举单测覆盖首次创建、幂等重复调用、部分补齐三类场景

## 6. 已知限制

- 当前幂等判断基于应用层的 `role_code` 扫描，数据库层尚未对 `agent.role_code` 建唯一约束。
- 启动自举只补齐 MVP 最小角色集，不会自动创建更细分的产品、架构、发布等扩展角色。
- 当前策略在自举失败时只记录日志、不阻断启动；如果未来要求更强的一致性，可再评估是否改为启动失败即退出。
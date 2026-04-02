# 039-运行时去Mock收口与严格LLM验收总结

## 1. 变更摘要

本次继续收紧真实 LLM 执行链路，完成了运行时 Mock Runner 清理、测试桩迁移、Agent 类型约束收口，以及 Demo 脚本对最新审批语义的适配。

同时对 Docker Compose 运行态做了真实验收：当前 API 在严格模式下仅暴露 `artifact_storage`，工作流在缺失真实 LLM API Key 时会直接返回上游错误，不再降级到 Mock 或产生意料之外的工件。

## 2. 文件清单

新增文件：

- `backend/internal/testutil/runtime_stub_runners.go`
- `backend/migrations/000017_agent_type_strict_llm.up.sql`
- `backend/migrations/000017_agent_type_strict_llm.down.sql`
- `docs/039-运行时去Mock收口与严格LLM验收总结.md`

修改文件：

- `backend/cmd/demo/main.go`
- `backend/internal/handler/agent_test.go`
- `backend/internal/model/agent.go`
- `backend/internal/model/agent_test.go`
- `backend/internal/runtime/llm_eval_samples.go`
- `backend/internal/runtime/llm_prompts_test.go`
- `backend/internal/runtime/llm_runner.go`
- `backend/internal/runtime/llm_runner_test.go`
- `backend/internal/runtime/protocol.go`
- `backend/internal/runtime/registry.go`
- `backend/internal/runtime/registry_test.go`
- `backend/internal/service/agent.go`
- `backend/internal/service/agent_bootstrap_test.go`
- `backend/internal/testutil/testutil.go`
- `docs/004-数据库设计.md`
- `docs/005-API 接口文档.md`
- `docs/038-默认工具集与Demo严格模式收口总结.md`

删除文件：

- `backend/internal/runtime/mock_backend_worker.go`
- `backend/internal/runtime/mock_frontend_worker.go`
- `backend/internal/runtime/mock_pm.go`
- `backend/internal/runtime/mock_qa_worker.go`
- `backend/internal/runtime/mock_reviewer.go`
- `backend/internal/runtime/mock_supervisor.go`
- `backend/internal/runtime/mock_worker.go`

## 3. 关键实现说明

1. 生产运行时不再保留任何内建 Mock Runner。

`backend/internal/runtime` 中原有的 PM、Supervisor、Worker、Reviewer 及其专长 Mock 实现已全部删除，`Registry.RegisterDefaults()` 也被移除，避免任何生产链路通过默认注册拿到伪造执行器。

2. 测试改为使用 `backend/internal/testutil` 下的确定性测试桩。

原本依赖生产包内 Mock Runner 的集成测试，改为通过 `registerDeterministicTestRunners` 注册测试专用执行器。这样既保留测试可重复性，也避免把假执行逻辑继续留在生产运行时目录中。

3. LLM Runner 兼容兜底接口已删除。

`LLMAgentRunner` 移除了 fallback 字段、`WithFallback` 以及 `RegisterLLMRunnersWithFallback` 兼容包装。现在 LLM 调用、解析或校验失败时会直接向上返回失败结果，不再偷偷切到其他执行器。

4. Agent 类型语义已收紧到 `llm` / `human`。

代码层删除了 `AgentTypeMock`，服务层写入校验不再接受 `mock`。数据库通过 `000017_agent_type_strict_llm` migration 将历史 `mock` 数据迁移为 `llm`，并把约束与默认值同步收紧。

5. Demo 脚本已对齐当前审批状态机。

`backend/cmd/demo/main.go` 不再手工额外创建审批请求，而是消费工作流自然产生的待审批项；同时把 `waiting_approval` 视为第一阶段工作流的合法暂停态，并用 `pending_approval` 语义展示批准/拒绝后的状态流转。

6. Live Compose 验证结果符合严格模式要求。

重启后的 API 已成功启动真实 LLM Runner，`GET /api/v1/tools` 仅返回 `artifact_storage`。随后端到端 Demo 在 PM 第一步真实调用 LLM 时收到 401 API key 缺失错误并立即失败，说明当前链路按预期“失败即上报”，没有任何 Mock 降级或伪造产物。

## 4. 运行方式

1. 在仓库根目录准备 `.env`，至少保证 `LLM_ENABLED=true`。
2. 为真实 LLM 提供可用配置，例如填写 `LLM_API_KEY`，或按 Provider 维度填写对应的 API Key。
3. 启动服务：`docker compose up --build -d migrate api`
4. 如需运行 Demo：进入 `backend/` 后执行 `go run ./cmd/demo -url http://localhost:18080`

## 5. 测试方式

1. 后端单元与集成测试：在 `backend/` 目录执行 `go test ./...`
2. 运行态健康检查：访问 `http://localhost:18080/healthz` 与 `http://localhost:18080/readyz`
3. 默认工具集检查：访问 `http://localhost:18080/api/v1/tools`，预期仅出现 `artifact_storage`
4. 真实链路验收：执行 Demo，若 LLM 配置有效则继续完成工作流；若配置缺失，应直接收到上游 LLM 报错而不是进入 Mock 成功路径

## 6. 已知限制

1. 本次未删除 fail-closed 的兼容工具类型定义；它们已不在默认工具集对外暴露，但类型仍保留用于兼容已有代码路径。
2. 当前本地 `.env` 中 `LLM_API_KEY` 仍为空，因此 live Demo 只能验证“严格失败”路径，无法在本机完成真实供应商下的成功验收。
3. 本次未修改前端页面或控制台交互，变更范围集中在后端运行时、测试与文档收口。
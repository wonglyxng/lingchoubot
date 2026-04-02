# 037-严格真实LLM链路与无Mock降级实现总结

## 1. 变更摘要

本次改动围绕“工作流链路必须只走真实 LLM、失败直接上抛、不能再产出 Mock/兜底结果”这一目标展开，修复了此前出现“计算器项目的可行性评估却产出 QA 测试报告并被放行”的根因。

本次实现后，主工作流链路具备以下行为：

- API 进程和 Temporal worker 在 `LLM_ENABLED=false` 时直接拒绝启动，不再注册主链路 Mock Runner。
- 工作流启动前会严格校验关键 baseline Agent 是否存在、是否激活、是否为 LLM Agent、是否带有有效 provider/model 配置。
- 任务分派、主管查找、专长 Worker 选择不再做任意兜底，缺角色或缺专长时直接失败。
- LLM 调用失败、输出 JSON 解析失败、结构/语义校验失败时直接返回失败，不再降级到 Mock 输出。
- 工件内容写入对象存储失败时直接报错，不再生成 `mock://` 伪 URI。
- reviewer 不再只按结构放行，而是会结合任务目标、工件类型和工件内容判断是否相关、是否真实、是否可批准。

## 2. 文件清单

### 启动与配置

- `backend/cmd/api/main.go`
- `backend/cmd/worker/main.go`
- `backend/internal/config/config.go`
- `backend/internal/config/config_test.go`
- `.env.example`
- `docker-compose.yml`

### Agent 模型与基线拓扑

- `backend/internal/model/agent.go`
- `backend/internal/service/agent.go`
- `backend/internal/service/agent_bootstrap.go`
- `backend/internal/service/agent_bootstrap_test.go`
- `backend/internal/service/agent_role_code_test.go`

### 编排与运行态

- `backend/internal/orchestrator/precheck.go`
- `backend/internal/orchestrator/agent_runtime.go`
- `backend/internal/orchestrator/engine.go`
- `backend/internal/orchestrator/engine_test.go`
- `backend/internal/orchestrator/temporal_activities.go`
- `backend/internal/runtime/registry.go`
- `backend/internal/runtime/registry_test.go`
- `backend/internal/runtime/llm_runner.go`
- `backend/internal/runtime/llm_runner_test.go`
- `backend/internal/runtime/llm_prompts.go`
- `backend/internal/runtime/llm_validator.go`
- `backend/internal/runtime/llm_validator_test.go`

### 工件持久化与服务层

- `backend/internal/gateway/artifact_storage.go`
- `backend/internal/gateway/artifact_storage_test.go`
- `backend/internal/gateway/gateway.go`
- `backend/internal/service/artifact.go`
- `backend/internal/service/repo_interfaces.go`
- `backend/internal/service/service_test.go`
- `backend/internal/repository/artifact.go`
- `backend/internal/testutil/testutil.go`

### 文档

- `docs/037-严格真实LLM链路与无Mock降级实现总结.md`

## 3. 关键实现说明

### 3.1 启动阶段强制真实 LLM

- API 和 worker 启动时新增显式检查：`LLM_ENABLED` 不是 `true` 就退出。
- 原本“LLM 关掉时注册默认 Mock Runner”的路径已从主链路移除。
- `LLM_FALLBACK_ENABLED` 仅保留为兼容旧环境变量的废弃字段，不再影响运行行为。

这样处理的原因是：如果启动时就允许进入 Mock 模式，后续任何看似“成功”的任务都可能只是模板产物，无法区分真实执行与兜底执行。

### 3.2 基线角色和路由改为严格模式

- 新增 `GENERAL_WORKER` 角色码，用于承接需求梳理、可行性评估、方案分析等通用分析类任务。
- baseline Agent 拓扑补齐 General Worker，并挂到 Development Supervisor 下。
- 工作流 precheck 从“只检查 PM / Supervisor / Worker / Reviewer 四大角色是否存在”升级为“逐个检查关键 role_code 是否存在且配置有效”。
- 专长 Runner 查找、Supervisor 查找、Worker 选择不再回退到任意同角色 Agent。

这样处理的原因是：此前“找不到精确执行者时退回任意 Worker”的策略会把分析任务误路由到 QA Worker，从而输出完全不相关的测试报告。

### 3.3 LLM 输出校验从结构校验升级为任务相关性校验

- PM prompt 要求阶段和任务必须贴合真实项目目标，且分析阶段明确包含需求梳理和可行性评估。
- Worker prompt 使用真实 `artifact_type` 枚举，并强制分析任务输出 `prd` 或 `design`，测试任务输出 `test_report`。
- Reviewer prompt 取消“首次提交倾向批准”，改为严格基于任务、工件类型、工件内容判断。
- validator 在原有 schema 校验之外新增输入相关校验：
  - 分析任务不能产出 `test_report` 或 `source_code` 作为主工件。
  - 工件正文必须非空，且要能绑定当前任务或项目。
  - 包含 `todo`、`待补充`、`mock ... agent` 等占位/伪造痕迹的内容会被拒绝。
  - reviewer 不能批准占位工件，也不能把纯测试报告当作分析任务结果批准。

这样处理的原因是：只做 JSON 结构校验，会让“格式正确但内容明显错误”的结果混过链路。

### 3.4 工件落盘改为失败即报错

- `ArtifactStorageTool` 初始化失败时会记录真实错误，并在执行期返回 `artifact storage unavailable`。
- 删除 `mock://minio/...` 伪 URI 生成逻辑。
- `ArtifactService` 新增 `CreateWithInitialVersion`，用于把工件和首版本作为一个逻辑单元创建；若版本创建失败，会回滚工件记录。

这样处理的原因是：伪造 URI 会让后续评审、审批和审计都基于不存在的对象继续推进，结果不可控。

### 3.5 编排步骤从“记录错误”改为“中断工作流”

- contract、assignment、artifact、handoff、review、transition 处理函数全部改为返回错误。
- 本地编排和 Temporal 活动都改为在这些处理失败时直接 fail 当前 step，而不是只写日志继续跑。

这样处理的原因是：主链路一旦接受局部失败继续推进，就会继续产生不完整或语义错误的产物。

## 4. 假设

- 运行环境会提供真实可用的 LLM 配置，包括 `LLM_BASE_URL`、`LLM_API_KEY`、`LLM_MODEL`，以及必要时的 provider 级配置。
- baseline Agent 启动自举流程会被执行，确保关键 role_code 对应 Agent 已创建并处于 active 状态。
- MinIO 或对应对象存储在需要产出工件时可访问；若不可访问，当前实现会显式失败而不是兜底。

## 5. 运行方式

在本地联调时，至少需要满足以下前提：

1. 设置 `LLM_ENABLED=true`。
2. 提供真实的 LLM base URL、API key 和 model。
3. 如需完整工件链路，确保 MinIO 已启动且配置正确。

后端测试或本地验证可在 `backend` 目录执行：

```bash
go test ./...
```

若使用 Docker Compose，请确保环境变量已经提供真实 LLM 参数，再启动 API 和 worker；当前默认行为已经改为严格真实 LLM 模式。

## 6. 测试方式

本次已执行：

- `go test ./...`（backend 全量通过）

重点覆盖内容包括：

- General Worker role_code 默认映射与基线挂载关系
- precheck 与 specialized runner 的严格查找行为
- LLM runner 在调用失败、解析失败、校验失败时不再降级
- 分析任务与测试任务的工件类型约束
- reviewer 对占位工件和错配工件的拒绝逻辑
- artifact storage 严格失败行为和 artifact 创建回滚路径

## 7. 已知限制

- 仓库中仍保留部分 Mock Runner / Mock Tool 代码用于测试与兼容性覆盖，但它们不再注册进主工作流链路，也不再作为 LLM 失败时的降级出口。
- 当前严格模式会显著提高环境配置要求；若 provider、model、API key 或对象存储不可用，工作流会直接失败，这是本次修复的预期行为。
- 本次没有删除全部历史 mock 代码文件，而是先把主链路上的注册、授权、路由与降级行为彻底切断；如果后续要做仓库级清理，可以再单独收口未被主链路使用的 mock 实现。
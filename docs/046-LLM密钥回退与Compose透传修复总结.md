# 046-LLM密钥回退与Compose透传修复总结

## 变更摘要

本次继续追踪真实 compose 工作流链路时，发现环境发生了重置：Docker Compose 恢复后重新创建了 `lingchoubot_pgdata`，原数据库中的项目、运行记录以及手工维护过的 LLM provider `api_key` 一并丢失。

在新的干净环境中重新启动真实 workflow run 后，链路的首个阻塞点不再是 PRD 校验，而是 PM 第一步直接因 OpenAI 401 缺少 API key 进入 `waiting_manual_intervention`。

围绕这个问题，本次完成了两类修复：

1. 运行时修复：当数据库里存在 provider 记录但 `api_key` 为空时，允许继续回退到容器环境变量配置，而不是把空 key 直接用于调用。
2. Compose 修复：将 `.env.example` 已声明但 `docker-compose.yml` 先前未透传的 provider 级、角色级 LLM 环境变量补齐到 `api`/`worker` 容器中。

## 文件清单

- 修改：`backend/internal/runtime/llm_runner.go`
- 修改：`backend/internal/runtime/llm_runner_test.go`
- 修改：`backend/internal/testutil/testutil.go`
- 修改：`docker-compose.yml`

## 关键实现说明

### 1. 数据库空 key 不再遮蔽环境变量回退

旧逻辑中，`LLMAgentRunner.clientForInput` 会优先调用数据库 provider 查找：

- 只要数据库里存在该 provider 记录，就认为命中成功；
- 即使返回的 `api_key` 是空字符串，也不会再回退到环境变量配置；
- 结果是数据库重置或重新种子化后，provider 记录仍在，但 `api_key` 被清空，真实 LLM 调用会直接带空 key 发起请求。

本次修复后：

- 动态 provider 查找到数据库记录时，仍以数据库值优先；
- 但如果 `base_url` 或 `api_key` 缺失，会继续用静态 provider 配置（即容器环境变量解析结果）补齐；
- 这样数据库 provider 记录与环境变量能够协同工作，不会出现“数据库有空值就把 env 回退彻底挡死”的情况。

### 2. Compose 透传补齐 provider/role 级变量

`.env.example` 之前已经声明了以下变量，但 `docker-compose.yml` 实际没有透传到容器：

- provider 级：`LLM_OPENAI_*`、`LLM_DEEPSEEK_*`、`LLM_QWEN_*`、`LLM_MOONSHOT_*`、`LLM_ZHIPU_*`、`LLM_SILICONFLOW_*`、`LLM_OPENROUTER_*`、`LLM_OLLAMA_*`
- role 级：`LLM_PM_*`、`LLM_SUPERVISOR_*`、`LLM_WORKER_*`、`LLM_REVIEWER_*`

这意味着即使宿主机 `.env` 已经设置了 provider-specific key，容器内 runtime 也拿不到这些值。

本次已将上述变量补齐到 `api` 和 `worker` 服务的环境变量列表中，使 provider/role 覆盖配置在 Compose 运行态真正生效。

### 3. 补齐测试夹具排序，恢复全量测试稳定性

在重跑后端全量测试时，`TestIntegration_LLMFailureWaitsForManualInterventionAndResume` 失败。根因不是本次业务修复，而是 `FakeWorkflowStepRepo` 从 map 直接返回步骤，顺序不稳定；而真实仓库实现是按 `sort_order, created_at` 排序。

本次将假仓库的 `ListByRunID` 与 `StepsForRun` 都调整为按 `sort_order, created_at` 稳定排序，对齐真实仓库行为，恢复了编排集成测试的稳定性。

## 运行方式

1. 在仓库根目录执行：`docker compose up -d --build api`
2. 确保 `.env` 中至少配置一个可用的真实 LLM key：
   - 全局方式：`LLM_API_KEY=...`
   - 或 provider 方式：如 `LLM_OPENAI_API_KEY=...` / `LLM_DEEPSEEK_API_KEY=...`
3. 调用 `POST /api/v1/projects` 创建项目，再调用 `POST /api/v1/orchestrator/runs` 启动 workflow

## 测试方式

- 运行时单测：`go test ./internal/runtime`
- 人工介入编排测试：`go test ./internal/orchestrator -run TestIntegration_LLMFailureWaitsForManualInterventionAndResume -count=1 -v`
- 后端全量测试：`go test ./...`
- Compose 配置渲染检查：`docker compose config`

## 当前验证结果

1. 新建验证项目 `Compose链路续跑验证-20260403` 后，run `2e608b18-58f8-4d8f-a52b-561cc6dfee0e` 在 PM 步骤进入 `waiting_manual_intervention`。
2. 错误信息明确为 OpenAI 401：未提供 API key。
3. 仓库根目录 `.env` 当前状态为：
   - `LLM_API_KEY` 存在但为空
   - `LLM_OPENAI_API_KEY` / `LLM_DEEPSEEK_API_KEY` / `LLM_QWEN_API_KEY` / `LLM_MOONSHOT_API_KEY` / `LLM_ZHIPU_API_KEY` 当前均未配置
4. 因此本地当前环境仍无法继续跑通真实 LLM 链路，但代码层面的“空 DB key -> env 回退”与 Compose 变量透传已经补齐。

## 已知限制

1. 当前宿主机 `.env` 中没有任何可用真实 LLM key，所以无法在本地继续验证更深一步的真实链路。
2. 这次修复保证“如果 key 在 env 里存在，数据库空值不会再挡住它”，但不能替代实际密钥配置本身。
3. 数据库重置导致此前历史运行记录与 provider 手工配置丢失；如果要恢复旧运行态，需要重新写入 provider key 或恢复旧 volume。
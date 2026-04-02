# 032-Agent默认LLM与模型配置实现总结

## 1. 变更摘要

本次将 Agent 创建流程改为默认使用 LLM，不再在前端新建表单中暴露 mock 类型；同时新增主流 OpenAI 兼容 Provider 与模型预设，并将 Agent 级别的 Provider / Model 配置真正接入运行时，使配置不再只是展示字段。

## 2. 文件清单

### 后端

- backend/internal/model/agent_llm.go
- backend/internal/service/agent.go
- backend/internal/service/agent_bootstrap.go
- backend/internal/service/agent_bootstrap_test.go
- backend/internal/config/config.go
- backend/internal/config/config_test.go
- backend/internal/runtime/protocol.go
- backend/internal/runtime/llm_runner.go
- backend/internal/runtime/llm_runner_test.go
- backend/internal/orchestrator/agent_runtime.go
- backend/internal/orchestrator/engine.go
- backend/internal/orchestrator/temporal_activities.go
- backend/cmd/api/main.go

### 前端

- frontend/src/lib/agent-llm.ts
- frontend/src/lib/types.ts
- frontend/src/app/agents/page.tsx

### 文档与配置

- .env.example
- README.md
- docs/005-API 接口文档.md
- docs/032-Agent默认LLM与模型配置实现总结.md

## 3. 关键实现说明

1. Agent 默认类型由 mock 调整为 llm，并在服务层统一补齐与校验 `metadata.llm`，避免前端漏传或脏数据直接入库。
2. 新增 Provider 级配置表，支持 OpenAI、DeepSeek、Qwen、Moonshot、Zhipu、SiliconFlow、OpenRouter、Ollama，并允许在运行时按 Agent 元数据覆盖模型与连接目标。
3. 本地编排与 Temporal 活动都注入 Agent 级 LLM 配置，确保控制台上选择的 Provider / Model 会实际影响执行。
4. 前端 Agent 页面移除新建 mock 选项，新增 Provider / 模型预设与自定义模型输入；编辑历史 mock Agent 时仍保留兼容入口。
5. 启动自举的基础 Agent 同步切换为默认 LLM 配置，保持系统初始数据与新规则一致。

## 4. 为什么这样改

- 用户目标是减少手工配置负担，并且直接为 Agent 选择真实可用的大模型。
- 仅改前端表单会造成“看起来可配置、实际上不生效”的假象，因此必须把 Agent 元数据一路传到 runtime。
- 保留 mock 的后端兼容能力，是为了不破坏历史测试数据和已有运行环境。

## 5. 假设

1. 各 Provider 均通过 OpenAI 兼容接口接入。
2. Agent 级模型覆盖只调整 Provider / Model，不单独维护每个 Agent 的 API Key，密钥仍通过环境变量按 Provider 管理。
3. 历史 mock Agent 仍可能存在，因此编辑和运行时兼容比彻底移除枚举更安全。

## 6. 运行方式

1. 在根目录复制 `.env.example` 为 `.env`。
2. 至少配置一个可用 Provider 的 API Key，例如 `LLM_OPENAI_API_KEY` 或 `LLM_DEEPSEEK_API_KEY`。
3. 设置 `LLM_ENABLED=true`，按需设置 `LLM_FALLBACK_ENABLED`。
4. 启动后端与前端，进入 Agent 管理页创建或编辑 Agent，选择 Provider 与模型。

## 7. 测试方式

1. 运行后端测试，覆盖配置加载、Agent 默认值、Provider 校验、runtime 覆盖逻辑。
2. 运行后端构建与前端构建，确认类型和编译通过。
3. 手动在控制台创建不同 Provider / Model 的 Agent，执行工作流并确认实际走向对应 Provider 配置。

## 8. 已知限制

1. 当前 Provider 列表是内置枚举，新增新厂商仍需要改代码与前端预设。
2. Agent 级别暂不支持单独填写 API Key，仅支持通过环境变量按 Provider 管理。
3. 旧数据中的 mock Agent 仍然存在，前端仅在编辑场景保留兼容，不会主动迁移历史记录。
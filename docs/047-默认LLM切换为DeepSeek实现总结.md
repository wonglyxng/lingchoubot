# 047-默认LLM切换为DeepSeek实现总结

## 1. 变更摘要

本次将系统默认 LLM 从 OpenAI 切换为 DeepSeek，覆盖后端默认值、Agent 自举默认元数据、前端默认选项、Docker Compose 默认环境变量以及示例环境文件，并补充数据库 migration 以修正已有 provider/model 排序与历史默认 Agent 元数据。

同时完成一次真实 Compose 运行态验证：恢复原先因 OpenAI 401 挂起的工作流后，运行已成功越过 PM 阶段并进入 QA 规划与执行，说明默认 LLM 已实际切换为 DeepSeek。当前新的阻塞点为 MinIO 未启动导致工件内容无法落盘，这属于运行环境问题，不属于本次 LLM 默认切换问题。

## 2. 文件清单

修改文件：

- `.env.example`
- `docker-compose.yml`
- `backend/cmd/demo/main.go`
- `backend/internal/config/config.go`
- `backend/internal/config/config_test.go`
- `backend/internal/model/agent_llm.go`
- `backend/internal/service/agent_bootstrap.go`
- `frontend/src/lib/agent-llm.ts`
- `backend/migrations/000019_default_llm_to_deepseek.up.sql`
- `backend/migrations/000019_default_llm_to_deepseek.down.sql`

## 3. 关键实现说明

### 3.1 后端默认值统一切换

- 在 `backend/internal/model/agent_llm.go` 中将默认 provider/model 改为 `deepseek` / `deepseek-chat`。
- 在 `backend/internal/service/agent_bootstrap.go` 中取消基线 Agent 对 OpenAI 的硬编码，改为读取统一默认常量，避免后续再次出现代码默认值与 bootstrap 默认值不一致。
- 在 `backend/internal/config/config.go` 中将全局回退默认 `LLM_BASE_URL` 改为 `https://api.deepseek.com/v1`，默认 `LLM_MODEL` 改为 `deepseek-chat`。

### 3.2 前端与 Demo 默认值同步

- 在 `frontend/src/lib/agent-llm.ts` 中将 Agent 配置界面的默认 provider/model 切换为 DeepSeek，避免前端新增或编辑 Agent 时继续回填 OpenAI。
- 在 `backend/cmd/demo/main.go` 中将 Demo 初始化的 LLM Agent 元数据改为 DeepSeek，保证演示链路与真实默认值一致。

### 3.3 运行态与存量数据兼容

- 在 `docker-compose.yml` 和 `.env.example` 中同步切换默认环境变量，保证未显式覆盖时 Compose 默认行为与代码一致。
- 新增 migration `000019_default_llm_to_deepseek`：
  - 将 `llm_provider` 中 DeepSeek 调整为更靠前的排序。
  - 将 `llm_model` 中 `deepseek-chat` 设为默认模型，并调整与 OpenAI 默认模型的顺序。
  - 将现有默认风格的 OpenAI Agent 元数据重写为 DeepSeek，避免数据库保留旧默认值导致代码生效不完整。

### 3.4 验证结果

- 恢复运行 `2e608b18-58f8-4d8f-a52b-561cc6dfee0e` 后，原先的 OpenAI 401 未再次出现。
- 基线 8 个 Agent 的 `metadata.llm` 已全部变为 `deepseek` / `deepseek-chat`。
- 工作流已从 PM 阶段继续推进到 QA 规划与执行步骤，证明运行态默认 LLM 切换已生效。
- 当前失败原因为工件存储访问 MinIO 失败：`lookup minio ... no such host`，与 LLM provider 无关。

## 4. 运行方式

### 4.1 后端测试

在 `backend/` 目录执行：

```bash
go test ./...
```

### 4.2 前端构建

在 `frontend/` 目录执行：

```bash
npm run build
```

### 4.3 Compose 验证

在仓库根目录执行：

```bash
docker compose up -d --build api
```

如需继续验证工件落盘链路，应补齐对象存储相关服务：

```bash
docker compose up -d minio createbuckets
```

## 5. 测试方式

本次已完成以下验证：

1. `backend` 全量测试通过。
2. `frontend` 生产构建通过。
3. 运行态查询确认基线 Agent 的 provider/model 已切换到 DeepSeek。
4. 恢复真实工作流运行，确认原先的 OpenAI 401 消失，执行链路继续推进。

## 6. 已知限制

1. 当前 Compose 环境未启动 MinIO，导致工件存储步骤失败；这不是本次 LLM 默认切换导致的问题。
2. 本次仅处理默认值与存量默认 Agent 的迁移，不会覆盖用户手工指定的非默认 provider/model 配置。
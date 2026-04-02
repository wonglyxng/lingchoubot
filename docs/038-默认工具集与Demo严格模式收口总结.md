# 038-默认工具集与Demo严格模式收口总结

## 1. 变更摘要

本次改动继续收口上一轮“严格真实 LLM、无 Mock 降级”整改后遗留的运行时入口，重点解决两类问题：

- gateway 默认工具集里仍然注册了会返回占位内容的 `doc_generator` 和 `test_runner`；虽然主工作流已不再授权它们，但它们仍可被运行时列出和调用。
- `backend/cmd/demo` 仍按旧逻辑创建 mock Agent、遗漏 `GENERAL_WORKER`、并把工具权限错误写进 `capabilities` 字段，导致 Demo 命令与当前严格模式设计不一致。

本次实现后：

- gateway 默认只注册 `artifact_storage` 这一条真实工具链路；若存储不可用，工具会显式失败。
- `doc_generator` 与 `test_runner` 保留类型定义仅用于兼容，但即使被误注册，也会直接返回失败，不再产生伪造产物。
- baseline PM Agent 不再携带 `doc_generator` 权限。
- Demo 命令改为“校验并补齐 baseline LLM Agent”，并使用 `artifact_storage` 演示 Tool Gateway，而不是调用假文档工具。

## 2. 文件清单

- `backend/internal/gateway/gateway.go`
- `backend/internal/gateway/artifact_storage.go`
- `backend/internal/gateway/doc_generator.go`
- `backend/internal/gateway/test_runner.go`
- `backend/internal/gateway/gateway_test.go`
- `backend/internal/gateway/artifact_storage_test.go`
- `backend/internal/gateway/permission_test.go`
- `backend/internal/service/agent_bootstrap.go`
- `backend/internal/service/agent_bootstrap_test.go`
- `backend/cmd/demo/main.go`
- `docs/038-默认工具集与Demo严格模式收口总结.md`

## 3. 关键实现说明

### 3.1 默认工具集只保留真实工具

- `Gateway.RegisterDefaults(...)` 不再注册 `DocGeneratorTool` 和 `TestRunnerTool`。
- 当未注入真实 `ArtifactStorageTool` 时，gateway 仍会注册一个“显式不可用”的 `artifact_storage`，这样：
  - `/api/v1/tools` 的返回仍然稳定；
  - 但真正调用时会返回失败，而不是伪成功。

这样处理的原因是：严格模式下，默认工具集不应再暴露会生成假结果的入口，但也要保留一致的工具注册行为，避免接口形态漂移。

### 3.2 假工具改为 fail-closed

- `DocGeneratorTool.Execute(...)` 现在直接返回 failed，并提示改由真实 LLM Agent 生成工件。
- `TestRunnerTool.Execute(...)` 现在直接返回 failed，并提示改走真实测试执行链路。

这样处理的原因是：仅仅“不默认注册”还不够，后续如果有人手动把这两个工具重新注册进 gateway，仍然可能再次制造模板化假产物，因此要把实现本身也改成关闭状态。

### 3.3 Demo 对齐严格基线

- Demo 第二步从“注册 Agent 组织树”改为“校验并补齐 Agent 组织树（严格 LLM baseline）”。
- 增补 `GENERAL_WORKER`，避免当前 precheck 因缺失关键角色码而失败。
- Agent 创建/更新统一使用 `agent_type=llm`。
- Tool 权限改写入 `allowed_tools`，不再错误写进 `capabilities`。
- Demo 的 Tool Gateway 演示从 `doc_generator` 切换为 `artifact_storage` 的 `write` 动作。

这样处理的原因是：Demo 命令必须验证当前真实链路，而不是继续复用旧版 Mock 演示逻辑。

### 3.4 baseline 权限进一步去 Mock 化

- PM baseline Agent 的 `AllowedTools` 从 `doc_generator` 改为 `[]`。
- 对应测试补充了断言，确保基线角色不会重新带回这个权限。

这样处理的原因是：项目级规划 Agent 在当前架构里应依赖 LLM 输出，不应再保留旧 mock 文档工具权限。

## 4. 假设

- API 服务启动时已经具备严格模式所需的真实 LLM 配置。
- Demo 运行环境中，如果要让 `artifact_storage` 工具步骤成功，MinIO 必须可用。
- 现有数据库中如果已存在同 role_code 的旧 Agent，Demo 会通过更新而不是重复创建来对齐到严格基线。

## 5. 运行方式

### 后端验证

在 `backend` 目录执行：

```bash
go test ./...
```

### Demo 验证

确保 API、数据库、对象存储与真实 LLM 配置已就绪后，可执行：

```bash
go run ./cmd/demo
```

当前 Demo 会先校验并补齐 baseline LLM Agent，再启动工作流，并用 `artifact_storage` 验证 Tool Gateway。

## 6. 测试方式

本次已执行：

- `go test ./...`（backend 全量通过）

新增/覆盖重点：

- 默认工具集只注册 `artifact_storage`
- `doc_generator` 和 `test_runner` 在严格模式下 fail-closed
- baseline PM Agent 不再持有 `doc_generator` 权限
- 不可用 artifact storage 的显式失败行为仍可测试覆盖

## 7. 已知限制

- 仓库内仍保留 `DocGeneratorTool`、`TestRunnerTool` 这类 fail-closed 兼容工具类型；它们已不在默认运行时链路中生效。
- Demo 现在依赖真实对象存储链路，因此若 MinIO 未配置或不可用，Tool Gateway 演示步骤会按严格模式直接失败。
- 本次未清理所有测试中的 `mock` 字面量，因为这些用例仍在验证兼容分支和模型枚举本身，而不是生产运行时行为。
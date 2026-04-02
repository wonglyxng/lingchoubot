# 044-SSE中间件透传与Artifact空Metadata修复总结

## 变更摘要

本次修复了运行日志中暴露出的两类真实代码问题：

1. `GET /api/v1/events/stream` 持续返回 500
2. 工作流执行过程中，Artifact 初始版本写入在 `metadata = null` 场景下触发 nil map panic

同时顺手修复了一个集成测试竞态，恢复 backend 全量测试稳定通过。

## 文件清单

- `backend/internal/middleware/logging.go`
- `backend/internal/middleware/logging_test.go`
- `backend/internal/service/artifact.go`
- `backend/internal/service/artifact_test.go`
- `backend/internal/orchestrator/integration_test.go`
- `docs/044-SSE中间件透传与Artifact空Metadata修复总结.md`

## 关键实现说明

### 1. SSE 500 的根因与修复

根因不是 SSE Handler 本身逻辑错误，而是 Logging 中间件包装 `http.ResponseWriter` 后，没有继续暴露 `http.Flusher` 能力。

SSE Handler 在进入流式响应前会检查：

- `w.(http.Flusher)`

由于中间件包装后的 writer 不再满足该接口，接口直接返回 500。

本次修复：

- 为 `wrappedWriter` 增加 `Flush()` 转发
- 为 `wrappedWriter` 增加 `Unwrap()`，确保 `http.NewResponseController()` 之类能力可以继续透传到底层 writer
- 补 middleware 单元测试，验证 Logging 包装后仍保留 Flusher 能力

### 2. Artifact nil map panic 的根因与修复

根因在于 `artifactVersionMetadata()` 使用 `json.Unmarshal()` 将 `metadata` 反序列化到 `map[string]any`。

当 `metadata` 内容为 JSON `null` 时：

- `Unmarshal` 不报错
- 但结果 map 为 `nil`

后续 `prepareVersionContent()` 再写入：

- `meta["stored_in"] = ...`

就会触发 `assignment to entry in nil map`。

本次修复：

- 在 `artifactVersionMetadata()` 中将 `null` 结果统一归一化为空 map
- 补 service 单元测试，覆盖 `metadata = null` 时仍能成功创建版本并写入衍生 metadata

### 3. 集成测试竞态修复

`Engine.ResumeRun()` 当前是异步恢复：

- 先把 run 状态改为 `running`
- 再通过 goroutine 继续编排执行

原有集成测试在 `ResumeRun()` 返回后立刻断言最终状态，存在竞态。

本次测试修正为短时间轮询等待最终状态，避免把异步实现误测成同步语义。

## 运行方式

后端目录执行：

```bash
go test ./...
```

如果需要重新验证容器内表现，重新构建并启动 API 服务后，可观察：

- `/api/v1/events/stream` 不再持续返回 500
- 工作流写入工件版本时不再因 `metadata = null` panic

## 测试方式

本次通过以下命令验证：

```bash
cd backend
go test ./internal/middleware/... ./internal/service/...
go test ./internal/orchestrator/...
go test ./...
```

验证结果：

- middleware 测试通过
- service 测试通过
- orchestrator 测试通过
- backend 全量测试通过

## 已知限制

- 日志中反复出现的 `postgres` 用户 `wanglixiang` 登录失败，更像宿主机上的外部客户端或扩展在尝试直连 `localhost:5432`，不属于当前 API 服务的数据库连接故障
- 若要让容器中的 API 实际用上本次修复，仍需重新构建并重启对应镜像
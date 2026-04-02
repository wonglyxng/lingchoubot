# 048-Compose-MinIO依赖与对象存储链路修复总结

## 1. 变更摘要

本次修复了 Docker Compose 运行态中 API 单独启动时不会自动带起 MinIO 的问题。此前使用 `docker compose up -d --build api` 时，只会启动 `postgres`、`migrate` 和 `api`，导致 API 在初始化工件存储工具时无法解析 `minio` 主机，后续工作流在首次写入 artifact 时失败。

修复后，`api` 与 `worker` 都显式声明依赖 `minio`，从而保证对象存储服务会随工作流执行面一起启动。完成修复后，重新强制重建 API 并启动新的真实工作流，已验证链路成功越过此前的 MinIO 失败点。

## 2. 文件清单

修改文件：

- `docker-compose.yml`

新增文档：

- `docs/048-Compose-MinIO依赖与对象存储链路修复总结.md`

## 3. 关键实现说明

### 3.1 根因分析

- `api` 运行时会初始化 `artifact_storage` 工具，并在启动阶段执行 MinIO bucket 检查。
- `docker-compose.yml` 中虽然为 `api` 注入了 `MINIO_ENDPOINT=minio:9000`，但 `api` 的 `depends_on` 里没有 `minio`。
- 因此当只启动 `api` 时，Compose 不会自动启动 MinIO，API 容器会在启动时缓存 `minio bucket check failed` 错误。
- 即使稍后再单独启动 MinIO，已经运行中的 API 容器仍会沿用启动时缓存的初始化错误，直到被重建。

### 3.2 修复内容

在 `docker-compose.yml` 中为以下服务新增了 `depends_on.minio`：

- `api`
- `worker`

依赖条件使用 `service_started`，因为当前 MinIO 服务未配置 healthcheck，而后端侧已经会在工具初始化时自行检查 bucket 是否存在。

### 3.3 真实运行态验证

修复后执行：

```bash
docker compose up -d minio
docker compose up -d --force-recreate api
```

随后新启动工作流运行 `c3deefac-bbf5-4f40-b26f-8f1940aab013`，观察到：

1. PM 项目分解已完成。
2. 第一个 worker 执行步骤已完成。
3. 第一个 reviewer 步骤已完成。
4. 后续返工与再次评审步骤继续推进。

这说明此前在首个 artifact 写入时触发的 MinIO 主机解析失败已经消失，对象存储链路恢复正常。

## 4. 运行方式

在仓库根目录执行：

```bash
docker compose up -d --build api
```

在本次修复后，上述命令会自动带起 `minio` 依赖。

如果 API 容器是在 MinIO 不可用时启动过，需再执行一次强制重建以清除缓存的初始化错误：

```bash
docker compose up -d --force-recreate api
```

## 5. 测试方式

本次完成的验证包括：

1. 检查 compose 服务状态，确认 `api` 容器已声明并带起 `minio` 依赖。
2. 强制重建 API，确保工件存储工具重新初始化。
3. 启动新的真实工作流运行并持续轮询。
4. 确认工作流已越过原先的 artifact 存储失败点，并继续进入评审与返工阶段。

## 6. 已知限制

1. 当前修复的是 Compose 依赖编排问题，不改变 MinIO 凭据或 bucket 命名策略。
2. 若运行中的 API 容器已经缓存旧的 MinIO 初始化错误，仍需重建容器；单纯启动 MinIO 不足以恢复已启动的 API 进程。
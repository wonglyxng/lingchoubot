# 027 Docker Compose 端口覆盖与前端地址构建修复总结

## 1. 变更摘要

本次修复了 `docker compose up` 在本机 `8080` 或 `3000` 端口被占用时无法顺利启动 API / 前端服务的问题。

修复内容包括：

- 为 Docker Compose 的 API 主机端口增加 `.env` 可覆盖能力。
- 为 Docker Compose 的前端主机端口增加 `.env` 可覆盖能力。
- 让前端镜像在构建期和运行期都使用可配置的 `NEXT_PUBLIC_API_URL`，避免只改端口后前端仍然请求旧地址。
- 在启动文档中补充端口冲突时的改法与使用方式。

## 2. 文件清单

修改文件：

- `docker-compose.yml`
- `frontend/Dockerfile`
- `.env.example`
- `README.md`

## 3. 关键实现说明

### 3.1 API 主机端口与容器端口解耦

`docker-compose.yml` 中原先将 API 端口写死为 `8080:8080`。现在改为：

- 宿主机端口使用 `API_HOST_PORT`
- 容器内监听端口继续使用 `SERVER_PORT`

这样在不改后端默认监听行为的情况下，可以通过 `.env` 将宿主机暴露端口切换到 `18080` 等其他值。

### 3.2 前端宿主机端口改为可覆盖

`docker-compose.yml` 中原先将前端端口写死为 `3000:3000`。现在改为：

- 宿主机端口使用 `FRONTEND_HOST_PORT`
- 容器内监听端口继续保持 `3000`

这样在不改前端容器内部启动方式的情况下，可以通过 `.env` 将宿主机暴露端口切换到 `13000` 等其他值。

### 3.3 前端 API 地址改为构建期可注入

前端代码使用 `NEXT_PUBLIC_API_URL` 作为 API 基础地址。该变量对浏览器端代码属于构建期注入变量，仅在 Docker Compose 的 service `environment` 中设置并不能保证客户端 bundle 拿到新值。

因此本次在 `frontend/Dockerfile` 中加入了 `ARG NEXT_PUBLIC_API_URL`，并在构建阶段写入 `ENV NEXT_PUBLIC_API_URL`，同时在 `docker-compose.yml` 中通过 `build.args` 传入该值。这样端口变化后，重新构建前端镜像即可使前端请求新的 API 地址。

## 4. 运行方式

默认启动：

```bash
cp .env.example .env
docker compose up --build
```

如果 `8080` 或 `3000` 已被占用：

```bash
cp .env.example .env
```

将 `.env` 中以下变量修改为：

```bash
API_HOST_PORT=18080
NEXT_PUBLIC_API_URL=http://localhost:18080
FRONTEND_HOST_PORT=13000
```

然后执行：

```bash
docker compose up --build -d
```

## 5. 测试方式

建议按以下步骤验证：

1. 保持本机 `8080` 或 `3000` 被其他进程占用，或直接在 `.env` 中将 `API_HOST_PORT` 改为 `18080`、`FRONTEND_HOST_PORT` 改为 `13000`。
2. 执行 `docker compose up --build -d`。
3. 访问 `http://localhost:18080/healthz`，确认 API 可用。
4. 打开 `http://localhost:13000`，确认前端可访问。
5. 在前端页面发起数据请求，确认浏览器仍访问新的 API 地址而不是固定的 `8080`。

## 6. 已知限制

- 当前 README 仍以默认端口 `8080` 作为示例地址；如果实际改了 `API_HOST_PORT`，访问地址需以 `.env` 中配置为准。
- 本次已处理 API `8080` 与前端 `3000` 的宿主机端口冲突问题，但尚未对数据库 `5432`、MinIO `9000/9001` 等其他宿主机端口做同样的可配置化。
- 若修改了 `NEXT_PUBLIC_API_URL`，需要重新构建前端镜像才能让浏览器端 bundle 生效。
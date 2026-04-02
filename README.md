# 灵筹 LingChou

面向复杂项目交付的多智能体组织操作系统。

核心链路：`项目 → 阶段 → 任务 → 工件 → 审批 → 审计`

## 技术栈

- **后端**: Go + PostgreSQL + REST API
- **前端**: Next.js + React + TypeScript + Tailwind CSS
- **基础设施**: Docker Compose

## 本地启动

### 前置要求

- Docker & Docker Compose
- Go 1.23+（仅本地开发时需要）
- Node.js 22+（仅本地开发时需要）

### 一键启动（Docker Compose）

```bash
# 复制环境变量
cp .env.example .env

# 启动所有服务（postgres + migration + api + frontend）
docker compose up --build
```

如果本机 `8080` 或 `3000` 已被占用，可先在 `.env` 中改成其他端口，例如：

```bash
API_HOST_PORT=18080
NEXT_PUBLIC_API_URL=http://localhost:18080
FRONTEND_HOST_PORT=13000
```

默认配置下启动后：
- API 服务: http://localhost:8080
- 前端控制台: http://localhost:3000
- 健康检查: http://localhost:8080/healthz
- 就绪检查: http://localhost:8080/readyz

如果使用了上面的覆盖示例，则前端控制台地址变为 `http://localhost:13000`。

### 本地开发（不用 Docker）

**启动数据库：**

```bash
docker compose up postgres -d
```

**启动后端：**

```bash
cd backend

# 运行 migration
DB_HOST=localhost go run ./cmd/migrate -direction up

# 启动 API 服务
DB_HOST=localhost go run ./cmd/api
```

**启动前端：**

```bash
cd frontend
npm install
npm run dev
```

## LLM Agent 配置

系统现在默认将新建 Agent 视为 LLM Agent。控制台创建 Agent 时不再提供 mock 选项，而是直接选择 Provider 与模型；后端仍兼容历史 mock 记录，避免旧数据失效。

当前内置 Provider 预设包括：OpenAI、DeepSeek、Qwen、Moonshot、Zhipu、SiliconFlow、OpenRouter、Ollama。对应连接信息通过 `.env` 配置，例如：

```bash
LLM_ENABLED=true
LLM_FALLBACK_ENABLED=false

LLM_OPENAI_API_KEY=sk-...
LLM_DEEPSEEK_API_KEY=sk-...
LLM_OLLAMA_BASE_URL=http://localhost:11434/v1
```

当 Agent 被配置为 LLM 类型时，其 `metadata.llm.provider` 与 `metadata.llm.model` 会在运行时覆盖默认角色模型配置。若某个 Provider 没有单独填写 Base URL，则使用系统内置的 OpenAI 兼容地址模板。

## 项目结构

当前主要结构概览如下：

```
lingchoubot/
├── AGENTS.md                    # 编码智能体操作规范
├── README.md                    # 项目说明与启动指引
├── docker-compose.yml           # 本地开发与联调编排
├── docs/                        # 架构、API、阶段总结文档
├── backend/                     # Go 后端
│   ├── Dockerfile               # 后端镜像构建（api / migrate / worker）
│   ├── cmd/
│   │   ├── api/                 # API 服务入口
│   │   ├── demo/                # 演示入口
│   │   ├── migrate/             # Migration 工具入口
│   │   └── worker/              # Temporal Worker 独立进程入口
│   ├── internal/
│   │   ├── config/              # 配置加载
│   │   ├── gateway/             # 工具网关与工件存储
│   │   ├── handler/             # HTTP Handler
│   │   ├── middleware/          # 中间件
│   │   ├── model/               # 领域模型
│   │   ├── orchestrator/        # 本地/Temporal 编排引擎
│   │   ├── repository/          # 数据访问层
│   │   ├── runtime/             # Agent 运行时适配
│   │   └── service/             # 业务逻辑层
│   └── migrations/              # SQL Migration 文件
└── frontend/                    # Next.js 前端
    ├── Dockerfile               # 前端镜像构建
    ├── public/                  # 静态资源
    └── src/
        ├── app/                 # App Router 页面
        ├── components/          # 共享组件
        └── lib/                 # 前端工具与 API 封装
```

## API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/healthz` | 健康检查 |
| GET | `/readyz` | 就绪检查（含数据库连通性） |
| GET | `/api/v1/ping` | API 连通性测试 |

> 更多 API 将在后续迭代中实现，参见 `docs/003-灵筹 MVP 开发路线图.md`

## 文档

### 设计文档
- [系统总体架构设计](docs/001-灵筹系统总体架构设计.md)
- [Agent 组织树与职责说明](docs/002-灵筹%20Agent%20组织树与职责说明.md)
- [MVP 开发路线图](docs/003-灵筹%20MVP%20开发路线图（按周拆解）.md)

### 技术文档
- [数据库设计](docs/004-数据库设计.md)
- [API 接口文档](docs/005-API%20接口文档.md)
- [变更日志](CHANGELOG.md)

### 规范
- [编码智能体操作规范](AGENTS.md)

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

启动后：
- API 服务: http://localhost:8080
- 前端控制台: http://localhost:3000
- 健康检查: http://localhost:8080/healthz
- 就绪检查: http://localhost:8080/readyz

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

## 项目结构

```
lingchoubot/
├── AGENTS.md                # 编码智能体操作规范
├── docker-compose.yml       # 本地环境编排
├── docs/                    # 设计文档
│   ├── 001-灵筹系统总体架构设计.md
│   ├── 002-灵筹 Agent 组织树与职责说明.md
│   └── 003-灵筹 MVP 开发路线图.md
├── backend/                 # Go 后端
│   ├── cmd/
│   │   ├── api/             # API 服务入口
│   │   └── migrate/         # Migration 工具
│   ├── internal/
│   │   ├── config/          # 配置管理
│   │   ├── model/           # 数据模型
│   │   ├── repository/      # 数据访问层
│   │   ├── service/         # 业务逻辑层
│   │   ├── handler/         # HTTP handler
│   │   └── middleware/      # 中间件
│   └── migrations/          # SQL migration 文件
└── frontend/                # Next.js 前端
    └── src/
        ├── app/             # 页面路由
        └── components/      # 共享组件
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

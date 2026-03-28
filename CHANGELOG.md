# 灵筹变更日志

本文档记录每个工作包（WP）的实现内容，便于后续了解每次变更的具体范围和决策。

---

## WP-01：初始化 Monorepo 骨架（2026-03-28）

### 目标

搭建灵筹最小工程骨架，确保仓库、数据库、API 进程、前端壳、Docker 环境都能启动。

### 新增文件

**后端（Go）：**

| 文件 | 说明 |
|------|------|
| `backend/go.mod` / `go.sum` | Go 模块，依赖 `lib/pq` + `golang-migrate` |
| `backend/cmd/api/main.go` | API 服务入口：HTTP server + JSON 日志 + 优雅关闭 |
| `backend/cmd/migrate/main.go` | Migration CLI 工具：支持 up / down |
| `backend/internal/config/config.go` | 配置加载：从环境变量读取，带默认值 |
| `backend/internal/repository/db.go` | PostgreSQL 连接池：25 max open, 5 max idle, 5min lifetime |
| `backend/internal/handler/health.go` | 健康检查：`/healthz`（存活）+ `/readyz`（含 DB ping） |
| `backend/internal/middleware/response.go` | 统一 JSON 响应：`{success, data, error}` 格式 |
| `backend/internal/middleware/logging.go` | 请求日志：method, path, status, duration_ms |
| `backend/internal/middleware/recovery.go` | panic 恢复：捕获 panic 返回 500 |
| `backend/internal/middleware/cors.go` | CORS：允许所有来源（MVP 阶段） |
| `backend/migrations/000001_init.up.sql` | 初始 migration：启用 uuid-ossp + schema_info 表 |
| `backend/migrations/000001_init.down.sql` | 回退：删除 schema_info 表 |
| `backend/Dockerfile` | 多阶段构建：builder(golang:1.25-alpine) -> runner(alpine:3.20) |

**前端（Next.js）：**

| 文件 | 说明 |
|------|------|
| `frontend/package.json` | Next.js 15 + React 19 + Tailwind CSS 4 |
| `frontend/tsconfig.json` | TypeScript 配置 |
| `frontend/next.config.ts` | Next.js 配置：standalone 输出 |
| `frontend/postcss.config.mjs` | PostCSS + Tailwind |
| `frontend/src/app/layout.tsx` | 根布局：侧边栏 + 内容区 |
| `frontend/src/app/page.tsx` | 首页：系统概览 + API/DB 健康状态实时检测 |
| `frontend/src/components/Sidebar.tsx` | 侧边栏导航：7 个页面入口 |
| `frontend/src/app/{projects,tasks,agents,artifacts,approvals,audit}/page.tsx` | 6 个占位页面 |
| `frontend/Dockerfile` | 多阶段构建：deps -> builder -> runner(standalone) |

**基础设施：**

| 文件 | 说明 |
|------|------|
| `docker-compose.yml` | 编排：postgres(16) + migrate + api + frontend |
| `.env` / `.env.example` | 环境变量 |
| `.gitignore` | Git 忽略规则 |
| `README.md` | 项目说明 + 本地启动指南 |

**文档：**

| 文件 | 说明 |
|------|------|
| `docs/004-数据库设计.md` | 数据库表结构、migration 清单、设计原则 |
| `docs/005-API 接口文档.md` | API 端点、请求响应格式、中间件、配置项 |
| `CHANGELOG.md` | 本文件，工作包变更记录 |

### 技术决策

| 决策 | 选择 | 理由 |
|------|------|------|
| Go HTTP 框架 | 标准库 `net/http` | MVP 阶段够用，避免引入额外依赖 |
| 路由 | Go 1.22+ 内置路由模式（`GET /path`） | 标准库已支持方法匹配 |
| 日志 | `log/slog`（JSON handler） | Go 标准库，结构化日志，无额外依赖 |
| Migration | `golang-migrate` + 纯 SQL | AGENTS.md 指定，简单可控 |
| 数据库驱动 | `lib/pq` | 成熟稳定的 PostgreSQL 驱动 |
| 前端框架 | Next.js 15 (App Router) | AGENTS.md 指定技术栈 |
| CSS | Tailwind CSS 4 | 快速开发，无需自定义样式文件 |
| Docker 基础镜像 | golang:1.25-alpine / node:22-alpine | 与本地开发环境一致 |

### 验证结果

| 检查项 | 结果 |
|--------|------|
| `docker compose up --build` 一键启动 | 通过 |
| Migration 自动执行 | 通过 |
| `GET /healthz` 返回 `{"success":true}` | 通过 |
| `GET /readyz` 返回 `{"success":true}` | 通过 |
| `GET /api/v1/ping` 返回版本号 | 通过 |
| 前端 http://localhost:3000 返回 200 | 通过 |
| 所有容器健康运行 | 通过 |

### 已知限制

- CORS 允许所有来源，生产环境需收紧
- 前端页面为占位状态，等待后续 API 实现后对接真实数据
- 无认证机制，MVP 阶段暂不需要
- 数据库连接池参数为硬编码默认值，后续可改为配置

### 下一步

WP-02：PostgreSQL migrations 与核心 schema 落地（project / phase / agent / task / audit_log）

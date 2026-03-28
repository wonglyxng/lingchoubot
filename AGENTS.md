# AGENTS.md

# 灵筹（LingChou）编码智能体操作规范

> 本文档是编码智能体（Cursor / Copilot / Claude 等）在灵筹仓库中工作时必须遵守的操作手册。
> 架构设计详见 `docs/001-灵筹系统总体架构设计.md`，Agent 组织与职责详见 `docs/002-灵筹 Agent 组织树与职责说明.md`，开发路线图详见 `docs/003-灵筹 MVP 开发路线图（按周拆解）.md`。

---

## 1. 项目概述

灵筹是一个**面向复杂项目交付的多智能体组织操作系统**。
核心链路：`项目 -> 阶段 -> 任务 -> 工件 -> 审批 -> 审计`。

当前处于 **MVP 第一阶段**，目标是做一个**最小可运行闭环**，而不是完整 AI 公司。

---

## 2. 当前阶段优先级

### P0 — 必须先做
- 仓库骨架与 Docker Compose
- 数据库 migration 与核心表
- 核心数据模型（project / phase / agent / task）
- 任务契约模型（task_contract）
- 基础 CRUD API
- 审计日志

### P1 — 随后完成
- 任务分派（task_assignment）
- Agent 组织树查询
- 工件与版本（artifact / artifact_version）
- 独立评审与打回机制（review_report）
- 审批请求（approval_request）

### P2 — MVP 后半段
- Tool Gateway 与工具调用记录
- 交接快照与恢复执行（handoff_snapshot）
- 最小 Agent Runtime 协议
- 简版工作流编排

### P3 — MVP 之后
- Temporal 深度接入
- NATS 事件总线
- 高级权限与多租户
- 向量检索与知识库

---

## 3. 技术栈

| 层面 | 选型 |
|------|------|
| 后端 | Go |
| 数据库 | PostgreSQL |
| Migration | 纯 SQL（golang-migrate） |
| API 风格 | REST（MVP 阶段） |
| 前端 | Next.js + React + TypeScript |
| 基础设施 | Docker Compose + MinIO |
| 工作流 | MVP 先用应用内编排，后续切 Temporal |

---

## 4. 核心架构原则

以下原则在编码时必须遵守，不可违反：

1. **项目优先于会话** — 顶层对象是 `Project -> Phase -> Task -> Artifact`，不是聊天会话。
2. **工件优先于对话** — 系统推进依据是工件和状态，不是自然语言讨论。
3. **状态机优先于自由流程** — 任务推进依赖明确状态流转，不允许隐式推进。
4. **最小权限** — 默认不授予多余权限，工具调用、文件写入、数据库变更都须在授权范围内。
5. **可审计** — 所有关键动作必须可追踪（谁发起、谁执行、输入输出、状态变化、是否失败）。
6. **小步提交** — 不要一次性大范围重写多个模块，优先局部闭环、快速可验证。
7. **任务契约先行** — 任务执行前必须先明确范围、非目标、完成定义、验收标准。
8. **独立评审** — 执行 Agent 的自检不等于最终验收，关键任务须独立评审方验收。
9. **结构化交接** — 关键执行轮次须产出交接信息（已完成项、未完成项、风险、下一步、关联工件）。

---

## 5. 仓库结构约定

采用 monorepo，结构如下：

```
lingchoubot/
├── AGENTS.md                    # 本文档
├── docker-compose.yml
├── docs/                        # 设计文档
├── backend/                     # Go 后端
│   ├── cmd/api/                 # 入口
│   ├── internal/
│   │   ├── config/              # 配置
│   │   ├── model/               # 数据模型
│   │   ├── repository/          # 数据访问层
│   │   ├── service/             # 业务逻辑层（状态流转集中在此）
│   │   ├── handler/             # HTTP handler
│   │   └── middleware/          # 中间件
│   ├── migrations/              # SQL migration 文件
│   └── go.mod
└── frontend/                    # Next.js 前端
    ├── src/
    └── package.json
```

**规则：**
1. 不要随意新增顶层目录
2. 新模块必须放入合理位置
3. migration 必须单独存放在 `backend/migrations/`
4. 文档必须放到 `docs/`

---

## 6. 数据库规范

### 6.1 Migration 规范
- 优先使用纯 SQL migration
- 每次 schema 变更必须有 migration
- migration 必须可重复执行或明确不可逆说明
- 不允许手工改库而不落 migration

### 6.2 表设计原则
- 主键统一使用 UUID
- 时间统一使用 `timestamptz`
- 核心状态字段优先 `text + CHECK constraint`
- JSONB 只用于半结构化元数据，不替代核心关系模型

### 6.3 核心表清单

以下表的修改必须谨慎，需要先确认 migration 方案：

| 表名 | 用途 |
|------|------|
| `project` | 项目 |
| `project_phase` | 项目阶段 |
| `agent` | Agent 注册信息 |
| `task` | 任务 |
| `task_contract` | 任务契约 |
| `task_assignment` | 任务分派记录 |
| `artifact` | 工件 |
| `artifact_version` | 工件版本 |
| `review_report` | 独立评审报告 |
| `handoff_snapshot` | 交接快照 |
| `approval_request` | 审批请求 |
| `audit_log` | 审计日志 |
| `tool_call` | 工具调用记录 |

### 6.4 禁止行为
- 不要直接删除核心字段
- 不要随意改动已有状态语义
- 不要把大量核心业务字段塞进 JSONB
- 不要绕过 migration 修改 schema

---

## 7. API 设计规范

### 7.1 统一返回格式

成功：
```json
{
  "success": true,
  "data": {},
  "error": null
}
```

失败：
```json
{
  "success": false,
  "data": null,
  "error": {
    "code": "SOME_ERROR",
    "message": "human readable error"
  }
}
```

### 7.2 API 实现顺序

按以下顺序实现，不要跳跃：

1. project CRUD
2. phase CRUD
3. agent CRUD
4. task CRUD
5. task_contract create/update/query
6. task_assignment create/query
7. artifact create/query + artifact_version
8. review_report create/query
9. handoff_snapshot create/query
10. approval_request create/decision
11. audit_log timeline query

---

## 8. 代码实现原则

### 8.1 不要过早抽象
MVP 阶段优先清晰、可读、可运行、易测试。不要为"未来可能扩展"过度设计。

### 8.2 不要把所有东西做成框架
优先完成具体业务闭环，再提炼通用层。

### 8.3 优先显式代码
优先显式结构体、显式接口、显式状态常量、显式 SQL。避免过多魔法和隐式行为。

### 8.4 状态流转必须集中管理
不要把任务状态修改分散在多个无约束函数里。状态流转逻辑必须统一放在 `service/` 层。

---

## 9. 审计规范

以下动作必须写入 `audit_log`：

- 项目/阶段/任务创建
- 任务分派与状态变化
- 任务契约更新
- 工件创建与版本增加
- 交接快照创建
- 评审报告创建
- 审批发起与决策
- 工具调用

审计日志字段：`actor_type`, `actor_id`, `event_type`, `event_summary`, `target_type`, `target_id`, `before_state`, `after_state`, `created_at`

---

## 10. 测试规范

### 10.1 关键路径必须有测试

MVP 阶段优先覆盖：

1. 项目/阶段/任务 CRUD
2. 任务状态流转校验
3. 任务契约创建/更新
4. 任务分派逻辑
5. 工件版本递增
6. 审批状态流转
7. 审计日志写入

### 10.2 不要求高覆盖率
但关键主链路必须有测试，尤其是状态流转和权限校验。

---

## 11. 前端实现规范

### 11.1 先做信息结构，不先做视觉炫技
优先保证数据真实、页面结构清晰、状态可读、操作可走通。

### 11.2 MVP 必备页面
- 项目列表页
- 项目详情页（含阶段、最近任务、最近工件、待审批项）
- 任务看板页（按状态分栏）
- Agent 组织树页
- 工件列表页
- 审批中心页
- 审计时间线页

### 11.3 禁止行为
- 不要先做大量静态假数据页面
- 不要为了 UI 复杂度拖慢核心联调
- 前端页面必须优先消费真实 API

---

## 12. 工作方式要求

### 12.1 一次只处理一个明确工作包
不要一次性同时大改数据库、后端、前端、Docker，除非任务明确要求。

### 12.2 输出必须包含以下内容
每次完成任务后，必须说明：
1. 改了哪些文件
2. 为什么这样改
3. 有哪些假设
4. 还有哪些未完成
5. 如何验证

### 12.3 不得擅自重构无关模块
如果任务是"实现 task CRUD"，不要顺手大改前端路由、Docker 文件、其他 service。

### 12.4 发现设计冲突时先最小修正
- 先做最小兼容实现
- 在结果中明确指出冲突点
- 不要擅自推翻整体设计

---

## 13. 推荐开发顺序

如果没有额外指示，默认按以下顺序推进：

| 步骤 | 内容 |
|------|------|
| Step 1 | 初始化 monorepo + Docker Compose + PostgreSQL + migration 框架 |
| Step 2 | 落核心表 + 跑通 migration + 建立 repository 层 |
| Step 3 | 实现 project / phase / agent / task CRUD + task_contract + audit_log |
| Step 4 | 实现 task_assignment + 组织树查询 + 最小 supervisor->worker 链路 |
| Step 5 | 实现 artifact / artifact_version + review_report + approval_request |
| Step 6 | 实现 Tool Gateway + handoff_snapshot + 最小工作流编排 |
| Step 7 | 实现 Web 控制台核心页面 |
| Step 8 | 端到端 Demo + 缺陷修复 + 演示文档 |

---

## 14. 当前禁止优先做的事项

在没有明确指示前，不要优先投入：

- 多租户
- 复杂 SSO / RBAC
- 高级知识库 / 向量数据库
- NATS 事件总线
- Temporal 深度接入
- 复杂事件总线编排
- 高拟真 Agent 社会模拟
- 生产级 Kubernetes 部署
- 花哨 UI 动效

---

## 15. 文档同步要求

任何影响以下内容的改动，必须同步更新 `docs/`：

- 架构变化
- 表结构变化
- API 变化
- 目录结构变化
- Agent 协议变化
- 审批/权限策略变化

---

## 16. 完成定义（Definition of Done）

一个工作包只有满足以下条件才算完成：

1. 代码已实现
2. 能本地运行或可验证
3. 必要测试已补
4. 文档已同步
5. 未越界修改大量无关内容
6. 输出了清晰的变更说明

---

## 17. 冲突处理

如果任务描述与本文档冲突：

1. 先遵循明确的用户任务
2. 其次遵循架构文档（`docs/001`）
3. 若无法确定，采用最小改动、最易回滚、最便于审计的实现方式

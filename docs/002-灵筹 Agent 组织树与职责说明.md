# 《灵筹 Agent 组织树与职责说明》V0.2

## 1. 文档信息

| 字段 | 值 |
|------|------|
| 文档名称 | 灵筹 Agent 组织树与职责说明 |
| 项目代号 | 灵筹（LingChou） |
| 文档版本 | V0.2 |
| 文档状态 | Draft |
| 面向对象 | 系统架构设计、运行时实现、Prompt/Policy 配置 |
| 关联文档 | `001-灵筹系统总体架构设计`、`003-MVP 开发路线图` |

---

## 2. 文档目标

定义灵筹系统中的 Agent 组织结构、汇报关系、职责边界、输入输出契约、权限范围与协作规则。

---

## 3. 角色设计原则

| 编号 | 原则 | 说明 |
|------|------|------|
| 1 | 单一责任 | 每个 Agent 有明确主责，不允许高度重叠 |
| 2 | 主管负责制 | 执行 Agent 不直接向用户汇报，先提交给主管 |
| 3 | 输出优先 | 职责最终收敛为结构化输出或工件 |
| 4 | 可替换 | 任意 Agent 可被同类 Agent 替代 |
| 5 | 权限最小化 | 只持有完成当前职责所需的最小权限 |
| 6 | 升级优先 | 目标不清、输入不足、风险过高时向上升级，不自行猜测 |
| 7 | 任务契约先行 | 执行前必须先明确范围、验收标准、验证步骤 |
| 8 | 独立评审 | 执行 Agent 自检不作为最终验收，关键任务须独立评审 |

---

## 4. 组织层级

灵筹采用四层组织结构：

```text
第 0 层：用户（最终审批人）

第 1 层：项目负责人 Agent（PM）
  └── 负责项目级规划、协调、汇总、对外汇报

第 2 层：部门主管 Agent（Supervisor）
  ├── 产品主管
  ├── 架构主管
  ├── 开发主管
  ├── 测试主管
  └── 发布主管

第 3 层：执行 Agent（Worker）
  ├── PRD Agent
  ├── API 设计 Agent
  ├── 数据模型 Agent
  ├── 后端开发 Agent
  ├── 前端开发 Agent
  ├── 单元测试 Agent
  ├── 集成测试 Agent
  └── 部署 Agent
```

### MVP 最小角色集

MVP 阶段不需要一次实现所有角色。建议分两批：

**第一批（MVP 核心，必须实现）：**
- 项目负责人 Agent
- 开发主管 Agent
- 测试主管 Agent
- 后端开发 Agent
- 单元测试 Agent

**第二批（MVP 扩展，按需实现）：**
- 产品主管 Agent + PRD Agent
- 架构主管 Agent + API 设计 Agent + 数据模型 Agent
- 前端开发 Agent
- 集成测试 Agent
- 发布主管 Agent + 部署 Agent

**MVP 后再考虑：**
- 用户流程 Agent
- 重构 Agent
- 回归测试 Agent
- 设计主管 Agent
- 安全主管 Agent
- 文档主管 Agent

### 当前系统默认基线 Agent

为降低初次启动与联调成本，当前后端 API 服务启动时会自动补齐一组最小可运行的基础 Agent。补齐逻辑按 `role_code` 幂等执行：

- 已存在对应 `role_code` 的 Agent 时不重复创建
- 仅补齐缺失角色
- 自动建立最小汇报链路 `reports_to`

当前默认基线角色如下：

| Agent | role_code | 上级 | 用途 |
|------|------|------|------|
| PM Agent | `PM_SUPERVISOR` | 无 | 项目级规划、协调与汇总 |
| Development Supervisor | `DEVELOPMENT_SUPERVISOR` | `PM_SUPERVISOR` | 统筹开发域任务分派与返工协调 |
| QA Supervisor | `QA_SUPERVISOR` | `PM_SUPERVISOR` | 统筹质量门、测试与评审 |
| Backend Worker | `BACKEND_DEV_WORKER` | `DEVELOPMENT_SUPERVISOR` | 负责后端实现 |
| Frontend Worker | `FRONTEND_DEV_WORKER` | `DEVELOPMENT_SUPERVISOR` | 负责前端实现 |
| QA Worker | `QA_WORKER` | `QA_SUPERVISOR` | 负责测试执行与回归验证 |
| Reviewer Agent | `REVIEWER_WORKER` | `QA_SUPERVISOR` | 负责独立评审 |

这组基线角色用于保证 MVP 工作流在开箱状态下具备最小可运行组织树；后续如需更细分角色，可在此基础上继续扩展，而不是替代现有基线启动逻辑。

---

## 5. 角色分类

### 5.1 Human Role

由真实用户承担：最终审批人、项目发起人。

### 5.2 Supervisor Agent

负责组织、拆解、验收、协调、升级。关注阶段目标和任务边界，不直接深度执行细节任务。

### 5.3 Worker Agent

负责具体交付物生成。职责聚焦，输入输出明确，不负责项目级方向判断。

### 5.4 独立评审模式

灵筹只区分 `supervisor` 与 `worker` 两种 Agent 类型，但在执行模式上区分：
- **生成/执行角色**：产出代码、文档、方案等交付物
- **评审/验证角色**：独立检查交付物是否满足任务契约

测试相关 Worker 默认属于评审侧。主管 Agent 决定谁来执行、谁来验收，不允许同一执行 Agent 自行宣布"验收通过"。

---

## 6. 通用协议

### 6.1 Agent 基本身份字段

```yaml
agent_id: string          # 唯一标识
agent_name: string        # 显示名称
agent_type: supervisor | worker
role_code: string         # 角色编码
reports_to: string | null # 上级 agent_id
managed_roles: string[]   # 管理的下级角色编码（supervisor 专属）
capabilities: string[]    # 能力标签
allowed_tools: string[]   # 允许调用的工具
risk_level: low | medium | high
```

### 6.2 通用输入结构

Agent 接收任务时的统一输入包：

```yaml
task_id: string
project_id: string
phase_id: string
goal: string                    # 任务目标
context_summary: string         # 上下文摘要
input_artifacts:                # 输入工件
  - artifact_id: string
    artifact_type: string
    uri: string
acceptance_criteria: string[]   # 验收标准
tool_permissions: string[]      # 允许使用的工具
task_contract:                  # 任务契约
  scope: string                 # 本轮范围
  non_goals: string[]           # 非目标
  done_definition: string[]     # 完成定义
  verification_plan: string[]   # 验证步骤
escalation_policy:              # 升级策略
  supervisor_agent_id: string
  must_escalate_when: string[]
```

> **MVP 简化说明：** `budget`（max_model_calls / max_tokens / max_runtime_seconds）、`deadline` 等字段在 MVP 阶段暂不实现强制执行，可作为 JSONB 元数据记录但不做运行时校验。

### 6.3 通用输出结构

Agent 结束任务时的统一输出：

```yaml
task_id: string
agent_id: string
status: success | blocked | needs_revision | failed | escalated
summary: string
produced_artifacts:
  - artifact_id: string
    artifact_type: string
    uri: string
completion_evidence: string[]   # 完成证据
risks: string[]
recommended_next_actions: string[]
handoff_artifacts:              # 交接工件
  - artifact_id: string
    artifact_type: string
    uri: string
```

### 6.4 必须升级的情形

任意 Agent 遇到以下情况必须升级，不得自行猜测执行：

1. 任务目标不明确
2. 关键输入工件缺失或冲突
3. 需要调用未授权工具
4. 需要写入未授权仓库或环境
5. 涉及数据库破坏性操作
6. 涉及生产环境发布
7. 涉及外部通信
8. 任务输出无法满足验收标准

### 6.5 所有 Agent 的禁止行为

1. 伪造完成状态
2. 绕过主管链路直接修改高风险目标
3. 绕过审批流执行受限动作
4. 在输入不足时虚构事实或工件
5. 擅自访问不属于当前项目的上下文
6. 将"建议"冒充为"已完成交付物"
7. 无限递归拆分任务
8. 未授权调用联网工具

---

## 7. 角色详细定义

### 7.1 用户（最终审批人）

| 字段 | 值 |
|------|------|
| role_code | `HUMAN_FINAL_APPROVER` |
| role_type | `human` |
| 上级 | 无 |
| 下级 | 项目负责人 Agent |

**使命：** 定义项目目标与约束，对高风险动作进行最终审批，验收最终交付物。

**核心职责：** 创建项目、给出目标与限制条件、审批关键阶段结果、审批高风险动作、验收最终交付物。

---

### 7.2 项目负责人 Agent

| 字段 | 值 |
|------|------|
| role_code | `PM_SUPERVISOR` |
| agent_type | `supervisor` |
| reports_to | `HUMAN_FINAL_APPROVER` |
| managed_roles | 所有部门主管 |

**使命：** 将用户目标转化为可执行项目计划，协调各部门主管完成交付，以阶段为单位向用户汇报。

**核心职责：**
1. 解析用户需求，生成项目章程
2. 划分阶段与里程碑
3. 组织部门主管协同
4. 创建项目级任务树
5. 跟踪风险、依赖与阻塞
6. 汇总部门成果，发起高风险审批
7. 输出阶段汇报与最终交付总结

**输出工件：** 项目章程、阶段计划、任务树、风险清单、阶段汇报、最终交付总结

**禁止：** 不直接编写底层代码、不绕过主管链条大规模派单、不跳过测试与发布宣告完成

**升级条件：** 目标重大变化、范围显著膨胀、预算/时间超限、需要生产发布审批

---

### 7.3 部门主管 Agent（通用模板）

所有部门主管共享以下基础职责，各自在专业领域有不同侧重：

| 字段 | 值 |
|------|------|
| agent_type | `supervisor` |
| reports_to | `PM_SUPERVISOR` |

**通用职责：**
1. 理解本部门在当前阶段的目标
2. 拆解子任务并分派给执行 Agent
3. 审核执行 Agent 的产出
4. 汇总本部门结果
5. 向项目负责人汇报
6. 在风险或阻塞时升级

**通用禁止：** 不绕过项目负责人自行改变项目方向、不跳过独立评审宣告可发布

#### 各主管差异化定义

| 主管 | role_code | managed_roles | 专业侧重 | 关键输出 |
|------|-----------|---------------|----------|----------|
| 产品主管 | `PRODUCT_SUPERVISOR` | PRD Worker | 需求边界、用户路径、功能优先级 | PRD、功能列表、验收口径 |
| 架构主管 | `ARCHITECTURE_SUPERVISOR` | API 设计 Worker、数据模型 Worker | 模块划分、接口契约、数据模型 | 架构设计文档、API 契约、数据模型 |
| 开发主管 | `DEVELOPMENT_SUPERVISOR` | 后端开发 Worker、前端开发 Worker | 任务拆解、代码质量、分支策略 | 开发任务清单、代码提交结果 |
| 测试主管 | `QA_SUPERVISOR` | 单元测试 Worker、集成测试 Worker | 测试策略、质量门、独立评审 | 测试计划、测试报告、评审结论 |
| 发布主管 | `RELEASE_SUPERVISOR` | 部署 Worker | 构建检查、部署方案、回滚方案 | 发布计划、部署记录 |

---

### 7.4 执行 Agent（Worker）

所有执行 Agent 共享以下基础约束：

| 字段 | 值 |
|------|------|
| agent_type | `worker` |
| reports_to | 对应部门主管 |

**通用约束：**
- 接受任务后按任务契约执行
- 产出结构化工件
- 遇到问题向主管升级
- 不直接决定项目方向
- 不直接合并主干或触发生产部署

#### 各执行 Agent 定义

| Agent | role_code | 上级 | 使命 | 关键输出 | 特殊禁止 |
|-------|-----------|------|------|----------|----------|
| PRD Agent | `PRD_WORKER` | 产品主管 | 输出结构化 PRD | PRD 草案、需求疑问清单 | 不定义技术选型 |
| API 设计 Agent | `API_DESIGN_WORKER` | 架构主管 | 产出接口定义 | OpenAPI 文档、接口约束 | 不直接修改数据库 |
| 数据模型 Agent | `DATA_MODEL_WORKER` | 架构主管 | 设计数据模型与迁移方案 | 数据模型文档、SQL/Migration 草案 | 不直接执行生产迁移 |
| 后端开发 Agent | `BACKEND_DEV_WORKER` | 开发主管 | 实现后端逻辑与数据访问 | 后端代码、单元测试、变更说明 | 不直接合并主干 |
| 前端开发 Agent | `FRONTEND_DEV_WORKER` | 开发主管 | 实现前端页面与交互 | 前端代码、交互说明 | 不直接更改后端接口契约 |
| 单元测试 Agent | `UNIT_TEST_WORKER` | 测试主管 | 验证单模块行为正确性 | 测试用例、测试结果、评审结论 | — |
| 集成测试 Agent | `INTEGRATION_TEST_WORKER` | 测试主管 | 验证多模块协同正确性 | 集成测试结果、跨模块问题清单 | — |
| 部署 Agent | `DEPLOY_WORKER` | 发布主管 | 按部署计划执行受控部署 | 部署结果、健康检查结果 | 不在无审批状态下部署生产 |

---

## 8. 汇报与升级链路

### 8.1 默认汇报链

```text
执行 Agent -> 部门主管 Agent -> 项目负责人 Agent -> 用户
```

### 8.2 禁止的跨级直连

1. 执行 Agent 直接向用户宣告完成
2. 执行 Agent 直接绕过主管改项目范围
3. 任意 Agent 绕过项目负责人直接协调多个主管级任务
4. 未授权 Agent 直接操作发布链路

---

## 9. 权限模型

### 9.1 项目负责人权限

```yaml
can_create_project_tasks: true
can_assign_supervisors: true
can_approve_release: false        # 需要用户审批
can_request_human_approval: true
can_write_repo: false
can_read_all_project_artifacts: true
```

### 9.2 主管 Agent 权限

```yaml
can_create_subtasks: true
can_review_worker_outputs: true
can_merge_department_outputs: true
can_approve_high_risk_actions: false
can_write_repo: false
can_read_department_artifacts: true
```

### 9.3 执行 Agent 权限

```yaml
can_create_subtasks: limited      # 仅限当前任务内
can_write_repo: scoped            # 仅限授权分支
can_run_tests: scoped
can_trigger_ci: false
can_trigger_prod_deploy: false
can_read_project_context: scoped  # 仅限当前任务相关
```

---

## 10. 实现要求

### 10.1 不要把 Agent 实现成自由聊天角色

每个 Agent 至少要绑定：
- `role_code` + `system_prompt_template`
- `input_schema` + `output_schema`
- `permission_profile` + `escalation_rules`
- `allowed_tools`
- `managed_roles`（supervisor 专属）

### 10.2 Agent 执行结果必须结构化

所有 Agent 调用结束时，必须返回结构化 JSON，不接受仅有自然语言段落作为唯一结果。

### 10.3 Supervisor 与 Worker 必须有不同行为模式

- **Supervisor**：理解目标、拆任务、审核、汇总、决策是否升级
- **Worker**：接受任务、产出工件、报告风险、请求升级

两者不能共用同一套默认行为。

### 10.4 Agent 必须受任务边界约束

- Agent 看不到不需要的项目上下文
- Agent 不能随意读取所有工件
- Agent 不能自行扩大任务范围
- Agent 的工具访问必须经过权限模板校验

---

## 11. 可选扩展角色（MVP 后）

以下角色建议在 MVP 之后引入：

| 角色 | 职责 |
|------|------|
| 用户流程 Agent | 将需求转换为用户流程与异常流程说明 |
| 重构 Agent | 在不改变对外行为前提下优化代码结构 |
| 回归测试 Agent | 验证新增变更未破坏历史功能 |
| CI Agent | 完成构建、校验、打包 |
| 设计主管 Agent | 管理 UI 设计、设计系统、交互规范 |
| 安全主管 Agent | 管理安全审查、审核敏感权限 |
| 文档主管 Agent | 管理文档标准、汇总技术/运维/用户文档 |

# 045-PRD任务校验误判与Compose验证总结

## 变更摘要

本次修复了 worker 输出校验中的任务类型误判问题：当需求梳理、PRD 编写类任务描述中包含“验证”“验收”等词时，旧逻辑会同时把任务识别为 analysis task 和 testing task，进而错误要求输出 `test_report`，导致真实 LLM 已经产出 `prd` 仍被判定失败。

修复后，analysis 类关键词优先生效；只有未被识别为 analysis task 的任务，才会因 testing 类关键词被强制要求输出 `test_report`。同时在 Docker Compose 运行态完成了回归验证，确认此前卡死在 `waiting_manual_intervention` 的 run 已被解除阻塞，PRD 工件能够正常落库。

## 文件清单

- 修改：`backend/internal/runtime/llm_validator.go`
- 修改：`backend/internal/runtime/llm_validator_test.go`

## 关键实现说明

### 1. 修正任务分类优先级

旧逻辑仅依据 `Task.Title + Task.Description` 的关键词做并行判定：

- 命中 analysis 关键词时要求产出 `prd/design`
- 命中 testing 关键词时要求产出 `test_report`

因为 testing 关键词包含“验证”，而需求/PRD 类任务常会出现“验证需求”“验收条件”等表述，所以会产生双命中，导致 worker 输出即使是合法 `prd` 也会被错误拦截。

本次最小修正做了两点：

1. 扩充 analysis 关键词，补入“计划”“梳理”“编写”“文档”“设计”等更贴近 PRD/方案类任务的信号。
2. testing 规则只在“未命中 analysis task”时生效，避免分析型文档任务被“验证”一词误伤。

### 2. 增加回归测试

新增了两个关键测试场景：

1. `需求梳理与PRD编写` 这类描述中包含“验证需求”的分析任务，输出 `prd` 时应通过校验。
2. `SSE修复验证` 这类非分析型验证任务，若没有 `test_report`，仍应被拦截。

这样既修复了当前 compose 阻塞点，也保留了原本对 QA/测试任务的约束。

### 3. Compose 运行态验证结果

重建并重启 compose `api` 服务后，对此前卡在人工介入状态的 run `842a1dba-e246-4159-9d64-ac1abb73a8cd` 执行 `resume`，观察到：

1. 旧的 worker 失败步骤仍保留在历史中。
2. 新触发的 worker 步骤已成功完成 `需求梳理与PRD编写`。
3. 新的 `prd` 工件已成功写入数据库。
4. run 已从原来的 `waiting_manual_intervention` 继续向后推进。

这证明当前修复已经在真实 compose 运行态生效，而不仅仅是单元测试通过。

### 4. 关于 5432 的异常登录噪音

额外排查确认：

- 仓库代码中不存在 `wanglixiang` 作为数据库用户配置。
- compose 内 API/worker/migrate 使用的数据库用户仍为 `lingchou`。
- Postgres 容器日志显示该异常登录稳定以 30 秒周期出现，属于外部本机短连接，而不是当前仓库服务自身的重试逻辑。

当前最强嫌疑是另一个 VS Code workspace 的后台 Java/NetBeans 类扩展进程（尤其是 `oracle.oracle-java` 拉起的 `nbcode64.exe`），但本次未直接捕获到“具体 PID -> 5432 短连接”的瞬时映射，因此暂未做侵入式停进程验证。

## 运行方式

1. 在 `backend` 目录执行：`go test ./...`
2. 在仓库根目录执行：`docker compose build api && docker compose up -d api`
3. 对目标 workflow run 调用：`POST /api/v1/orchestrator/runs/{id}/resume`
4. 通过 `GET /api/v1/orchestrator/runs/{id}` 与 `GET /api/v1/artifacts` 观察步骤推进与工件落库

## 测试方式

- 单元测试：`go test ./internal/runtime`
- 后端全量测试：`go test ./...`
- 运行态验证：
  - 恢复 run `842a1dba-e246-4159-9d64-ac1abb73a8cd`
  - 确认新增 worker 步骤完成 `需求梳理与PRD编写`
  - 确认新项目 `7bc3c8bc-6ff3-4a4d-84d4-a0705b0137cf` 已新增 `prd` 工件

## 已知限制

1. 任务分类仍主要依赖标题和描述关键词，尚未升级为基于结构化任务意图或契约字段的完整判定。
2. `wanglixiang` 的 5432 异常登录已确认不来自当前仓库运行链路，但尚未 100% 锁定到单一宿主机进程。
3. 当前验证只证明本次 PRD 任务误判已解除；后续更深层次的 reviewer/revision 闭环仍应继续观察真实 LLM 输出质量。
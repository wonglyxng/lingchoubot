# 031 Agent role_code 唯一约束实现总结

## 1. 变更摘要

本次为 `agent.role_code` 增加了数据库唯一约束，并在服务层与接口层补齐冲突校验，避免同一组织角色被重复创建。

本次改动完成了三个层面的闭环：

- 数据库层：为非空 `role_code` 建立唯一索引
- 服务层：创建和更新 Agent 时主动检测冲突
- 接口层：对重复 `role_code` 返回明确的 `409 ROLE_CODE_CONFLICT`

## 2. 文件清单

新增文件：

- `backend/internal/handler/agent_test.go`
- `backend/migrations/000014_agent_role_code_unique.up.sql`
- `backend/migrations/000014_agent_role_code_unique.down.sql`
- `docs/031-Agent-role_code唯一约束实现总结.md`

修改文件：

- `backend/internal/service/agent.go`
- `backend/internal/service/repo_interfaces.go`
- `backend/internal/repository/agent.go`
- `backend/internal/handler/agent.go`
- `backend/internal/service/agent_bootstrap_test.go`
- `backend/internal/testutil/testutil.go`
- `docs/004-数据库设计.md`
- `docs/005-API 接口文档.md`

## 3. 关键实现说明

### 3.1 只约束非空 role_code

`agent.role_code` 早期 migration 默认值是空字符串。为了兼容已有数据，本次唯一索引采用部分索引，只约束 `role_code <> ''` 的记录：

- 现有空值数据不会因加索引直接失败
- 已显式设置的组织角色编码必须唯一

### 3.2 迁移前先检查脏数据

`000014_agent_role_code_unique.up.sql` 在创建唯一索引前，会先扫描是否存在重复的非空 `role_code`。

如果存在重复值，迁移会直接失败，并给出重复的 `role_code` 列表，避免静默失败或只抛出难读的底层唯一键异常。

### 3.3 服务层主动校验并兼容并发场景

`AgentService` 新增了两层保护：

- 在写入前主动按 `role_code` 查询冲突
- 如果并发写入仍然撞上数据库唯一索引，再把底层唯一键异常转换成统一的业务错误

这样既能在正常路径下给出清晰提示，也能在竞态条件下保持行为一致。

### 3.4 接口层返回 409 冲突

`AgentHandler` 现在会把 `role_code` 冲突映射为：

- HTTP `409`
- error code `ROLE_CODE_CONFLICT`

这比此前把所有写入失败都归到 `400 CREATE_FAILED/UPDATE_FAILED` 更准确，也更方便前端做差异化提示。

## 4. 运行方式

升级数据库迁移后，按原有方式启动服务即可：

```bash
cd backend
go run ./cmd/api
```

或：

```bash
docker compose up --build
```

若数据库中已存在重复的非空 `role_code`，迁移会报错并阻止继续升级，需要先清理重复数据后再执行迁移。

## 5. 测试方式

本次建议执行：

```bash
cd backend
go test ./...
go build ./cmd/api
```

新增覆盖点包括：

1. 服务层创建重复 `role_code` 会返回冲突错误
2. 服务层更新为重复 `role_code` 会返回冲突错误
3. Agent 创建接口对重复 `role_code` 返回 `409 ROLE_CODE_CONFLICT`
4. Agent 更新接口对重复 `role_code` 返回 `409 ROLE_CODE_CONFLICT`

## 6. 已知限制

- 当前唯一约束只覆盖非空 `role_code`，空字符串仍允许存在多条记录。
- 迁移在发现重复 `role_code` 时会直接失败，但不会自动清洗旧数据。
- 如果未来要把 `role_code` 变成真正必填字段，还需要再补一轮数据治理与非空约束迁移。
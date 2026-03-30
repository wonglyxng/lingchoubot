# 016 - Repository 层关键测试补充总结

## 变更摘要

为 repository 层补充了关键单元测试，覆盖 workflow_run、workflow_step、approval_request、task 四个核心仓库的 CRUD 与状态更新操作。使用 go-sqlmock 进行数据库层 mock，无需真实数据库即可验证 SQL 查询构建和结果扫描逻辑。

## 文件清单

### 新增文件
- `backend/internal/repository/workflow_test.go` — WorkflowRunRepo + WorkflowStepRepo 测试（12 个测试用例）
- `backend/internal/repository/approval_request_test.go` — ApprovalRequestRepo 测试（6 个测试用例）
- `backend/internal/repository/task_test.go` — TaskRepo 测试（9 个测试用例）

### 修改文件
- `backend/go.mod` / `backend/go.sum` — 新增 `github.com/DATA-DOG/go-sqlmock v1.5.2` 依赖

## 关键实现说明

### 测试策略
- 采用 `go-sqlmock` 对 `database/sql` 层进行 mock，验证 SQL 语句构建、参数传递、结果扫描是否正确
- 每个 Repository 方法至少覆盖正常路径和关键异常路径（如 NotFound、AlreadyDecided）

### 覆盖范围

| Repository | 方法 | 测试用例 |
|---|---|---|
| WorkflowRunRepo | Create, GetByID, UpdateStatus, List | 7 个（含 NotFound、带过滤条件 List） |
| WorkflowStepRepo | Create, UpdateStatus, ListByRunID | 5 个（含 NotFound、空列表） |
| ApprovalRequestRepo | Create, GetByID, List, Decide | 6 个（含 NotFound、AlreadyDecided） |
| TaskRepo | Create, GetByID, List, UpdateStatus, Update, Delete | 9 个（含 NotFound、多条件过滤 List） |

**总计：27 个新增测试用例，全部通过。**

## 运行方式

```bash
cd backend

# 仅运行 repository 层测试
go test ./internal/repository/... -count=1 -v

# 全量后端测试
go test ./internal/... -count=1
```

## 测试方式

所有测试均为纯单元测试，使用 go-sqlmock 模拟数据库，无需启动 PostgreSQL。

```bash
go test ./internal/repository/... -count=1 -v
```

预期输出：27 个 PASS，0 个 FAIL。

## 已知限制

- go-sqlmock v1 不支持 `NamedArgs` 和 `context` 级别 mock，但对当前 positional args 模式完全够用
- 未覆盖复杂并发场景（如同时写入冲突），这类场景需集成测试配合真实数据库验证
- 其他 repository（agent、artifact、audit 等）暂未补测试，可后续按需增补

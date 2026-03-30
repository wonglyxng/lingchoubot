package runtime

import (
	"crypto/sha256"
	"fmt"
)

// MockQAWorkerAgent simulates a QA-specialized worker.
// It produces test plans and test report artifacts.
type MockQAWorkerAgent struct{}

func (a *MockQAWorkerAgent) Role() string           { return "worker" }
func (a *MockQAWorkerAgent) Specialization() string { return "qa" }

func (a *MockQAWorkerAgent) Execute(input *AgentTaskInput) (*AgentTaskOutput, error) {
	if input.Task == nil {
		return &AgentTaskOutput{Status: OutputStatusFailed, Error: "qa worker requires task context"}, nil
	}

	task := input.Task
	content := fmt.Sprintf(`# 测试报告：%s

## 1. 测试范围
- 任务：%s
- 描述：%s

## 2. 测试用例

| 编号 | 场景 | 预期结果 | 实际结果 | 状态 |
|------|------|---------|---------|------|
| TC-001 | 正常流程 | 功能按预期工作 | 符合预期 | PASS |
| TC-002 | 边界条件 | 正确处理边界值 | 符合预期 | PASS |
| TC-003 | 异常输入 | 返回合适错误信息 | 符合预期 | PASS |
| TC-004 | 并发场景 | 无数据竞争 | 待验证 | PENDING |

## 3. 测试结论
- 通过率：75%% (3/4)
- 风险项：并发场景尚需验证
- 建议：可进入评审阶段，并发测试后续补充

由 Mock QA Worker Agent 自动生成。
`, task.Title, task.Title, task.Description)

	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))

	return &AgentTaskOutput{
		Status:  OutputStatusNeedsReview,
		Summary: fmt.Sprintf("QA 任务「%s」已完成：测试报告已生成，通过率 75%%", task.Title),
		Artifacts: []ArtifactAction{
			{
				Name:         fmt.Sprintf("%s-测试报告", task.Title),
				ArtifactType: "test_report",
				Description:  fmt.Sprintf("任务「%s」的测试计划与执行报告", task.Title),
				URI:          fmt.Sprintf("mock://artifacts/%s/test_report", task.ID),
				ContentType:  "text/markdown",
				SizeBytes:    int64(len(content)),
				Content:      content,
				Metadata:     map[string]any{"checksum": checksum, "generator": "mock_qa_worker", "pass_rate": 0.75},
			},
		},
		Handoffs: []HandoffAction{
			{
				Summary:        fmt.Sprintf("QA 任务「%s」执行完成", task.Title),
				CompletedItems: []string{"测试计划已制定", "3/4 测试用例已执行通过"},
				PendingItems:   []string{"并发场景测试待补充", "等待独立评审"},
				Risks:          []string{"并发场景覆盖不足"},
				NextSteps:      []string{"补充并发测试", "提交评审"},
			},
		},
		Transitions: []TransitionAction{
			{TaskTitle: task.Title, NewStatus: "in_review"},
		},
	}, nil
}

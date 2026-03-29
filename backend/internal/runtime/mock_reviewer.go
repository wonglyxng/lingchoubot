package runtime

import "fmt"

// MockReviewerAgent simulates a Reviewer agent.
// It evaluates task artifacts and produces a review report.
type MockReviewerAgent struct{}

func (a *MockReviewerAgent) Role() string { return "reviewer" }

func (a *MockReviewerAgent) Execute(input *AgentTaskInput) (*AgentTaskOutput, error) {
	if input.Task == nil {
		return &AgentTaskOutput{
			Status: OutputStatusFailed,
			Error:  "reviewer agent requires task context",
		}, nil
	}

	task := input.Task
	artCount := len(input.Artifacts)

	reviews := []ReviewAction{
		{
			Verdict: "approved",
			Summary: fmt.Sprintf("任务「%s」的 %d 个交付物已通过评审", task.Title, artCount),
			Findings: []string{
				"交付物内容覆盖任务描述中的核心要求",
				"交付物格式规范，结构清晰",
				"交接信息完整，无遗漏关键项",
			},
			Recommendations: []string{
				"建议后续迭代中补充更详细的验证用例",
				"建议关注交付物与上下游任务的一致性",
			},
		},
	}

	return &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: fmt.Sprintf("任务「%s」评审通过", task.Title),
		Reviews: reviews,
	}, nil
}

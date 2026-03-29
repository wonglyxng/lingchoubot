package runtime

import "fmt"

// MockSupervisorAgent simulates a Supervisor agent.
// It creates task contracts and plans worker assignments.
type MockSupervisorAgent struct{}

func (a *MockSupervisorAgent) Role() string { return "supervisor" }

func (a *MockSupervisorAgent) Execute(input *AgentTaskInput) (*AgentTaskOutput, error) {
	if input.Task == nil {
		return &AgentTaskOutput{
			Status: OutputStatusFailed,
			Error:  "supervisor agent requires task context",
		}, nil
	}

	task := input.Task

	contracts := []ContractAction{
		{
			TaskTitle: task.Title,
			Scope:     fmt.Sprintf("完成「%s」的全部工作内容", task.Title),
			NonGoals: []string{
				"不涉及超出本任务范围的功能扩展",
				"不处理与本任务无关的技术债务",
			},
			DoneDefinition: []string{
				fmt.Sprintf("「%s」的核心交付物已生成", task.Title),
				"交付物通过格式与内容基本检查",
				"交接快照已创建",
			},
			VerificationSteps: []string{
				"检查交付物是否完整覆盖任务描述中的要求",
				"验证交付物格式是否符合项目规范",
				"确认交接快照中无遗漏风险项",
			},
			AcceptanceCriteria: []string{
				"交付物内容与任务描述一致",
				"独立评审结论为 approved",
			},
		},
	}

	assignments := []AssignmentAction{
		{
			TaskTitle: task.Title,
			AgentRole: "worker",
			Role:      "executor",
			Note:      fmt.Sprintf("分派执行「%s」", task.Title),
		},
	}

	transitions := []TransitionAction{
		{TaskTitle: task.Title, NewStatus: "assigned"},
	}

	return &AgentTaskOutput{
		Status:      OutputStatusSuccess,
		Summary:     fmt.Sprintf("已为任务「%s」创建契约并规划执行分派", task.Title),
		Contracts:   contracts,
		Assignments: assignments,
		Transitions: transitions,
	}, nil
}

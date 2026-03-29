package runtime

import "fmt"

// MockPMAgent simulates a Project Manager agent.
// It decomposes a project into phases and tasks.
type MockPMAgent struct{}

func (a *MockPMAgent) Role() string { return "pm" }

func (a *MockPMAgent) Execute(input *AgentTaskInput) (*AgentTaskOutput, error) {
	if input.Project == nil {
		return &AgentTaskOutput{
			Status: OutputStatusFailed,
			Error:  "PM agent requires project context",
		}, nil
	}

	proj := input.Project

	phases := []PhaseAction{
		{
			Name:        "需求分析",
			Description: fmt.Sprintf("对「%s」进行需求梳理、目标定义与可行性分析", proj.Name),
			SortOrder:   1,
		},
		{
			Name:        "方案设计",
			Description: fmt.Sprintf("设计「%s」的技术方案、数据模型与接口规范", proj.Name),
			SortOrder:   2,
		},
		{
			Name:        "开发实现",
			Description: fmt.Sprintf("编码实现「%s」的核心功能模块", proj.Name),
			SortOrder:   3,
		},
		{
			Name:        "测试验收",
			Description: fmt.Sprintf("对「%s」进行测试、评审与验收", proj.Name),
			SortOrder:   4,
		},
	}

	tasks := []TaskAction{
		{PhaseName: "需求分析", Title: "需求文档编写", Description: "编写项目需求规格说明书（PRD）", Priority: 1},
		{PhaseName: "需求分析", Title: "可行性评估", Description: "评估技术可行性与资源需求", Priority: 2},
		{PhaseName: "方案设计", Title: "架构设计", Description: "设计系统整体架构与模块划分", Priority: 1},
		{PhaseName: "方案设计", Title: "数据库设计", Description: "设计数据库表结构与关系模型", Priority: 2},
		{PhaseName: "方案设计", Title: "API 接口设计", Description: "定义 RESTful API 接口规范", Priority: 3},
		{PhaseName: "开发实现", Title: "后端核心开发", Description: "实现核心业务逻辑与数据访问层", Priority: 1},
		{PhaseName: "开发实现", Title: "前端页面开发", Description: "实现前端用户界面与交互逻辑", Priority: 2},
		{PhaseName: "测试验收", Title: "集成测试", Description: "执行端到端集成测试", Priority: 1},
		{PhaseName: "测试验收", Title: "交付评审", Description: "组织项目交付评审会议", Priority: 2},
	}

	return &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: fmt.Sprintf("项目「%s」已分解为 %d 个阶段、%d 个任务", proj.Name, len(phases), len(tasks)),
		Phases:  phases,
		Tasks:   tasks,
	}, nil
}

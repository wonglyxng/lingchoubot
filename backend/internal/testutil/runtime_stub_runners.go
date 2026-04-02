package testutil

import (
	"fmt"
	"strings"

	"github.com/lingchou/lingchoubot/backend/internal/runtime"
)

func registerDeterministicTestRunners(reg *runtime.Registry) {
	reg.Register("pm", &pmTestRunner{})
	reg.Register("supervisor", &supervisorTestRunner{})
	reg.Register("worker", &workerTestRunner{spec: "general"})
	reg.Register("reviewer", &reviewerTestRunner{})
	reg.RegisterSpecialized("worker", "backend", &workerTestRunner{spec: "backend"})
	reg.RegisterSpecialized("worker", "frontend", &workerTestRunner{spec: "frontend"})
	reg.RegisterSpecialized("worker", "qa", &workerTestRunner{spec: "qa"})
}

type pmTestRunner struct{}

func (r *pmTestRunner) Role() string { return "pm" }

func (r *pmTestRunner) Execute(input *runtime.AgentTaskInput) (*runtime.AgentTaskOutput, error) {
	if input.Project == nil {
		return &runtime.AgentTaskOutput{Status: runtime.OutputStatusFailed, Error: "pm runner requires project context"}, nil
	}
	proj := input.Project
	return &runtime.AgentTaskOutput{
		Status:  runtime.OutputStatusSuccess,
		Summary: fmt.Sprintf("项目「%s」已分解为 4 个阶段、9 个任务", proj.Name),
		Phases: []runtime.PhaseAction{
			{Name: "需求分析", Description: fmt.Sprintf("对「%s」进行需求梳理、目标定义与可行性分析", proj.Name), SortOrder: 1},
			{Name: "方案设计", Description: fmt.Sprintf("设计「%s」的技术方案、数据模型与接口规范", proj.Name), SortOrder: 2},
			{Name: "开发实现", Description: fmt.Sprintf("编码实现「%s」的核心功能模块", proj.Name), SortOrder: 3},
			{Name: "测试验收", Description: fmt.Sprintf("对「%s」进行测试、评审与验收", proj.Name), SortOrder: 4},
		},
		Tasks: []runtime.TaskAction{
			{PhaseName: "需求分析", Title: "需求文档编写", Description: "编写项目需求规格说明书（PRD）", Priority: 1},
			{PhaseName: "需求分析", Title: "可行性评估", Description: "评估技术可行性与资源需求", Priority: 2},
			{PhaseName: "方案设计", Title: "架构设计", Description: "设计系统整体架构与模块划分", Priority: 1},
			{PhaseName: "方案设计", Title: "数据库设计", Description: "设计数据库表结构与关系模型", Priority: 2},
			{PhaseName: "方案设计", Title: "API 接口设计", Description: "定义 RESTful API 接口规范", Priority: 3},
			{PhaseName: "开发实现", Title: "后端核心开发", Description: "实现核心业务逻辑与数据访问层", Priority: 1},
			{PhaseName: "开发实现", Title: "前端页面开发", Description: "实现前端用户界面与交互逻辑", Priority: 2},
			{PhaseName: "测试验收", Title: "集成测试", Description: "执行端到端集成测试", Priority: 1},
			{PhaseName: "测试验收", Title: "交付评审", Description: "组织项目交付评审会议", Priority: 2},
		},
	}, nil
}

type supervisorTestRunner struct{}

func (r *supervisorTestRunner) Role() string { return "supervisor" }

func (r *supervisorTestRunner) Execute(input *runtime.AgentTaskInput) (*runtime.AgentTaskOutput, error) {
	if input.Task == nil {
		return &runtime.AgentTaskOutput{Status: runtime.OutputStatusFailed, Error: "supervisor runner requires task context"}, nil
	}
	task := input.Task
	return &runtime.AgentTaskOutput{
		Status:  runtime.OutputStatusSuccess,
		Summary: fmt.Sprintf("已为任务「%s」创建契约并规划执行分派", task.Title),
		Contracts: []runtime.ContractAction{{
			TaskTitle:          task.Title,
			Scope:              fmt.Sprintf("完成「%s」的全部工作内容", task.Title),
			NonGoals:           []string{"不涉及超出本任务范围的功能扩展", "不处理与本任务无关的技术债务"},
			DoneDefinition:     []string{fmt.Sprintf("「%s」的核心交付物已生成", task.Title), "交付物通过格式与内容基本检查", "交接快照已创建"},
			VerificationSteps:  []string{"检查交付物是否完整覆盖任务描述中的要求", "验证交付物格式是否符合项目规范", "确认交接快照中无遗漏风险项"},
			AcceptanceCriteria: []string{"交付物内容与任务描述一致", "独立评审结论为 approved"},
		}},
		Assignments: []runtime.AssignmentAction{{TaskTitle: task.Title, AgentRole: "worker", Role: "executor", Note: fmt.Sprintf("分派执行「%s」", task.Title)}},
		Transitions: []runtime.TransitionAction{{TaskTitle: task.Title, NewStatus: "assigned"}},
	}, nil
}

type workerTestRunner struct{ spec string }

func (r *workerTestRunner) Role() string { return "worker" }

func (r *workerTestRunner) Specialization() string { return r.spec }

func (r *workerTestRunner) Execute(input *runtime.AgentTaskInput) (*runtime.AgentTaskOutput, error) {
	if input.Task == nil {
		return &runtime.AgentTaskOutput{Status: runtime.OutputStatusFailed, Error: "worker runner requires task context"}, nil
	}
	task := input.Task
	name, artifactType, contentType, content := deterministicArtifactForTask(task.Title, task.Description, r.spec)
	return &runtime.AgentTaskOutput{
		Status:  runtime.OutputStatusNeedsReview,
		Summary: fmt.Sprintf("任务「%s」执行完成，已生成交付物", task.Title),
		Artifacts: []runtime.ArtifactAction{{
			Name:         name,
			ArtifactType: artifactType,
			Description:  fmt.Sprintf("任务「%s」的核心交付物", task.Title),
			URI:          fmt.Sprintf("artifact://test/%s/%s", task.ID, artifactType),
			ContentType:  contentType,
			SizeBytes:    int64(len(content)),
			Content:      content,
			Metadata:     map[string]any{"generator": "deterministic_test_runner", "spec": r.spec},
		}},
		Handoffs: []runtime.HandoffAction{{
			Summary:        fmt.Sprintf("任务「%s」已完成执行，交付物已生成", task.Title),
			CompletedItems: []string{fmt.Sprintf("完成「%s」核心内容编写", task.Title), "交付物已持久化"},
			PendingItems:   []string{"等待独立评审"},
			Risks:          []string{"测试桩内容仅用于验证编排行为"},
			NextSteps:      []string{"提交独立评审", "根据评审意见修订（如需要）"},
		}},
		Transitions: []runtime.TransitionAction{{TaskTitle: task.Title, NewStatus: "in_review"}},
	}, nil
}

type reviewerTestRunner struct{}

func (r *reviewerTestRunner) Role() string { return "reviewer" }

func (r *reviewerTestRunner) Execute(input *runtime.AgentTaskInput) (*runtime.AgentTaskOutput, error) {
	taskTitle := "任务"
	if input != nil && input.Task != nil && input.Task.Title != "" {
		taskTitle = input.Task.Title
	}
	artifactCount := 0
	if input != nil {
		artifactCount = len(input.Artifacts)
	}
	return &runtime.AgentTaskOutput{
		Status:  runtime.OutputStatusSuccess,
		Summary: fmt.Sprintf("任务「%s」评审完成，%d 个交付物已通过评审", taskTitle, artifactCount),
		Reviews: []runtime.ReviewAction{{
			Verdict:         "approved",
			Summary:         fmt.Sprintf("任务「%s」交付物内容完整，允许推进审批", taskTitle),
			Findings:        []string{"交付物与任务目标一致", "内容结构完整且可追踪"},
			Recommendations: []string{"进入审批阶段前补充一次最终人工复核"},
		}},
	}, nil
}

func deterministicArtifactForTask(title, description, spec string) (name, artifactType, contentType, content string) {
	switch spec {
	case "backend":
		return fmt.Sprintf("%s-后端实现", title), "source_code", "text/x-go", fmt.Sprintf("package handler\n\n// %s\n// %s\nfunc Execute() {}\n", title, description)
	case "frontend":
		return fmt.Sprintf("%s-前端实现", title), "source_code", "text/typescript", fmt.Sprintf("export function Render%s() { return '%s' }\n", sanitizeIdentifier(title), title)
	case "qa":
		return fmt.Sprintf("%s-测试报告", title), "test_report", "text/markdown", fmt.Sprintf("# %s\n\n- scope: %s\n- result: pass\n", title, description)
	default:
		if containsAnalysisKeyword(title) {
			return fmt.Sprintf("%s-分析结论", title), "design", "text/markdown", fmt.Sprintf("# %s\n\n## 背景目标\n%s\n\n## 技术可行性\n可行\n\n## 风险\n范围需继续澄清\n\n## 结论建议\n进入下一阶段。\n", title, description)
		}
		return fmt.Sprintf("%s-交付物", title), "other", "text/markdown", fmt.Sprintf("# %s\n\n%s\n", title, description)
	}
}

func containsAnalysisKeyword(title string) bool {
	for _, keyword := range []string{"需求", "可行性", "分析", "评估", "方案", "PRD"} {
		if strings.Contains(title, keyword) {
			return true
		}
	}
	return false
}

func sanitizeIdentifier(title string) string {
	var builder strings.Builder
	upper := true
	for _, r := range title {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			if upper && r >= 'a' && r <= 'z' {
				r -= 32
			}
			upper = false
			builder.WriteRune(r)
			continue
		}
		upper = true
	}
	if builder.Len() == 0 {
		return "Task"
	}
	return builder.String()
}
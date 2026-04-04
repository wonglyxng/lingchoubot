package runtime

import (
	"testing"
)

func TestValidateOutput_PM_Valid(t *testing.T) {
	output := &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: "项目分解完成",
		Phases:  []PhaseAction{{Name: "设计阶段", Description: "d", SortOrder: 1}},
		Tasks:   []TaskAction{{PhaseName: "设计阶段", Title: "API 设计", Description: "d", Priority: 3}},
	}
	if err := ValidateOutput("pm", "", output); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestValidateOutput_PM_MissingPhases(t *testing.T) {
	output := &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: "项目分解完成",
		Tasks:   []TaskAction{{PhaseName: "设计阶段", Title: "API 设计", Description: "d"}},
	}
	err := ValidateOutput("pm", "", output)
	if err == nil {
		t.Fatal("expected validation error for missing phases")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if ve.Role != "pm" {
		t.Errorf("expected role=pm, got %s", ve.Role)
	}
}

func TestValidateOutput_PM_MissingTasks(t *testing.T) {
	output := &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: "项目分解完成",
		Phases:  []PhaseAction{{Name: "设计阶段", Description: "d", SortOrder: 1}},
	}
	err := ValidateOutput("pm", "", output)
	if err == nil {
		t.Fatal("expected validation error for missing tasks")
	}
}

func TestValidateOutput_PM_EmptyPhaseName(t *testing.T) {
	output := &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: "ok",
		Phases:  []PhaseAction{{Name: "", Description: "d"}},
		Tasks:   []TaskAction{{PhaseName: "a", Title: "t"}},
	}
	err := ValidateOutput("pm", "", output)
	if err == nil {
		t.Fatal("expected validation error for empty phase name")
	}
}

func TestValidateOutputForInput_PM_RejectsProjectGoalDrift(t *testing.T) {
	input := &AgentTaskInput{
		Project: &ProjectCtx{
			Name:        "Web四则运算计算器（仅前端）",
			Description: "开发一个简易的 Web 四则运算计算器，支持加减乘除、清空与连续输入，不包含 CMS、后台管理或内容发布能力。",
		},
	}
	output := &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: "已拆解项目",
		Phases: []PhaseAction{
			{Name: "需求分析", Description: "梳理宠物领养平台的业务需求", SortOrder: 1},
		},
		Tasks: []TaskAction{
			{PhaseName: "需求分析", Title: "需求梳理与PRD文档编写", Description: "分析宠物领养平台的领养申请、内容管理和 CMS 配置需求", Priority: 5},
		},
	}

	if err := ValidateOutputForInput("pm", "", input, output); err == nil {
		t.Fatal("expected PM validation failure for project goal drift")
	}
}

func TestValidateOutputForInput_PM_AllowsBindingViaPhaseContext(t *testing.T) {
	input := &AgentTaskInput{
		Project: &ProjectCtx{
			Name:        "简易前端计算器调试项目",
			Description: "开发一个简易 Web 四则运算计算器，支持加减乘除、连续输入和结果显示。",
		},
	}
	output := &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: "将简易前端计算器调试项目分解为3个阶段和6个任务",
		Phases: []PhaseAction{
			{Name: "需求分析与技术规划", Description: "明确计算器功能需求，评估技术实现方案，制定开发计划", SortOrder: 1},
		},
		Tasks: []TaskAction{
			{PhaseName: "需求分析与技术规划", Title: "需求梳理与PRD编写", Description: "梳理用户需求、交互流程与验收标准，输出产品需求文档", Priority: 5},
		},
	}

	if err := ValidateOutputForInput("pm", "", input, output); err != nil {
		t.Fatalf("expected PM task to inherit project binding from phase context, got: %v", err)
	}
}

func TestValidateOutputForInput_PM_RejectsExplicitOutOfScopeTasks(t *testing.T) {
	input := &AgentTaskInput{
		Project: &ProjectCtx{
			Name:        "Web四则运算计算器（仅前端）",
			Description: "开发一个简易 Web 四则运算计算器，支持加减乘除、清空与连续输入，不包含后端 API、CMS、后台管理或内容发布能力。",
		},
	}
	output := &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: "将前端计算器项目拆解为需求、前端开发、后端 API 和 CMS 配置任务",
		Phases: []PhaseAction{
			{Name: "前端开发与联调", Description: "实现页面、并接入后端 API 与管理后台", SortOrder: 1},
		},
		Tasks: []TaskAction{
			{PhaseName: "前端开发与联调", Title: "计算器前端页面开发", Description: "实现计算器页面与交互逻辑", Priority: 5},
			{PhaseName: "前端开发与联调", Title: "计算器后端API开发", Description: "开发计算表达式接口并提供数据持久化", Priority: 4},
			{PhaseName: "前端开发与联调", Title: "CMS配置与后台管理页面", Description: "配置内容管理系统并实现管理后台", Priority: 3},
		},
	}

	err := ValidateOutputForInput("pm", "", input, output)
	if err == nil {
		t.Fatal("expected PM validation failure for explicit out-of-scope tasks")
	}
}

func TestValidateOutputForInput_PM_AllowsFrontendOnlyPlanWithinScope(t *testing.T) {
	input := &AgentTaskInput{
		Project: &ProjectCtx{
			Name:        "Web四则运算计算器（仅前端）",
			Description: "开发一个简易 Web 四则运算计算器，支持加减乘除、清空与连续输入，不包含后端 API、CMS、后台管理或内容发布能力。",
		},
	}
	output := &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: "将前端计算器项目拆解为需求分析、前端实现和测试验证任务",
		Phases: []PhaseAction{
			{Name: "前端开发与验证", Description: "实现页面、交互逻辑并完成前端验证", SortOrder: 1},
		},
		Tasks: []TaskAction{
			{PhaseName: "前端开发与验证", Title: "计算器前端页面开发", Description: "实现计算器按钮、显示区和响应式布局", Priority: 5},
			{PhaseName: "前端开发与验证", Title: "计算器交互逻辑实现", Description: "实现加减乘除、连续输入、清空重置和结果显示", Priority: 5},
			{PhaseName: "前端开发与验证", Title: "计算器前端功能测试", Description: "验证交互流程、边界输入和显示结果", Priority: 4},
		},
	}

	if err := ValidateOutputForInput("pm", "", input, output); err != nil {
		t.Fatalf("expected frontend-only plan to stay within scope, got: %v", err)
	}
}

func TestValidateOutputForInput_PM_AllowsRestatingExplicitNonGoals(t *testing.T) {
	input := &AgentTaskInput{
		Project: &ProjectCtx{
			Name:        "Web四则运算计算器（仅前端）",
			Description: "开发一个简易 Web 四则运算计算器，支持加减乘除、清空与连续输入，不包含后端 API、CMS、后台管理或内容发布能力。",
		},
	}
	output := &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: "本项目仅包含前端计算器功能，不涉及后端 API、CMS 或后台管理。",
		Phases: []PhaseAction{
			{Name: "前端开发与验证", Description: "围绕前端页面和交互逻辑实现，不接入后端 API", SortOrder: 1},
		},
		Tasks: []TaskAction{
			{PhaseName: "前端开发与验证", Title: "计算器前端页面开发", Description: "实现计算器按钮、显示区和响应式布局", Priority: 5},
			{PhaseName: "前端开发与验证", Title: "计算器交互逻辑实现", Description: "实现加减乘除、连续输入、清空重置和结果显示", Priority: 5},
		},
	}

	if err := ValidateOutputForInput("pm", "", input, output); err != nil {
		t.Fatalf("expected explicit non-goal restatement to pass validation, got: %v", err)
	}
}

func TestValidateOutput_Supervisor_Valid(t *testing.T) {
	output := &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: "契约已创建",
		Contracts: []ContractAction{{
			TaskTitle:          "task1",
			Scope:              "scope",
			DoneDefinition:     []string{"done1", "done2"},
			AcceptanceCriteria: []string{"ac1"},
		}},
	}
	if err := ValidateOutput("supervisor", "", output); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestValidateOutput_Supervisor_ReviewPolicyRequiresReasonAndSource(t *testing.T) {
	output := &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: "契约已创建",
		Contracts: []ContractAction{{
			TaskTitle:          "task1",
			Scope:              "scope",
			DoneDefinition:     []string{"done1", "done2"},
			AcceptanceCriteria: []string{"ac1"},
			ReviewPolicy: map[string]any{
				"pass_threshold": 85,
			},
		}},
	}
	err := ValidateOutput("supervisor", "", output)
	if err == nil {
		t.Fatal("expected validation error for review policy without reason/source")
	}
}

func TestValidateOutput_Supervisor_MissingContracts(t *testing.T) {
	output := &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: "ok",
	}
	err := ValidateOutput("supervisor", "", output)
	if err == nil {
		t.Fatal("expected validation error for missing contracts")
	}
}

func TestValidateOutput_Supervisor_InsufficientDoneDefinition(t *testing.T) {
	output := &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: "ok",
		Contracts: []ContractAction{{
			TaskTitle:          "task1",
			Scope:              "scope",
			DoneDefinition:     []string{"only_one"},
			AcceptanceCriteria: []string{"ac1"},
		}},
	}
	err := ValidateOutput("supervisor", "", output)
	if err == nil {
		t.Fatal("expected validation error for insufficient done_definition")
	}
}

func TestValidateOutput_Worker_Valid(t *testing.T) {
	output := &AgentTaskOutput{
		Status:  OutputStatusNeedsReview,
		Summary: "工件产出完成",
		Artifacts: []ArtifactAction{{
			Name:         "handler.go",
			ArtifactType: "source_code",
			ContentType:  "text/x-go",
			Content:      "package main\nfunc main() {}",
		}},
	}
	if err := ValidateOutput("worker", "backend", output); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestValidateOutput_Worker_MissingArtifacts(t *testing.T) {
	output := &AgentTaskOutput{
		Status:  OutputStatusNeedsReview,
		Summary: "ok",
	}
	err := ValidateOutput("worker", "backend", output)
	if err == nil {
		t.Fatal("expected validation error for missing artifacts")
	}
}

func TestValidateOutput_Reviewer_Valid(t *testing.T) {
	output := &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: "评审完成",
		Reviews: []ReviewAction{{
			Verdict:         "approved",
			TemplateKey:     "backend_v1",
			PassThreshold:   80,
			TotalScore:      85,
			HardGateResults: []HardGateResultAction{{Key: "goal_match", Passed: true, Reason: "工件与任务目标一致"}},
			ScoreItems:      []ScoreItemResultAction{{Key: "functional_correctness", Name: "功能正确性", Weight: 35, Score: 35, MaxScore: 35, Reason: "功能实现完整"}},
			Findings:        []string{"good structure", "content is relevant"},
			Recommendations: []string{"keep adding focused evidence"},
		}},
	}
	if err := ValidateOutput("reviewer", "", output); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestValidateOutput_Reviewer_InvalidVerdict(t *testing.T) {
	output := &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: "评审完成",
		Reviews: []ReviewAction{{
			Verdict:         "rejected",
			Findings:        []string{"bad", "still bad"},
			Recommendations: []string{"fix it"},
		}},
	}
	err := ValidateOutput("reviewer", "", output)
	if err == nil {
		t.Fatal("expected validation error for invalid verdict")
	}
}

func TestValidateOutput_Reviewer_MissingReviews(t *testing.T) {
	output := &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: "ok",
	}
	err := ValidateOutput("reviewer", "", output)
	if err == nil {
		t.Fatal("expected validation error for missing reviews")
	}
}

func TestValidateOutput_Reviewer_MultipleReviews(t *testing.T) {
	output := &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: "评审完成",
		Reviews: []ReviewAction{
			{
				Verdict:         "approved",
				Findings:        []string{"good structure", "content is relevant"},
				Recommendations: []string{"keep adding focused evidence"},
			},
			{
				Verdict:         "needs_revision",
				Findings:        []string{"scope drift", "missing acceptance criteria"},
				Recommendations: []string{"merge into one task-level conclusion"},
			},
		},
	}
	err := ValidateOutput("reviewer", "", output)
	if err == nil {
		t.Fatal("expected validation error for multiple reviews")
	}
}

func TestValidateOutput_NilOutput(t *testing.T) {
	err := ValidateOutput("pm", "", nil)
	if err == nil {
		t.Fatal("expected validation error for nil output")
	}
}

func TestValidateOutput_EmptyStatus(t *testing.T) {
	output := &AgentTaskOutput{
		Summary: "ok",
		Phases:  []PhaseAction{{Name: "a"}},
		Tasks:   []TaskAction{{PhaseName: "a", Title: "t"}},
	}
	err := ValidateOutput("pm", "", output)
	if err == nil {
		t.Fatal("expected validation error for empty status")
	}
}

func TestValidationError_Error(t *testing.T) {
	ve := &ValidationError{
		Role:     "pm",
		Failures: []string{"missing phases", "missing tasks"},
	}
	msg := ve.Error()
	if msg == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestValidateOutputForInput_AnalysisTaskRejectsTestReport(t *testing.T) {
	input := &AgentTaskInput{
		Project: &ProjectCtx{Name: "计算器"},
		Task:    &TaskCtx{Title: "可行性评估", Description: "评估简易计算器的技术可行性与资源需求"},
	}
	output := &AgentTaskOutput{
		Status:  OutputStatusNeedsReview,
		Summary: "done",
		Artifacts: []ArtifactAction{{
			Name:         "report.md",
			ArtifactType: "test_report",
			ContentType:  "text/markdown",
			Content:      "# 测试报告\n与可行性分析无关",
		}},
	}
	if err := ValidateOutputForInput("worker", "general", input, output); err == nil {
		t.Fatal("expected analysis task validation failure")
	}
}

func TestValidateOutputForInput_AnalysisTaskWithValidationLanguageAllowsPRD(t *testing.T) {
	input := &AgentTaskInput{
		Project: &ProjectCtx{Name: "Artifact修复验证项目"},
		Task: &TaskCtx{
			Title:       "需求梳理与PRD编写",
			Description: "详细分析artifact写入修复和SSE修复的具体验证需求，明确验证场景、成功标准和验收条件，编写项目需求文档",
		},
	}
	output := &AgentTaskOutput{
		Status:  OutputStatusNeedsReview,
		Summary: "已输出需求文档",
		Artifacts: []ArtifactAction{{
			Name:         "artifact修复验证-prd.md",
			ArtifactType: "prd",
			ContentType:  "text/markdown",
			Content:      "# Artifact修复验证项目需求文档\n\n任务：需求梳理与PRD编写\n\n目标：明确验证场景、成功标准和验收条件。",
		}},
	}
	if err := ValidateOutputForInput("worker", "qa", input, output); err != nil {
		t.Fatalf("expected analysis-style PRD task to pass validation, got: %v", err)
	}
}

func TestValidateOutputForInput_Worker_AllowsKeywordLevelProjectBinding(t *testing.T) {
	input := &AgentTaskInput{
		Project: &ProjectCtx{
			Name:        "前端四则运算计算器验证项目",
			Description: "实现简易四则运算计算器页面，支持加减乘除、连续输入和清空重置。",
		},
		Task: &TaskCtx{
			Title:       "需求梳理与PRD文档编写",
			Description: "分析前端四则运算计算器的交互需求和页面行为，输出 PRD 文档。",
		},
	}
	output := &AgentTaskOutput{
		Status:  OutputStatusNeedsReview,
		Summary: "已输出需求文档",
		Artifacts: []ArtifactAction{{
			Name:         "calculator-prd.md",
			ArtifactType: "prd",
			ContentType:  "text/markdown",
			Content:      "# 四则运算计算器需求文档\n\n目标：明确加减乘除、连续输入、清空重置和结果显示的页面交互规则。",
		}},
	}

	if err := ValidateOutputForInput("worker", "general", input, output); err != nil {
		t.Fatalf("expected keyword-level binding to pass validation, got: %v", err)
	}
}

func TestValidateOutputForInput_Worker_RejectsUnrelatedProjectContent(t *testing.T) {
	input := &AgentTaskInput{
		Project: &ProjectCtx{
			Name:        "前端四则运算计算器验证项目",
			Description: "实现简易四则运算计算器页面，支持加减乘除、连续输入和清空重置。",
		},
		Task: &TaskCtx{
			Title:       "需求梳理与PRD文档编写",
			Description: "分析前端四则运算计算器的交互需求和页面行为，输出 PRD 文档。",
		},
	}
	output := &AgentTaskOutput{
		Status:  OutputStatusNeedsReview,
		Summary: "已输出需求文档",
		Artifacts: []ArtifactAction{{
			Name:         "pet-platform-prd.md",
			ArtifactType: "prd",
			ContentType:  "text/markdown",
			Content:      "# 宠物领养平台需求文档\n\n目标：明确领养申请、内容管理和 CMS 配置流程。",
		}},
	}

	if err := ValidateOutputForInput("worker", "general", input, output); err == nil {
		t.Fatal("expected unrelated project content to fail validation")
	}
}

func TestValidateOutputForInput_VerificationTaskStillRequiresTestReport(t *testing.T) {
	input := &AgentTaskInput{
		Task: &TaskCtx{Title: "SSE修复验证", Description: "验证事件推送稳定性与错误处理机制"},
	}
	output := &AgentTaskOutput{
		Status:  OutputStatusNeedsReview,
		Summary: "done",
		Artifacts: []ArtifactAction{{
			Name:         "verification-notes.md",
			ArtifactType: "other",
			ContentType:  "text/markdown",
			Content:      "仅记录观察，没有测试报告。",
		}},
	}
	if err := ValidateOutputForInput("worker", "qa", input, output); err == nil {
		t.Fatal("expected verification task to require test_report artifact")
	}
}

func TestValidateOutputForInput_ReviewerRejectsPlaceholderApproval(t *testing.T) {
	input := &AgentTaskInput{
		Task: &TaskCtx{Title: "可行性评估", Description: "评估项目可行性"},
		Artifacts: []ArtifactCtx{{
			Name:         "可行性评估-测试报告",
			ArtifactType: "test_report",
			Content:      "由 Mock QA Worker Agent 自动生成。",
		}},
	}
	output := &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: "评审完成",
		Reviews: []ReviewAction{{
			Verdict:         "approved",
			TemplateKey:     "architecture_v1",
			PassThreshold:   80,
			TotalScore:      82,
			HardGateResults: []HardGateResultAction{{Key: "goal_match", Passed: true, Reason: "目标一致"}},
			ScoreItems:      []ScoreItemResultAction{{Key: "technical_feasibility", Name: "技术可行性", Weight: 25, Score: 20, MaxScore: 25, Reason: "基本可行"}},
			Findings:        []string{"内容完整", "结构清晰"},
			Recommendations: []string{"继续推进"},
		}},
	}
	if err := ValidateOutputForInput("reviewer", "", input, output); err == nil {
		t.Fatal("expected reviewer validation failure")
	}
}

func TestValidateOutputForInput_ReviewerRequiresFullScorecardCoverage(t *testing.T) {
	input := &AgentTaskInput{
		Task: &TaskCtx{Title: "需求梳理与PRD编写", Description: "输出完整 PRD"},
		Contract: &ContractCtx{
			Scope:              "输出 PRD",
			AcceptanceCriteria: []string{"验收标准明确"},
			ReviewPolicy: &ReviewPolicyCtx{
				TemplateKey:   "prd_v1",
				TaskCategory:  "prd",
				PassThreshold: 80,
				HardGates: []HardGateCtx{
					{Key: "goal_match", Name: "工件与任务目标一致"},
					{Key: "acceptance_testable", Name: "验收标准可验证"},
				},
				ScoreItems: []ScoreItemCtx{
					{Key: "completeness", Name: "完整性", Weight: 25},
					{Key: "executability", Name: "可执行性", Weight: 20},
				},
			},
		},
	}
	output := &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: "评审完成",
		Reviews: []ReviewAction{{
			Verdict:       "needs_revision",
			TemplateKey:   "prd_v1",
			PassThreshold: 80,
			TotalScore:    74,
			HardGateResults: []HardGateResultAction{
				{Key: "goal_match", Passed: true, Reason: "内容与任务一致"},
			},
			ScoreItems: []ScoreItemResultAction{
				{Key: "completeness", Name: "完整性", Weight: 25, Score: 20, MaxScore: 25, Reason: "主体基本完整"},
			},
			Findings:        []string{"验收标准不够细", "缺少边界场景"},
			Recommendations: []string{"补充更多细节"},
			MustFixItems:    []string{"补充可验证的验收标准"},
		}},
	}
	if err := ValidateOutputForInput("reviewer", "", input, output); err == nil {
		t.Fatal("expected scorecard coverage validation failure")
	}
}

func TestValidateOutputForInput_ReviewerNeedsRevisionRequiresMustFixItems(t *testing.T) {
	input := &AgentTaskInput{
		Task: &TaskCtx{Title: "后端接口实现", Description: "完成 API"},
		Contract: &ContractCtx{
			ReviewPolicy: &ReviewPolicyCtx{
				TemplateKey:   "backend_v1",
				TaskCategory:  "backend",
				PassThreshold: 80,
				HardGates:     []HardGateCtx{{Key: "goal_match", Name: "工件与任务目标一致"}},
				ScoreItems:    []ScoreItemCtx{{Key: "functional_correctness", Name: "功能正确性", Weight: 100}},
			},
		},
	}
	output := &AgentTaskOutput{
		Status:  OutputStatusSuccess,
		Summary: "评审完成",
		Reviews: []ReviewAction{{
			Verdict:       "needs_revision",
			TemplateKey:   "backend_v1",
			PassThreshold: 80,
			TotalScore:    70,
			HardGateResults: []HardGateResultAction{
				{Key: "goal_match", Passed: true, Reason: "整体一致"},
			},
			ScoreItems: []ScoreItemResultAction{
				{Key: "functional_correctness", Name: "功能正确性", Weight: 100, Score: 70, MaxScore: 100, Reason: "边界条件不足"},
			},
			Findings:        []string{"边界处理不足", "错误返回不完整"},
			Recommendations: []string{"补充边界条件"},
		}},
	}
	if err := ValidateOutputForInput("reviewer", "", input, output); err == nil {
		t.Fatal("expected must_fix_items validation failure")
	}
}

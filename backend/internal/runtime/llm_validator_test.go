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
			Findings:        []string{"内容完整", "结构清晰"},
			Recommendations: []string{"继续推进"},
		}},
	}
	if err := ValidateOutputForInput("reviewer", "", input, output); err == nil {
		t.Fatal("expected reviewer validation failure")
	}
}

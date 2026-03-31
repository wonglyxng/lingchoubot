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
			ArtifactType: "code",
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
			Verdict:  "approved",
			Findings: []string{"good structure"},
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
			Verdict:  "rejected",
			Findings: []string{"bad"},
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

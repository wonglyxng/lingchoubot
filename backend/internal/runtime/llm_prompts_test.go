package runtime

import (
	"testing"
)

func TestGetPromptVersion_KnownRoles(t *testing.T) {
	roles := []string{"pm", "supervisor", "worker", "reviewer"}
	for _, role := range roles {
		pv := GetPromptVersion(role, "")
		if pv.Role != role {
			t.Errorf("expected role=%s, got %s", role, pv.Role)
		}
		if pv.Version == "" || pv.Version == "0.0.0" {
			t.Errorf("expected non-zero version for role=%s, got %s", role, pv.Version)
		}
		if pv.ChangeLog == "" {
			t.Errorf("expected non-empty changelog for role=%s", role)
		}
	}
}

func TestGetPromptVersion_UnknownRole(t *testing.T) {
	pv := GetPromptVersion("unknown", "")
	if pv.Version != "0.0.0" {
		t.Errorf("expected 0.0.0 for unknown role, got %s", pv.Version)
	}
}

func TestBuildSystemPrompt_AllRolesPresent(t *testing.T) {
	cases := []struct {
		role string
		spec string
	}{
		{"pm", ""},
		{"supervisor", ""},
		{"worker", ""},
		{"worker", "backend"},
		{"worker", "frontend"},
		{"worker", "qa"},
		{"reviewer", ""},
		{"unknown", ""},
	}
	for _, c := range cases {
		p := buildSystemPrompt(c.role, c.spec)
		if p == "" {
			t.Errorf("expected non-empty prompt for role=%s spec=%s", c.role, c.spec)
		}
	}
}

func TestBuildUserPrompt_NonEmpty(t *testing.T) {
	input := &AgentTaskInput{
		RunID:       "r1",
		AgentID:     "a1",
		AgentRole:   "pm",
		Instruction: "test",
	}
	prompt, err := buildUserPrompt(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prompt == "" {
		t.Fatal("expected non-empty user prompt")
	}
}

func TestEvalSamples_Count(t *testing.T) {
	samples := EvalSamples()
	if len(samples) < 4 {
		t.Errorf("expected at least 4 eval samples, got %d", len(samples))
	}
	roles := make(map[string]bool)
	for _, s := range samples {
		roles[s.Role] = true
		if s.Name == "" {
			t.Error("sample has empty name")
		}
		if s.Input == nil {
			t.Errorf("sample %s has nil input", s.Name)
		}
	}
	for _, r := range []string{"pm", "supervisor", "worker", "reviewer"} {
		if !roles[r] {
			t.Errorf("missing eval sample for role=%s", r)
		}
	}
}

func TestEvalSamples_Validation(t *testing.T) {
	// Verify that mock runners can produce valid output for each eval sample
	samples := EvalSamples()
	mockRunners := map[string]AgentRunner{
		"pm":       &MockPMAgent{},
		"supervisor": &MockSupervisorAgent{},
		"worker":   &MockBackendWorkerAgent{},
		"reviewer": &MockReviewerAgent{},
	}
	for _, s := range samples {
		runner, ok := mockRunners[s.Role]
		if !ok {
			continue
		}
		output, err := runner.Execute(s.Input)
		if err != nil {
			t.Errorf("sample %s: mock runner error: %v", s.Name, err)
			continue
		}
		if valErr := ValidateOutput(s.Role, s.Spec, output); valErr != nil {
			t.Errorf("sample %s: mock output fails validation: %v", s.Name, valErr)
		}
	}
}

package runtime

import "testing"

func TestRegistryGetForSpec(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterDefaults()

	tests := []struct {
		name    string
		role    string
		spec    string
		wantErr bool
	}{
		{"base pm", "pm", "", false},
		{"base worker", "worker", "", false},
		{"base worker general", "worker", "general", false},
		{"specialized backend", "worker", "backend", false},
		{"specialized frontend", "worker", "frontend", false},
		{"specialized qa", "worker", "qa", false},
		{"unregistered spec falls back", "worker", "release", false},
		{"unknown role", "unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner, err := reg.GetForSpec(tt.role, tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if runner == nil {
				t.Errorf("expected runner, got nil")
			}
		})
	}
}

func TestRegistrySpecializedRunnerOutput(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterDefaults()

	input := &AgentTaskInput{
		RunID:     "test-run",
		AgentID:   "test-agent",
		AgentRole: "worker",
		Task: &TaskCtx{
			ID:          "test-task",
			Title:       "后端 API 开发",
			Description: "实现用户管理接口",
		},
	}

	// Test backend worker produces source_code artifact
	runner, _ := reg.GetForSpec("worker", "backend")
	output, err := runner.Execute(input)
	if err != nil {
		t.Fatalf("backend worker execute: %v", err)
	}
	if len(output.Artifacts) == 0 {
		t.Fatal("expected artifacts from backend worker")
	}
	if output.Artifacts[0].ArtifactType != "source_code" {
		t.Errorf("expected source_code, got %s", output.Artifacts[0].ArtifactType)
	}

	// Test frontend worker produces source_code artifact with tsx content
	runner, _ = reg.GetForSpec("worker", "frontend")
	output, err = runner.Execute(input)
	if err != nil {
		t.Fatalf("frontend worker execute: %v", err)
	}
	if len(output.Artifacts) == 0 {
		t.Fatal("expected artifacts from frontend worker")
	}
	if output.Artifacts[0].ContentType != "text/typescript" {
		t.Errorf("expected text/typescript, got %s", output.Artifacts[0].ContentType)
	}

	// Test QA worker produces test_report artifact
	runner, _ = reg.GetForSpec("worker", "qa")
	output, err = runner.Execute(input)
	if err != nil {
		t.Fatalf("qa worker execute: %v", err)
	}
	if len(output.Artifacts) == 0 {
		t.Fatal("expected artifacts from qa worker")
	}
	if output.Artifacts[0].ArtifactType != "test_report" {
		t.Errorf("expected test_report, got %s", output.Artifacts[0].ArtifactType)
	}
}

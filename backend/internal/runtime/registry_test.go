package runtime

import "testing"

type registryTestRunner struct {
	role   string
	spec   string
	output *AgentTaskOutput
}

func (r *registryTestRunner) Role() string { return r.role }

func (r *registryTestRunner) Execute(input *AgentTaskInput) (*AgentTaskOutput, error) {
	if r.output != nil {
		return r.output, nil
	}
	return &AgentTaskOutput{Status: OutputStatusSuccess, Summary: "ok"}, nil
}

func TestRegistryGetForSpec(t *testing.T) {
	reg := NewRegistry()
	reg.Register("pm", &registryTestRunner{role: "pm"})
	reg.Register("worker", &registryTestRunner{role: "worker"})
	reg.RegisterSpecialized("worker", "backend", &registryTestRunner{role: "worker", spec: "backend"})
	reg.RegisterSpecialized("worker", "frontend", &registryTestRunner{role: "worker", spec: "frontend"})
	reg.RegisterSpecialized("worker", "qa", &registryTestRunner{role: "worker", spec: "qa"})

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
		{"unregistered spec fails", "worker", "release", true},
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
	reg.RegisterSpecialized("worker", "backend", &registryTestRunner{role: "worker", spec: "backend", output: &AgentTaskOutput{
		Status: OutputStatusNeedsReview,
		Artifacts: []ArtifactAction{{ArtifactType: "source_code", ContentType: "text/x-go", Content: "package main"}},
	}})
	reg.RegisterSpecialized("worker", "frontend", &registryTestRunner{role: "worker", spec: "frontend", output: &AgentTaskOutput{
		Status: OutputStatusNeedsReview,
		Artifacts: []ArtifactAction{{ArtifactType: "source_code", ContentType: "text/typescript", Content: "export default function Demo() {}"}},
	}})
	reg.RegisterSpecialized("worker", "qa", &registryTestRunner{role: "worker", spec: "qa", output: &AgentTaskOutput{
		Status: OutputStatusNeedsReview,
		Artifacts: []ArtifactAction{{ArtifactType: "test_report", ContentType: "text/markdown", Content: "# QA"}},
	}})

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

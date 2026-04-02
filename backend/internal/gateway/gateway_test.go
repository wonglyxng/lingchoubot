package gateway

import (
	"context"
	"testing"
)

func TestMatchCapabilities(t *testing.T) {
	tests := []struct {
		name         string
		capabilities []string
		required     []string
		wantErr      bool
	}{
		{
			name:         "no required permissions",
			capabilities: nil,
			required:     nil,
			wantErr:      false,
		},
		{
			name:         "wildcard grants all",
			capabilities: []string{"tool.*"},
			required:     []string{"tool.doc_generator", "tool.test_runner"},
			wantErr:      false,
		},
		{
			name:         "exact match single",
			capabilities: []string{"tool.doc_generator"},
			required:     []string{"tool.doc_generator"},
			wantErr:      false,
		},
		{
			name:         "exact match multiple",
			capabilities: []string{"tool.doc_generator", "tool.test_runner", "tool.artifact_storage"},
			required:     []string{"tool.doc_generator", "tool.test_runner"},
			wantErr:      false,
		},
		{
			name:         "missing one permission",
			capabilities: []string{"tool.doc_generator"},
			required:     []string{"tool.doc_generator", "tool.test_runner"},
			wantErr:      true,
		},
		{
			name:         "no capabilities at all",
			capabilities: nil,
			required:     []string{"tool.doc_generator"},
			wantErr:      true,
		},
		{
			name:         "empty capabilities",
			capabilities: []string{},
			required:     []string{"tool.doc_generator"},
			wantErr:      true,
		},
		{
			name:         "unrelated capabilities",
			capabilities: []string{"tool.artifact_storage"},
			required:     []string{"tool.doc_generator"},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := matchCapabilities(tt.capabilities, tt.required)
			if (err != nil) != tt.wantErr {
				t.Errorf("matchCapabilities() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestRegisterDefaultsRegistersOnlyArtifactStorage(t *testing.T) {
	g := NewGateway(nil, nil, nil, nil, nil)
	g.RegisterDefaults(nil)

	tools := g.ListTools()
	if len(tools) != 1 {
		t.Fatalf("registered tools = %d, want 1", len(tools))
	}
	if tools[0].Name != "artifact_storage" {
		t.Fatalf("registered tool = %s, want artifact_storage", tools[0].Name)
	}
}

func TestDeprecatedMockToolsFailClosed(t *testing.T) {
	docResult, err := (&DocGeneratorTool{}).Execute(context.Background(), map[string]any{"title": "demo"})
	if err != nil {
		t.Fatalf("doc generator returned unexpected error: %v", err)
	}
	if docResult.Status != "failed" {
		t.Fatalf("doc generator status = %s, want failed", docResult.Status)
	}

	testResult, err := (&TestRunnerTool{}).Execute(context.Background(), map[string]any{"suite_name": "demo"})
	if err != nil {
		t.Fatalf("test runner returned unexpected error: %v", err)
	}
	if testResult.Status != "failed" {
		t.Fatalf("test runner status = %s, want failed", testResult.Status)
	}
}

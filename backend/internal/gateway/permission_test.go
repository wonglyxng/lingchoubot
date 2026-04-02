package gateway

import (
	"testing"
)

func TestMatchCapabilities_Extended(t *testing.T) {
	tests := []struct {
		name         string
		capabilities []string
		required     []string
		wantErr      bool
	}{
		{
			name:         "no requirements",
			capabilities: nil,
			required:     nil,
			wantErr:      false,
		},
		{
			name:         "wildcard grants all",
			capabilities: []string{"tool.*"},
			required:     []string{"tool.doc_generator", "tool.artifact_storage"},
			wantErr:      false,
		},
		{
			name:         "exact match",
			capabilities: []string{"tool.doc_generator"},
			required:     []string{"tool.doc_generator"},
			wantErr:      false,
		},
		{
			name:         "multiple capabilities satisfy multiple requirements",
			capabilities: []string{"tool.doc_generator", "tool.test_runner"},
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
			name:         "empty capabilities with requirements",
			capabilities: nil,
			required:     []string{"tool.doc_generator"},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := matchCapabilities(tt.capabilities, tt.required)
			if (err != nil) != tt.wantErr {
				t.Errorf("matchCapabilities() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMatchActionPermission(t *testing.T) {
	tests := []struct {
		name         string
		capabilities []string
		toolName     string
		actionPerm   string
		wantErr      bool
	}{
		{
			name:         "global wildcard",
			capabilities: []string{"tool.*"},
			toolName:     "artifact_storage",
			actionPerm:   "tool.artifact_storage:write",
			wantErr:      false,
		},
		{
			name:         "tool-level grants all actions",
			capabilities: []string{"tool.artifact_storage"},
			toolName:     "artifact_storage",
			actionPerm:   "tool.artifact_storage:write",
			wantErr:      false,
		},
		{
			name:         "tool wildcard action",
			capabilities: []string{"tool.artifact_storage:*"},
			toolName:     "artifact_storage",
			actionPerm:   "tool.artifact_storage:delete",
			wantErr:      false,
		},
		{
			name:         "exact action match",
			capabilities: []string{"tool.artifact_storage:write"},
			toolName:     "artifact_storage",
			actionPerm:   "tool.artifact_storage:write",
			wantErr:      false,
		},
		{
			name:         "wrong action",
			capabilities: []string{"tool.artifact_storage:read"},
			toolName:     "artifact_storage",
			actionPerm:   "tool.artifact_storage:write",
			wantErr:      true,
		},
		{
			name:         "wrong tool",
			capabilities: []string{"tool.doc_generator:write"},
			toolName:     "artifact_storage",
			actionPerm:   "tool.artifact_storage:write",
			wantErr:      true,
		},
		{
			name:         "no capabilities",
			capabilities: nil,
			toolName:     "artifact_storage",
			actionPerm:   "tool.artifact_storage:write",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := matchActionPermission(tt.capabilities, tt.toolName, tt.actionPerm)
			if (err != nil) != tt.wantErr {
				t.Errorf("matchActionPermission() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRiskLevelConstants(t *testing.T) {
	if RiskNormal != "normal" {
		t.Errorf("RiskNormal = %q, want 'normal'", RiskNormal)
	}
	if RiskSensitive != "sensitive" {
		t.Errorf("RiskSensitive = %q, want 'sensitive'", RiskSensitive)
	}
	if RiskCritical != "critical" {
		t.Errorf("RiskCritical = %q, want 'critical'", RiskCritical)
	}
}

func TestArtifactStorageTool_Actions(t *testing.T) {
	tool := NewUnavailableArtifactStorageTool()

	actions := tool.Actions()
	expected := map[string]bool{"read": true, "write": true, "delete": true}
	if len(actions) != len(expected) {
		t.Fatalf("Actions() returned %d items, want %d", len(actions), len(expected))
	}
	for _, a := range actions {
		if !expected[a] {
			t.Errorf("unexpected action %q", a)
		}
	}
}

func TestArtifactStorageTool_RiskLevel(t *testing.T) {
	tool := NewUnavailableArtifactStorageTool()

	tests := []struct {
		action string
		want   RiskLevel
	}{
		{"read", RiskNormal},
		{"write", RiskNormal},
		{"delete", RiskCritical},
		{"unknown", RiskNormal},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			got := tool.RiskLevel(tt.action)
			if got != tt.want {
				t.Errorf("RiskLevel(%q) = %q, want %q", tt.action, got, tt.want)
			}
		})
	}
}

// TestActionAwareToolInterface verifies that ArtifactStorageTool implements ActionAwareTool.
func TestActionAwareToolInterface(t *testing.T) {
	var _ ActionAwareTool = (*ArtifactStorageTool)(nil)
}

// TestToolInterfaceBackwardCompat verifies that non-ActionAware tools still implement Tool.
func TestToolInterfaceBackwardCompat(t *testing.T) {
	var _ Tool = (*DocGeneratorTool)(nil)
	var _ Tool = (*TestRunnerTool)(nil)
}

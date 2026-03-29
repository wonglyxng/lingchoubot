package model

import "testing"

func TestWorkflowRunStatusConstants(t *testing.T) {
	tests := []struct {
		status WorkflowRunStatus
		want   string
	}{
		{WorkflowRunPending, "pending"},
		{WorkflowRunRunning, "running"},
		{WorkflowRunCompleted, "completed"},
		{WorkflowRunFailed, "failed"},
	}
	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("WorkflowRunStatus = %q, want %q", tt.status, tt.want)
		}
	}
}

func TestWorkflowStepStatusConstants(t *testing.T) {
	tests := []struct {
		status WorkflowStepStatus
		want   string
	}{
		{WorkflowStepPending, "pending"},
		{WorkflowStepRunning, "running"},
		{WorkflowStepCompleted, "completed"},
		{WorkflowStepFailed, "failed"},
		{WorkflowStepSkipped, "skipped"},
	}
	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("WorkflowStepStatus = %q, want %q", tt.status, tt.want)
		}
	}
}

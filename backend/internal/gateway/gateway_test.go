package gateway

import "testing"

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

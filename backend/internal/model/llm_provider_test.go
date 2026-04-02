package model

import "testing"

func TestLLMProviderMaskedAPIKey(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
		want   string
	}{
		{"empty", "", ""},
		{"short", "abc", "****"},
		{"exact4", "ab12", "****"},
		{"normal", "sk-1234567890abcdef", "****cdef"},
		{"long", "sk-proj-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx1234", "****1234"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &LLMProvider{APIKey: tc.apiKey}
			got := p.MaskedAPIKey()
			if got != tc.want {
				t.Errorf("MaskedAPIKey(%q) = %q, want %q", tc.apiKey, got, tc.want)
			}
		})
	}
}

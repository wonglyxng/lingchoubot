package model

import "testing"

func TestAgentTypeConstants(t *testing.T) {
	tests := []struct {
		at   AgentType
		want string
	}{
		{AgentTypeMock, "mock"},
		{AgentTypeLLM, "llm"},
		{AgentTypeHuman, "human"},
	}
	for _, tt := range tests {
		if string(tt.at) != tt.want {
			t.Errorf("AgentType = %q, want %q", tt.at, tt.want)
		}
	}
}

func TestAgentSpecializationConstants(t *testing.T) {
	tests := []struct {
		spec AgentSpecialization
		want string
	}{
		{AgentSpecGeneral, "general"},
		{AgentSpecBackend, "backend"},
		{AgentSpecFrontend, "frontend"},
		{AgentSpecQA, "qa"},
		{AgentSpecRelease, "release"},
		{AgentSpecDevOps, "devops"},
		{AgentSpecDesign, "design"},
	}
	for _, tt := range tests {
		if string(tt.spec) != tt.want {
			t.Errorf("AgentSpecialization = %q, want %q", tt.spec, tt.want)
		}
	}
}

func TestMatchesSpecialization(t *testing.T) {
	tests := []struct {
		name     string
		agent    Agent
		spec     AgentSpecialization
		expected bool
	}{
		{
			name:     "general matches any",
			agent:    Agent{Specialization: AgentSpecGeneral},
			spec:     AgentSpecBackend,
			expected: true,
		},
		{
			name:     "exact match",
			agent:    Agent{Specialization: AgentSpecBackend},
			spec:     AgentSpecBackend,
			expected: true,
		},
		{
			name:     "mismatch",
			agent:    Agent{Specialization: AgentSpecFrontend},
			spec:     AgentSpecBackend,
			expected: false,
		},
		{
			name:     "general matches general",
			agent:    Agent{Specialization: AgentSpecGeneral},
			spec:     AgentSpecGeneral,
			expected: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.agent.MatchesSpecialization(tt.spec); got != tt.expected {
				t.Errorf("MatchesSpecialization(%q) = %v, want %v", tt.spec, got, tt.expected)
			}
		})
	}
}

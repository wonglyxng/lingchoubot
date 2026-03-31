package runtime

import (
	"fmt"
	"strings"
)

// ValidationError represents a structured output validation failure.
type ValidationError struct {
	Role     string   `json:"role"`
	Spec     string   `json:"spec,omitempty"`
	Failures []string `json:"failures"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("output validation failed for role=%s: %s", e.Role, strings.Join(e.Failures, "; "))
}

// ValidateOutput checks the LLM output against role-specific schema constraints.
// Returns nil if valid, or a *ValidationError listing all failures.
func ValidateOutput(role, spec string, output *AgentTaskOutput) error {
	if output == nil {
		return &ValidationError{Role: role, Spec: spec, Failures: []string{"output is nil"}}
	}

	var failures []string

	// Common checks
	if output.Status == "" {
		failures = append(failures, "status is empty")
	}
	if output.Summary == "" {
		failures = append(failures, "summary is empty")
	}

	// Role-specific checks
	switch role {
	case "pm":
		failures = append(failures, validatePMOutput(output)...)
	case "supervisor":
		failures = append(failures, validateSupervisorOutput(output)...)
	case "worker":
		failures = append(failures, validateWorkerOutput(output)...)
	case "reviewer":
		failures = append(failures, validateReviewerOutput(output)...)
	}

	if len(failures) > 0 {
		return &ValidationError{Role: role, Spec: spec, Failures: failures}
	}
	return nil
}

func validatePMOutput(o *AgentTaskOutput) []string {
	var f []string
	if len(o.Phases) == 0 {
		f = append(f, "PM output must contain at least 1 phase")
	}
	if len(o.Phases) > 10 {
		f = append(f, fmt.Sprintf("PM output has %d phases (max 10)", len(o.Phases)))
	}
	if len(o.Tasks) == 0 {
		f = append(f, "PM output must contain at least 1 task")
	}
	for i, p := range o.Phases {
		if p.Name == "" {
			f = append(f, fmt.Sprintf("phase[%d].name is empty", i))
		}
	}
	for i, t := range o.Tasks {
		if t.Title == "" {
			f = append(f, fmt.Sprintf("task[%d].title is empty", i))
		}
		if t.PhaseName == "" {
			f = append(f, fmt.Sprintf("task[%d].phase_name is empty", i))
		}
	}
	return f
}

func validateSupervisorOutput(o *AgentTaskOutput) []string {
	var f []string
	if len(o.Contracts) == 0 {
		f = append(f, "Supervisor output must contain at least 1 contract")
	}
	for i, c := range o.Contracts {
		if c.TaskTitle == "" {
			f = append(f, fmt.Sprintf("contract[%d].task_title is empty", i))
		}
		if c.Scope == "" {
			f = append(f, fmt.Sprintf("contract[%d].scope is empty", i))
		}
		if len(c.DoneDefinition) < 2 {
			f = append(f, fmt.Sprintf("contract[%d].done_definition has %d items (min 2)", i, len(c.DoneDefinition)))
		}
		if len(c.AcceptanceCriteria) < 1 {
			f = append(f, fmt.Sprintf("contract[%d].acceptance_criteria is empty", i))
		}
	}
	return f
}

func validateWorkerOutput(o *AgentTaskOutput) []string {
	var f []string
	if len(o.Artifacts) == 0 {
		f = append(f, "Worker output must contain at least 1 artifact")
	}
	for i, a := range o.Artifacts {
		if a.Name == "" {
			f = append(f, fmt.Sprintf("artifact[%d].name is empty", i))
		}
		if a.ArtifactType == "" {
			f = append(f, fmt.Sprintf("artifact[%d].artifact_type is empty", i))
		}
	}
	return f
}

func validateReviewerOutput(o *AgentTaskOutput) []string {
	var f []string
	if len(o.Reviews) == 0 {
		f = append(f, "Reviewer output must contain at least 1 review")
	}
	for i, r := range o.Reviews {
		if r.Verdict == "" {
			f = append(f, fmt.Sprintf("review[%d].verdict is empty", i))
		} else if r.Verdict != "approved" && r.Verdict != "needs_revision" {
			f = append(f, fmt.Sprintf("review[%d].verdict=%q (must be approved|needs_revision)", i, r.Verdict))
		}
		if len(r.Findings) < 1 {
			f = append(f, fmt.Sprintf("review[%d].findings is empty", i))
		}
	}
	return f
}

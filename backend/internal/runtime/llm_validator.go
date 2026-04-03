package runtime

import (
	"fmt"
	"strings"
)

var validArtifactTypes = map[string]struct{}{
	"prd":             {},
	"design":          {},
	"api_spec":        {},
	"schema_sql":      {},
	"source_code":     {},
	"test_report":     {},
	"deployment_plan": {},
	"release_note":    {},
	"other":           {},
}

var analysisTaskKeywords = []string{"可行性", "需求", "分析", "评估", "调研", "方案", "规划", "计划", "梳理", "编写", "文档", "设计", "prd"}
var testingTaskKeywords = []string{"测试", "验证", "回归", "qa", "test"}
var placeholderMarkers = []string{"待补充", "todo", "tbd", "占位", "mock qa worker agent", "mock worker agent", "mock reviewer agent", "由 doc_generator 工具自动生成"}

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

// ValidateOutputForInput adds task-aware validation on top of the role schema checks.
func ValidateOutputForInput(role, spec string, input *AgentTaskInput, output *AgentTaskOutput) error {
	var failures []string
	if err := ValidateOutput(role, spec, output); err != nil {
		if ve, ok := err.(*ValidationError); ok {
			failures = append(failures, ve.Failures...)
		} else {
			failures = append(failures, err.Error())
		}
	}

	switch role {
	case "worker":
		failures = append(failures, validateWorkerOutputAgainstInput(input, output)...)
	case "reviewer":
		failures = append(failures, validateReviewerOutputAgainstInput(input, output)...)
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
		} else if _, ok := validArtifactTypes[a.ArtifactType]; !ok {
			f = append(f, fmt.Sprintf("artifact[%d].artifact_type=%q is invalid", i, a.ArtifactType))
		}
		if strings.TrimSpace(a.Content) == "" {
			f = append(f, fmt.Sprintf("artifact[%d].content is empty", i))
		}
		if strings.TrimSpace(a.ContentType) == "" {
			f = append(f, fmt.Sprintf("artifact[%d].content_type is empty", i))
		}
	}
	return f
}

func validateReviewerOutput(o *AgentTaskOutput) []string {
	var f []string
	if len(o.Reviews) != 1 {
		f = append(f, fmt.Sprintf("Reviewer output must contain exactly 1 review, got %d", len(o.Reviews)))
	}
	for i, r := range o.Reviews {
		if r.Verdict == "" {
			f = append(f, fmt.Sprintf("review[%d].verdict is empty", i))
		} else if r.Verdict != "approved" && r.Verdict != "needs_revision" {
			f = append(f, fmt.Sprintf("review[%d].verdict=%q (must be approved|needs_revision)", i, r.Verdict))
		}
		if strings.TrimSpace(r.TemplateKey) == "" {
			f = append(f, fmt.Sprintf("review[%d].template_key is empty", i))
		}
		if r.PassThreshold <= 0 {
			f = append(f, fmt.Sprintf("review[%d].pass_threshold must be positive", i))
		}
		if r.TotalScore < 0 || r.TotalScore > 100 {
			f = append(f, fmt.Sprintf("review[%d].total_score=%d (must be between 0 and 100)", i, r.TotalScore))
		}
		if len(r.HardGateResults) == 0 {
			f = append(f, fmt.Sprintf("review[%d].hard_gate_results is empty", i))
		}
		for j, gate := range r.HardGateResults {
			if strings.TrimSpace(gate.Key) == "" {
				f = append(f, fmt.Sprintf("review[%d].hard_gate_results[%d].key is empty", i, j))
			}
			if strings.TrimSpace(gate.Reason) == "" {
				f = append(f, fmt.Sprintf("review[%d].hard_gate_results[%d].reason is empty", i, j))
			}
		}
		if len(r.ScoreItems) == 0 {
			f = append(f, fmt.Sprintf("review[%d].score_items is empty", i))
		}
		for j, item := range r.ScoreItems {
			if strings.TrimSpace(item.Key) == "" {
				f = append(f, fmt.Sprintf("review[%d].score_items[%d].key is empty", i, j))
			}
			if strings.TrimSpace(item.Name) == "" {
				f = append(f, fmt.Sprintf("review[%d].score_items[%d].name is empty", i, j))
			}
			if item.Weight <= 0 {
				f = append(f, fmt.Sprintf("review[%d].score_items[%d].weight must be positive", i, j))
			}
			if item.MaxScore <= 0 {
				f = append(f, fmt.Sprintf("review[%d].score_items[%d].max_score must be positive", i, j))
			}
			if item.Score < 0 || item.Score > item.MaxScore {
				f = append(f, fmt.Sprintf("review[%d].score_items[%d].score=%d exceeds range [0,%d]", i, j, item.Score, item.MaxScore))
			}
			if strings.TrimSpace(item.Reason) == "" {
				f = append(f, fmt.Sprintf("review[%d].score_items[%d].reason is empty", i, j))
			}
		}
		if r.Verdict == "needs_revision" && len(r.MustFixItems) == 0 {
			f = append(f, fmt.Sprintf("review[%d].must_fix_items is empty for needs_revision verdict", i))
		}
		if len(r.Findings) < 2 {
			f = append(f, fmt.Sprintf("review[%d].findings has %d items (min 2)", i, len(r.Findings)))
		}
		if len(r.Recommendations) < 1 {
			f = append(f, fmt.Sprintf("review[%d].recommendations is empty", i))
		}
	}
	return f
}

func validateWorkerOutputAgainstInput(input *AgentTaskInput, output *AgentTaskOutput) []string {
	if input == nil || input.Task == nil || output == nil {
		return nil
	}

	var failures []string
	taskText := input.Task.Title + " " + input.Task.Description
	analysisTask := containsAnyFold(taskText, analysisTaskKeywords)
	testingTask := !analysisTask && containsAnyFold(taskText, testingTaskKeywords)
	hasProjectBinding := func(content string) bool {
		if strings.TrimSpace(content) == "" {
			return false
		}
		if containsFold(content, input.Task.Title) {
			return true
		}
		return input.Project != nil && containsFold(content, input.Project.Name)
	}

	hasAnalysisArtifact := false
	hasTestReport := false
	for i, artifact := range output.Artifacts {
		if containsAnyFold(artifact.Content, placeholderMarkers) {
			failures = append(failures, fmt.Sprintf("artifact[%d] contains placeholder/mock markers", i))
		}
		if artifact.ArtifactType == "prd" || artifact.ArtifactType == "design" {
			hasAnalysisArtifact = true
		}
		if artifact.ArtifactType == "test_report" {
			hasTestReport = true
		}
		if analysisTask {
			if artifact.ArtifactType == "test_report" || artifact.ArtifactType == "source_code" {
				failures = append(failures, fmt.Sprintf("artifact[%d].artifact_type=%q does not match analysis task", i, artifact.ArtifactType))
			}
			if !hasProjectBinding(artifact.Content) {
				failures = append(failures, fmt.Sprintf("artifact[%d] is not clearly bound to the current project/task", i))
			}
		}
	}

	if analysisTask && !hasAnalysisArtifact {
		failures = append(failures, "analysis task must produce prd or design artifact")
	}
	if testingTask && !hasTestReport {
		failures = append(failures, "testing task must produce test_report artifact")
	}

	return failures
}

func validateReviewerOutputAgainstInput(input *AgentTaskInput, output *AgentTaskOutput) []string {
	if input == nil || input.Task == nil || output == nil {
		return nil
	}

	var failures []string
	taskText := input.Task.Title + " " + input.Task.Description
	analysisTask := containsAnyFold(taskText, analysisTaskKeywords)
	allTestReports := len(input.Artifacts) > 0
	hasPlaceholderArtifacts := false

	for _, artifact := range input.Artifacts {
		if artifact.ArtifactType != "test_report" {
			allTestReports = false
		}
		if containsAnyFold(artifact.Content, placeholderMarkers) {
			hasPlaceholderArtifacts = true
		}
	}

	for i, review := range output.Reviews {
		if input.Contract != nil && input.Contract.ReviewPolicy != nil {
			failures = append(failures, validateReviewScorecardCoverage(i, input.Contract.ReviewPolicy, review)...)
		}

		if review.Verdict != "approved" {
			if review.Verdict == "needs_revision" && len(review.MustFixItems) == 0 {
				failures = append(failures, fmt.Sprintf("review[%d] needs_revision must include must_fix_items", i))
			}
			continue
		}
		if len(input.Artifacts) == 0 {
			failures = append(failures, fmt.Sprintf("review[%d] cannot approve empty artifact list", i))
		}
		if hasPlaceholderArtifacts {
			failures = append(failures, fmt.Sprintf("review[%d] approved placeholder/mock artifacts", i))
		}
		if analysisTask && allTestReports {
			failures = append(failures, fmt.Sprintf("review[%d] approved test_report artifacts for analysis task", i))
		}
	}

	return failures
}

func validateReviewScorecardCoverage(index int, policy *ReviewPolicyCtx, review ReviewAction) []string {
	if policy == nil {
		return nil
	}
	var failures []string
	if review.TemplateKey != policy.TemplateKey {
		failures = append(failures, fmt.Sprintf("review[%d].template_key=%q does not match policy %q", index, review.TemplateKey, policy.TemplateKey))
	}
	if review.PassThreshold != policy.PassThreshold {
		failures = append(failures, fmt.Sprintf("review[%d].pass_threshold=%d does not match policy %d", index, review.PassThreshold, policy.PassThreshold))
	}

	gateResults := map[string]HardGateResultAction{}
	hasFailedGate := false
	for _, gate := range review.HardGateResults {
		gateResults[gate.Key] = gate
		if !gate.Passed {
			hasFailedGate = true
		}
	}
	for _, expected := range policy.HardGates {
		if _, ok := gateResults[expected.Key]; !ok {
			failures = append(failures, fmt.Sprintf("review[%d] missing hard gate result for %q", index, expected.Key))
		}
	}

	scoreResults := map[string]ScoreItemResultAction{}
	for _, item := range review.ScoreItems {
		scoreResults[item.Key] = item
	}
	for _, expected := range policy.ScoreItems {
		got, ok := scoreResults[expected.Key]
		if !ok {
			failures = append(failures, fmt.Sprintf("review[%d] missing score item result for %q", index, expected.Key))
			continue
		}
		if got.Weight != expected.Weight {
			failures = append(failures, fmt.Sprintf("review[%d].score_items[%q].weight=%d does not match policy %d", index, expected.Key, got.Weight, expected.Weight))
		}
	}

	if review.Verdict == "approved" && hasFailedGate {
		failures = append(failures, fmt.Sprintf("review[%d] approved despite failed hard gate", index))
	}
	if review.Verdict == "approved" && review.TotalScore < policy.PassThreshold {
		failures = append(failures, fmt.Sprintf("review[%d] approved with total_score=%d below threshold=%d", index, review.TotalScore, policy.PassThreshold))
	}
	if review.Verdict == "needs_revision" && len(review.MustFixItems) == 0 {
		failures = append(failures, fmt.Sprintf("review[%d] needs_revision without must_fix_items", index))
	}
	return failures
}

func containsAnyFold(s string, keywords []string) bool {
	for _, keyword := range keywords {
		if containsFold(s, keyword) {
			return true
		}
	}
	return false
}

func containsFold(s, keyword string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(keyword))
}

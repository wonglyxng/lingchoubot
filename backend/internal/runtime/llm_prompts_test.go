package runtime

import (
	"strings"
	"testing"
)

func TestGetPromptVersion_KnownRoles(t *testing.T) {
	roles := []string{"pm", "supervisor", "worker", "reviewer"}
	for _, role := range roles {
		pv := GetPromptVersion(role, "")
		if pv.Role != role {
			t.Errorf("expected role=%s, got %s", role, pv.Role)
		}
		if pv.Version == "" || pv.Version == "0.0.0" {
			t.Errorf("expected non-zero version for role=%s, got %s", role, pv.Version)
		}
		if pv.ChangeLog == "" {
			t.Errorf("expected non-empty changelog for role=%s", role)
		}
	}
}

func TestGetPromptVersion_UnknownRole(t *testing.T) {
	pv := GetPromptVersion("unknown", "")
	if pv.Version != "0.0.0" {
		t.Errorf("expected 0.0.0 for unknown role, got %s", pv.Version)
	}
}

func TestBuildSystemPrompt_AllRolesPresent(t *testing.T) {
	cases := []struct {
		role string
		spec string
	}{
		{"pm", ""},
		{"supervisor", ""},
		{"worker", ""},
		{"worker", "backend"},
		{"worker", "frontend"},
		{"worker", "qa"},
		{"reviewer", ""},
		{"unknown", ""},
	}
	for _, c := range cases {
		p := buildSystemPrompt(c.role, c.spec)
		if p == "" {
			t.Errorf("expected non-empty prompt for role=%s spec=%s", c.role, c.spec)
		}
	}
}

func TestBuildUserPrompt_NonEmpty(t *testing.T) {
	input := &AgentTaskInput{
		RunID:       "r1",
		AgentID:     "a1",
		AgentRole:   "pm",
		Instruction: "test",
	}
	prompt, err := buildUserPrompt(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prompt == "" {
		t.Fatal("expected non-empty user prompt")
	}
}

func TestEvalSamples_Count(t *testing.T) {
	samples := EvalSamples()
	if len(samples) < 4 {
		t.Errorf("expected at least 4 eval samples, got %d", len(samples))
	}
	roles := make(map[string]bool)
	for _, s := range samples {
		roles[s.Role] = true
		if s.Name == "" {
			t.Error("sample has empty name")
		}
		if s.Input == nil {
			t.Errorf("sample %s has nil input", s.Name)
		}
	}
	for _, r := range []string{"pm", "supervisor", "worker", "reviewer"} {
		if !roles[r] {
			t.Errorf("missing eval sample for role=%s", r)
		}
	}
}

func TestEvalSamples_Validation(t *testing.T) {
	samples := EvalSamples()
	for _, s := range samples {
		output := validEvalOutputForSample(s)
		if valErr := ValidateOutput(s.Role, s.Spec, output); valErr != nil {
			t.Errorf("sample %s: deterministic output fails validation: %v", s.Name, valErr)
		}
	}
}

func TestReviewerPrompt_ContainsScorecardSchema(t *testing.T) {
	prompt := buildSystemPrompt("reviewer", "")
	required := []string{
		`"template_key"`,
		`"pass_threshold"`,
		`"total_score"`,
		`"hard_gate_results"`,
		`"score_items"`,
		`"must_fix_items"`,
	}
	for _, item := range required {
		if !containsPromptText(prompt, item) {
			t.Fatalf("reviewer prompt missing %s", item)
		}
	}
}

func TestSupervisorPrompt_ContainsReviewPolicyOverrideRules(t *testing.T) {
	prompt := buildSystemPrompt("supervisor", "")
	required := []string{
		"默认情况下不要输出 review_policy",
		"只有任务存在明确专项要求时，才输出 review_policy",
		"新增 score_items 最多 2 个",
		"按 weight 降序、key 升序保留前 2 个，并记录裁剪痕迹",
	}
	for _, item := range required {
		if !containsPromptText(prompt, item) {
			t.Fatalf("supervisor prompt missing %s", item)
		}
	}
}

func validEvalOutputForSample(sample EvalSample) *AgentTaskOutput {
	switch sample.Role {
	case "pm":
		return &AgentTaskOutput{
			Status:  OutputStatusSuccess,
			Summary: "项目已拆解",
			Phases:  []PhaseAction{{Name: "需求分析", Description: "梳理需求", SortOrder: 1}},
			Tasks:   []TaskAction{{PhaseName: "需求分析", Title: "需求梳理", Description: "整理需求", Priority: 3}},
		}
	case "supervisor":
		return &AgentTaskOutput{
			Status:  OutputStatusSuccess,
			Summary: "契约已创建",
			Contracts: []ContractAction{{
				TaskTitle:          sample.Input.Task.Title,
				Scope:              "完成任务范围",
				NonGoals:           []string{"不做无关扩展"},
				DoneDefinition:     []string{"交付物生成完成", "通过基础质量检查"},
				VerificationSteps:  []string{"执行验证"},
				AcceptanceCriteria: []string{"满足任务要求"},
			}},
			Assignments: []AssignmentAction{{TaskTitle: sample.Input.Task.Title, AgentRole: "worker", Role: "executor", Note: "执行任务"}},
			Transitions: []TransitionAction{{TaskTitle: sample.Input.Task.Title, NewStatus: "assigned"}},
		}
	case "worker":
		return &AgentTaskOutput{
			Status:  OutputStatusNeedsReview,
			Summary: "工件已产出",
			Artifacts: []ArtifactAction{{
				Name:         "user_register_handler.go",
				ArtifactType: "source_code",
				Description:  "注册接口实现",
				URI:          "artifact://eval/user_register_handler.go",
				ContentType:  "text/x-go",
				SizeBytes:    24,
				Content:      "package handler\nfunc Register() {}",
			}},
			Handoffs:    []HandoffAction{{Summary: "交接完成", CompletedItems: []string{"代码已提交"}, PendingItems: []string{"等待评审"}, Risks: []string{"需继续补充测试"}, NextSteps: []string{"提交评审"}}},
			Transitions: []TransitionAction{{TaskTitle: sample.Input.Task.Title, NewStatus: "in_review"}},
		}
	case "reviewer":
		return &AgentTaskOutput{
			Status:  OutputStatusSuccess,
			Summary: "评审完成",
			Reviews: []ReviewAction{{
				Verdict:       "approved",
				Summary:       "实现合理",
				TemplateKey:   "backend_v1",
				PassThreshold: 80,
				TotalScore:    85,
				HardGateResults: []HardGateResultAction{
					{Key: "goal_match", Passed: true, Reason: "工件与任务目标一致"},
				},
				ScoreItems: []ScoreItemResultAction{
					{Key: "functional_correctness", Name: "功能正确性", Weight: 35, Score: 35, MaxScore: 35, Reason: "功能实现完整"},
				},
				Findings:        []string{"接口边界清晰", "工件与任务一致"},
				Recommendations: []string{"补充更多测试覆盖"},
			}},
		}
	default:
		return &AgentTaskOutput{Status: OutputStatusSuccess, Summary: "ok"}
	}
}

func containsPromptText(prompt, target string) bool {
	return len(prompt) > 0 && len(target) > 0 && strings.Contains(prompt, target)
}

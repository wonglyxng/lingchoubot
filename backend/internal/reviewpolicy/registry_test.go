package reviewpolicy

import "testing"

func TestResolvePolicy_DefaultPRDTemplate(t *testing.T) {
	policy, err := ResolvePolicy("prd", nil)
	if err != nil {
		t.Fatalf("ResolvePolicy: %v", err)
	}
	if policy.TemplateKey != "prd_v1" {
		t.Fatalf("template = %q, want %q", policy.TemplateKey, "prd_v1")
	}
	if policy.TaskCategory != "prd" {
		t.Fatalf("task category = %q, want %q", policy.TaskCategory, "prd")
	}
	if policy.PassThreshold != 80 {
		t.Fatalf("threshold = %d, want 80", policy.PassThreshold)
	}
	if len(policy.HardGates) == 0 {
		t.Fatal("expected hard gates")
	}
	if len(policy.ScoreItems) == 0 {
		t.Fatal("expected score items")
	}
}

func TestResolvePolicy_RejectsThresholdOverrideOutsideBounds(t *testing.T) {
	_, err := ResolvePolicy("backend", map[string]any{
		"pass_threshold": 95,
	})
	if err == nil {
		t.Fatal("expected threshold override error")
	}
}

func TestResolvePolicy_RejectsWeightSumMismatch(t *testing.T) {
	_, err := ResolvePolicy("qa", map[string]any{
		"score_items": []map[string]any{
			{
				"key":    "coverage",
				"name":   "覆盖度",
				"weight": 60,
			},
			{
				"key":    "reproducibility",
				"name":   "可复现性",
				"weight": 20,
			},
		},
	})
	if err == nil {
		t.Fatal("expected score item weight sum error")
	}
}

func TestResolvePolicy_TrimsExtraScoreItemsDeterministically(t *testing.T) {
	policy, err := ResolvePolicy("architecture", map[string]any{
		"score_items": []map[string]any{
			{
				"key":    "technical_feasibility",
				"name":   "技术可行性",
				"weight": 20,
			},
			{
				"key":    "tradeoff_reasoning",
				"name":   "取舍说明",
				"weight": 15,
			},
			{
				"key":    "constraint_alignment",
				"name":   "约束一致性",
				"weight": 10,
			},
			{
				"key":    "implementation_guidance",
				"name":   "实施指导性",
				"weight": 20,
			},
			{
				"key":    "risk_control",
				"name":   "风险控制",
				"weight": 15,
			},
			{
				"key":    "extra_b",
				"name":   "业务一致性",
				"weight": 10,
			},
			{
				"key":    "extra_c",
				"name":   "团队协作性",
				"weight": 1,
			},
			{
				"key":    "extra_a",
				"name":   "方案落地收益",
				"weight": 10,
			},
		},
	})
	if err != nil {
		t.Fatalf("ResolvePolicy: %v", err)
	}

	if len(policy.ScoreItems) != 7 {
		t.Fatalf("score item count = %d, want 7", len(policy.ScoreItems))
	}
	if got := policy.ScoreItems[5].Key; got != "extra_a" {
		t.Fatalf("score_items[5] = %q, want extra_a", got)
	}
	if got := policy.ScoreItems[6].Key; got != "extra_b" {
		t.Fatalf("score_items[6] = %q, want extra_b", got)
	}

	if policy.ResolutionTrace == nil || policy.ResolutionTrace.ExtraScoreItemsTrim == nil {
		t.Fatal("expected extra score item trim trace")
	}
	trace := policy.ResolutionTrace.ExtraScoreItemsTrim
	if trace.MaxExtraItems != 2 {
		t.Fatalf("trace.MaxExtraItems = %d, want 2", trace.MaxExtraItems)
	}
	if trace.SelectionRule != "weight_desc_key_asc" {
		t.Fatalf("trace.SelectionRule = %q, want weight_desc_key_asc", trace.SelectionRule)
	}
	if len(trace.KeptExtraScoreItems) != 2 {
		t.Fatalf("kept extra score items = %d, want 2", len(trace.KeptExtraScoreItems))
	}
	if trace.KeptExtraScoreItems[0].Key != "extra_a" || trace.KeptExtraScoreItems[1].Key != "extra_b" {
		t.Fatalf("kept extra score items = %#v, want extra_a then extra_b", trace.KeptExtraScoreItems)
	}
	if len(trace.DroppedExtraScoreItems) != 1 || trace.DroppedExtraScoreItems[0].Key != "extra_c" {
		t.Fatalf("dropped extra score items = %#v, want [extra_c]", trace.DroppedExtraScoreItems)
	}
}

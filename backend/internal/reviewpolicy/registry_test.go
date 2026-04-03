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

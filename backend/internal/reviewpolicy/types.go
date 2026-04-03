package reviewpolicy

// HardGate defines a pass/fail checkpoint that cannot be compensated for by score.
type HardGate struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ScoreItem defines a weighted scoring dimension used after hard gates pass.
type ScoreItem struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Weight      int    `json:"weight"`
	Description string `json:"description,omitempty"`
}

// OverridePolicy constrains how much a task-level override may diverge from defaults.
type OverridePolicy struct {
	ThresholdMin               int  `json:"threshold_min"`
	ThresholdMax               int  `json:"threshold_max"`
	MaxExtraItems              int  `json:"max_extra_items"`
	AllowWeightAdjustment      bool `json:"allow_weight_adjustment"`
	AllowRemoveGlobalHardGates bool `json:"allow_remove_global_hard_gates"`
}

// Template is the default review scorecard definition for one task category.
type Template struct {
	TemplateKey    string         `json:"template_key"`
	TaskCategory   string         `json:"task_category"`
	PassThreshold  int            `json:"pass_threshold"`
	HardGates      []HardGate     `json:"hard_gates"`
	ScoreItems     []ScoreItem    `json:"score_items"`
	OverridePolicy OverridePolicy `json:"override_policy"`
}

// PolicyOverride stores contract-level review policy customizations.
type PolicyOverride struct {
	PassThreshold *int        `json:"pass_threshold,omitempty"`
	HardGates     []HardGate  `json:"hard_gates,omitempty"`
	ScoreItems    []ScoreItem `json:"score_items,omitempty"`
}

// ResolvedPolicy is the final policy handed to runtime and persisted in metadata.
type ResolvedPolicy struct {
	TemplateKey   string      `json:"template_key"`
	TaskCategory  string      `json:"task_category"`
	PassThreshold int         `json:"pass_threshold"`
	HardGates     []HardGate  `json:"hard_gates"`
	ScoreItems    []ScoreItem `json:"score_items"`
}

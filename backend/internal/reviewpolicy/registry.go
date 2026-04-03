package reviewpolicy

import (
	"encoding/json"
	"fmt"
	"sort"
)

var defaultTemplates = map[string]Template{
	"prd": {
		TemplateKey:   "prd_v1",
		TaskCategory:  "prd",
		PassThreshold: 80,
		HardGates: append(globalHardGates(),
			HardGate{Key: "scope_defined", Name: "范围明确"},
			HardGate{Key: "non_goals_defined", Name: "非目标明确"},
			HardGate{Key: "core_scenarios_complete", Name: "核心场景齐全"},
			HardGate{Key: "acceptance_testable", Name: "验收标准可验证"},
			HardGate{Key: "risks_listed", Name: "风险与未决问题已列出"},
		),
		ScoreItems: []ScoreItem{
			{Key: "completeness", Name: "完整性", Weight: 25},
			{Key: "boundary_clarity", Name: "边界清晰度", Weight: 20},
			{Key: "consistency", Name: "内部一致性", Weight: 15},
			{Key: "executability", Name: "可执行性", Weight: 20},
			{Key: "acceptance_testability", Name: "验收可测性", Weight: 20},
		},
		OverridePolicy: defaultOverridePolicy(),
	},
	"architecture": {
		TemplateKey:   "architecture_v1",
		TaskCategory:  "architecture",
		PassThreshold: 80,
		HardGates: append(globalHardGates(),
			HardGate{Key: "problem_context_clear", Name: "问题背景明确"},
			HardGate{Key: "solution_conclusion_clear", Name: "方案结论明确"},
			HardGate{Key: "dependency_estimated", Name: "依赖与资源评估明确"},
			HardGate{Key: "major_risks_listed", Name: "主要风险明确"},
			HardGate{Key: "tradeoff_recommendation_clear", Name: "推荐方案及取舍明确"},
		),
		ScoreItems: []ScoreItem{
			{Key: "technical_feasibility", Name: "技术可行性", Weight: 25},
			{Key: "tradeoff_reasoning", Name: "取舍说明", Weight: 20},
			{Key: "constraint_alignment", Name: "约束一致性", Weight: 15},
			{Key: "implementation_guidance", Name: "实施指导性", Weight: 20},
			{Key: "risk_control", Name: "风险控制", Weight: 20},
		},
		OverridePolicy: defaultOverridePolicy(),
	},
	"backend": {
		TemplateKey:   "backend_v1",
		TaskCategory:  "backend",
		PassThreshold: 80,
		HardGates: append(globalHardGates(),
			HardGate{Key: "scope_implemented", Name: "实现覆盖任务范围"},
			HardGate{Key: "no_placeholder_code", Name: "无明显占位代码"},
			HardGate{Key: "error_handling_present", Name: "关键错误处理存在"},
			HardGate{Key: "contract_aligned", Name: "接口与数据结构与契约一致"},
			HardGate{Key: "required_tests_present", Name: "契约要求测试时必须有测试证据"},
		),
		ScoreItems: []ScoreItem{
			{Key: "functional_correctness", Name: "功能正确性", Weight: 35},
			{Key: "contract_fitness", Name: "契约符合度", Weight: 20},
			{Key: "robustness", Name: "健壮性", Weight: 15},
			{Key: "readability", Name: "代码可读性", Weight: 15},
			{Key: "testing_validation", Name: "测试与验证", Weight: 15},
		},
		OverridePolicy: defaultOverridePolicy(),
	},
	"frontend": {
		TemplateKey:   "frontend_v1",
		TaskCategory:  "frontend",
		PassThreshold: 80,
		HardGates: append(globalHardGates(),
			HardGate{Key: "goal_fulfilled", Name: "页面或组件满足任务目标"},
			HardGate{Key: "state_complete", Name: "状态完整覆盖"},
			HardGate{Key: "api_real", Name: "真实 API 对接符合要求"},
			HardGate{Key: "no_placeholder_ui", Name: "无明显占位 UI"},
			HardGate{Key: "responsive_accessible_baseline", Name: "基础响应式与可访问性满足最低要求"},
		),
		ScoreItems: []ScoreItem{
			{Key: "requirement_fit", Name: "需求符合度", Weight: 25},
			{Key: "interaction_integrity", Name: "交互完整性", Weight: 20},
			{Key: "state_handling", Name: "状态处理", Weight: 20},
			{Key: "code_quality", Name: "代码质量", Weight: 15},
			{Key: "usability", Name: "可用性", Weight: 10},
			{Key: "accessibility", Name: "可访问性", Weight: 10},
		},
		OverridePolicy: defaultOverridePolicy(),
	},
	"qa": {
		TemplateKey:   "qa_v1",
		TaskCategory:  "qa",
		PassThreshold: 80,
		HardGates: append(globalHardGates(),
			HardGate{Key: "scope_explicit", Name: "测试范围明确"},
			HardGate{Key: "steps_reproducible", Name: "执行步骤可复现"},
			HardGate{Key: "evidence_present", Name: "结果有证据"},
			HardGate{Key: "risks_recorded", Name: "缺陷与风险有记录"},
			HardGate{Key: "conclusion_clear", Name: "结论明确"},
		),
		ScoreItems: []ScoreItem{
			{Key: "coverage", Name: "覆盖度", Weight: 30},
			{Key: "reproducibility", Name: "复现性", Weight: 20},
			{Key: "defect_quality", Name: "缺陷质量", Weight: 20},
			{Key: "risk_targeting", Name: "风险针对性", Weight: 15},
			{Key: "automation_value", Name: "自动化价值", Weight: 15},
		},
		OverridePolicy: defaultOverridePolicy(),
	},
	"release": {
		TemplateKey:   "release_v1",
		TaskCategory:  "release",
		PassThreshold: 80,
		HardGates: append(globalHardGates(),
			HardGate{Key: "steps_clear", Name: "发布步骤明确"},
			HardGate{Key: "rollback_clear", Name: "回滚方案明确"},
			HardGate{Key: "changes_clear", Name: "配置与依赖变更明确"},
			HardGate{Key: "verification_clear", Name: "验证步骤明确"},
			HardGate{Key: "risk_monitoring_clear", Name: "风险与监控方案明确"},
		),
		ScoreItems: []ScoreItem{
			{Key: "safety", Name: "安全性", Weight: 30},
			{Key: "completeness", Name: "完整性", Weight: 20},
			{Key: "operability", Name: "可操作性", Weight: 20},
			{Key: "observability", Name: "可观测性", Weight: 15},
			{Key: "clarity", Name: "表达清晰度", Weight: 15},
		},
		OverridePolicy: defaultOverridePolicy(),
	},
}

func ResolvePolicy(taskCategory string, override map[string]any) (*ResolvedPolicy, error) {
	template, ok := defaultTemplates[taskCategory]
	if !ok {
		return nil, fmt.Errorf("unsupported task category %q", taskCategory)
	}

	resolved := &ResolvedPolicy{
		TemplateKey:   template.TemplateKey,
		TaskCategory:  template.TaskCategory,
		PassThreshold: template.PassThreshold,
		HardGates:     cloneHardGates(template.HardGates),
		ScoreItems:    cloneScoreItems(template.ScoreItems),
	}

	if len(override) == 0 {
		if err := validateResolvedPolicy(resolved); err != nil {
			return nil, err
		}
		return resolved, nil
	}

	parsed, err := parseOverride(override)
	if err != nil {
		return nil, err
	}

	if parsed.PassThreshold != nil {
		if *parsed.PassThreshold < template.OverridePolicy.ThresholdMin || *parsed.PassThreshold > template.OverridePolicy.ThresholdMax {
			return nil, fmt.Errorf("pass_threshold %d out of bounds [%d,%d]", *parsed.PassThreshold, template.OverridePolicy.ThresholdMin, template.OverridePolicy.ThresholdMax)
		}
		resolved.PassThreshold = *parsed.PassThreshold
	}

	if len(parsed.HardGates) > 0 {
		resolved.HardGates = append(resolved.HardGates, parsed.HardGates...)
	}

	if len(parsed.ScoreItems) > 0 {
		items, trace, err := mergeScoreItems(template.ScoreItems, parsed.ScoreItems, template.OverridePolicy)
		if err != nil {
			return nil, err
		}
		resolved.ScoreItems = items
		resolved.ResolutionTrace = mergeResolutionTrace(resolved.ResolutionTrace, trace)
	}

	if err := validateResolvedPolicy(resolved); err != nil {
		return nil, err
	}
	return resolved, nil
}

func parseOverride(raw map[string]any) (*PolicyOverride, error) {
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal override: %w", err)
	}
	var override PolicyOverride
	if err := json.Unmarshal(b, &override); err != nil {
		return nil, fmt.Errorf("unmarshal override: %w", err)
	}
	return &override, nil
}

func mergeScoreItems(defaults, overrides []ScoreItem, policy OverridePolicy) ([]ScoreItem, *ResolutionTrace, error) {
	items := cloneScoreItems(defaults)
	indexByKey := make(map[string]int, len(items))
	for i, item := range items {
		indexByKey[item.Key] = i
	}

	extraItems := make([]ScoreItem, 0)
	for _, item := range overrides {
		if item.Key == "" {
			return nil, nil, fmt.Errorf("score item key is required")
		}
		if item.Name == "" {
			return nil, nil, fmt.Errorf("score item %q name is required", item.Key)
		}
		if item.Weight <= 0 {
			return nil, nil, fmt.Errorf("score item %q weight must be positive", item.Key)
		}
		if idx, ok := indexByKey[item.Key]; ok {
			if !policy.AllowWeightAdjustment {
				return nil, nil, fmt.Errorf("score item %q cannot be adjusted", item.Key)
			}
			items[idx] = item
			continue
		}
		extraItems = append(extraItems, item)
	}

	sort.Slice(extraItems, func(i, j int) bool {
		if extraItems[i].Weight != extraItems[j].Weight {
			return extraItems[i].Weight > extraItems[j].Weight
		}
		if extraItems[i].Key != extraItems[j].Key {
			return extraItems[i].Key < extraItems[j].Key
		}
		return extraItems[i].Name < extraItems[j].Name
	})

	keptExtraItems := cloneScoreItems(extraItems)
	droppedExtraItems := []ScoreItem(nil)
	if len(extraItems) > policy.MaxExtraItems {
		keptExtraItems = cloneScoreItems(extraItems[:policy.MaxExtraItems])
		droppedExtraItems = cloneScoreItems(extraItems[policy.MaxExtraItems:])
	}

	for _, item := range keptExtraItems {
		items = append(items, item)
		indexByKey[item.Key] = len(items) - 1
	}

	return items, buildResolutionTrace(policy, extraItems, keptExtraItems, droppedExtraItems), nil
}

func validateResolvedPolicy(policy *ResolvedPolicy) error {
	if policy == nil {
		return fmt.Errorf("policy is nil")
	}
	if policy.TemplateKey == "" {
		return fmt.Errorf("template_key is required")
	}
	if policy.TaskCategory == "" {
		return fmt.Errorf("task_category is required")
	}
	if len(policy.HardGates) == 0 {
		return fmt.Errorf("hard_gates must not be empty")
	}
	if len(policy.ScoreItems) == 0 {
		return fmt.Errorf("score_items must not be empty")
	}
	totalWeight := 0
	for _, item := range policy.ScoreItems {
		totalWeight += item.Weight
	}
	if totalWeight != 100 {
		return fmt.Errorf("score item weights sum to %d, want 100", totalWeight)
	}
	return nil
}

func defaultOverridePolicy() OverridePolicy {
	return OverridePolicy{
		ThresholdMin:               75,
		ThresholdMax:               90,
		MaxExtraItems:              2,
		AllowWeightAdjustment:      true,
		AllowRemoveGlobalHardGates: false,
	}
}

func globalHardGates() []HardGate {
	return []HardGate{
		{Key: "goal_match", Name: "工件与任务目标一致"},
		{Key: "no_placeholder", Name: "不能存在模板占位或空洞内容"},
		{Key: "project_binding", Name: "必须绑定当前项目与任务"},
	}
}

func cloneHardGates(items []HardGate) []HardGate {
	out := make([]HardGate, len(items))
	copy(out, items)
	return out
}

func cloneScoreItems(items []ScoreItem) []ScoreItem {
	out := make([]ScoreItem, len(items))
	copy(out, items)
	return out
}

func buildResolutionTrace(policy OverridePolicy, requestedExtraItems, keptExtraItems, droppedExtraItems []ScoreItem) *ResolutionTrace {
	if len(droppedExtraItems) == 0 {
		return nil
	}
	return &ResolutionTrace{
		ExtraScoreItemsTrim: &ExtraScoreItemsTrimTrace{
			SelectionRule:            "weight_desc_key_asc",
			MaxExtraItems:            policy.MaxExtraItems,
			RequestedExtraScoreItems: cloneScoreItems(requestedExtraItems),
			KeptExtraScoreItems:      cloneScoreItems(keptExtraItems),
			DroppedExtraScoreItems:   cloneScoreItems(droppedExtraItems),
		},
	}
}

func mergeResolutionTrace(current, next *ResolutionTrace) *ResolutionTrace {
	if current == nil {
		return next
	}
	if next == nil {
		return current
	}
	if next.ExtraScoreItemsTrim != nil {
		current.ExtraScoreItemsTrim = next.ExtraScoreItemsTrim
	}
	return current
}

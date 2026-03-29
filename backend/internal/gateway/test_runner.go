package gateway

import (
	"context"
	"fmt"
	"time"
)

// TestRunnerTool simulates running a test suite.
// In MVP this produces mock test results; future versions will execute real tests.
type TestRunnerTool struct{}

func (t *TestRunnerTool) Name() string        { return "test_runner" }
func (t *TestRunnerTool) Description() string  { return "执行测试套件并返回结果报告（MVP 模拟执行）" }
func (t *TestRunnerTool) RequiredPermissions() []string { return []string{"tool.test_runner"} }

func (t *TestRunnerTool) Execute(ctx context.Context, input map[string]any) (*ToolResult, error) {
	suiteName, _ := input["suite_name"].(string)
	targetModule, _ := input["target_module"].(string)

	if suiteName == "" {
		suiteName = "default_suite"
	}

	totalTests := 5
	passed := 5
	failed := 0
	skipped := 0
	durationSec := 2.3

	cases := []map[string]any{
		{"name": "test_basic_functionality", "status": "passed", "duration_ms": 120},
		{"name": "test_edge_cases", "status": "passed", "duration_ms": 340},
		{"name": "test_error_handling", "status": "passed", "duration_ms": 210},
		{"name": "test_integration", "status": "passed", "duration_ms": 890},
		{"name": "test_performance", "status": "passed", "duration_ms": 450},
	}

	var summary string
	if failed == 0 {
		summary = fmt.Sprintf("测试套件「%s」全部通过：%d/%d", suiteName, passed, totalTests)
	} else {
		summary = fmt.Sprintf("测试套件「%s」存在失败：%d passed, %d failed, %d skipped", suiteName, passed, failed, skipped)
	}

	return &ToolResult{
		Status: "success",
		Output: map[string]any{
			"suite_name":    suiteName,
			"target_module": targetModule,
			"total":         totalTests,
			"passed":        passed,
			"failed":        failed,
			"skipped":       skipped,
			"duration_sec":  durationSec,
			"cases":         cases,
			"summary":       summary,
			"executed_at":   time.Now().Format(time.RFC3339),
		},
	}, nil
}

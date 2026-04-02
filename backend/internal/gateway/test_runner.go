package gateway

import (
	"context"
	"fmt"
)

// TestRunnerTool is a deprecated mock tool kept only for backward-compatible types.
// It is no longer registered in the production gateway.
type TestRunnerTool struct{}

func (t *TestRunnerTool) Name() string        { return "test_runner" }
func (t *TestRunnerTool) Description() string  { return "已停用：不再返回模拟测试结果，请改用真实测试执行链路" }
func (t *TestRunnerTool) RequiredPermissions() []string { return []string{"tool.test_runner"} }

func (t *TestRunnerTool) Execute(ctx context.Context, input map[string]any) (*ToolResult, error) {
	return &ToolResult{
		Status: "failed",
		Error:  fmt.Sprintf("tool %s has been removed from the strict runtime; execute real tests outside the mock tool path", t.Name()),
	}, nil
}

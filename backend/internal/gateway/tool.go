package gateway

import "context"

// RiskLevel indicates how sensitive a tool operation is.
type RiskLevel string

const (
	RiskNormal    RiskLevel = "normal"    // 默认：正常工具调用
	RiskSensitive RiskLevel = "sensitive" // 敏感：需要显式授权
	RiskCritical  RiskLevel = "critical"  // 高危：需要审批或白名单
)

// Tool is the interface that every tool must implement.
type Tool interface {
	Name() string
	Description() string
	RequiredPermissions() []string
	Execute(ctx context.Context, input map[string]any) (*ToolResult, error)
}

// ActionAwareTool extends Tool with action-level granularity.
// Tools that implement this interface support per-action permission checking.
type ActionAwareTool interface {
	Tool
	Actions() []string                 // 可执行的动作列表，如 ["read", "write", "delete"]
	RiskLevel(action string) RiskLevel // 指定动作的风险等级
}

// ToolRequest is the unified input for a tool call through the gateway.
type ToolRequest struct {
	ToolName string         `json:"tool_name"`
	AgentID  string         `json:"agent_id"`
	TaskID   string         `json:"task_id,omitempty"`
	Action   string         `json:"action,omitempty"` // 具体动作，如 "read" / "write"
	Input    map[string]any `json:"input"`
}

// ToolResult is the unified output from a tool execution.
type ToolResult struct {
	Status string         `json:"status"`
	Output map[string]any `json:"output"`
	Error  string         `json:"error,omitempty"`
}

// ToolInfo describes a registered tool for listing purposes.
type ToolInfo struct {
	Name                string    `json:"name"`
	Description         string    `json:"description"`
	RequiredPermissions []string  `json:"required_permissions"`
	Actions             []string  `json:"actions,omitempty"`
	RiskLevel           RiskLevel `json:"risk_level"`
}

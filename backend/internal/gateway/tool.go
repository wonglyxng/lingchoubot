package gateway

import "context"

// Tool is the interface that every tool must implement.
type Tool interface {
	Name() string
	Description() string
	RequiredPermissions() []string
	Execute(ctx context.Context, input map[string]any) (*ToolResult, error)
}

// ToolRequest is the unified input for a tool call through the gateway.
type ToolRequest struct {
	ToolName string         `json:"tool_name"`
	AgentID  string         `json:"agent_id"`
	TaskID   string         `json:"task_id,omitempty"`
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
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	RequiredPermissions []string `json:"required_permissions"`
}

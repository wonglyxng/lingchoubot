package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/service"
)

// Gateway is the central dispatcher for all tool calls.
// It handles permission checking, execution, recording, and auditing.
type Gateway struct {
	tools    map[string]Tool
	callSvc  *service.ToolCallService
	agentSvc *service.AgentService
	auditSvc *service.AuditService
	logger   *slog.Logger
}

func NewGateway(callSvc *service.ToolCallService, agentSvc *service.AgentService, auditSvc *service.AuditService, logger *slog.Logger) *Gateway {
	return &Gateway{
		tools:    make(map[string]Tool),
		callSvc:  callSvc,
		agentSvc: agentSvc,
		auditSvc: auditSvc,
		logger:   logger,
	}
}

func (g *Gateway) RegisterTool(tool Tool) {
	g.tools[tool.Name()] = tool
}

func (g *Gateway) ListTools() []ToolInfo {
	list := make([]ToolInfo, 0, len(g.tools))
	for _, t := range g.tools {
		list = append(list, ToolInfo{
			Name:                t.Name(),
			Description:         t.Description(),
			RequiredPermissions: t.RequiredPermissions(),
		})
	}
	return list
}

// Execute dispatches a tool request through the gateway.
// It validates permissions, records the call, executes the tool, and audits the result.
func (g *Gateway) Execute(ctx context.Context, req *ToolRequest) (*ToolResult, error) {
	tool, ok := g.tools[req.ToolName]
	if !ok {
		return &ToolResult{Status: "failed", Error: fmt.Sprintf("unknown tool: %s", req.ToolName)}, nil
	}

	inputBytes, _ := json.Marshal(req.Input)
	var taskID *string
	if req.TaskID != "" {
		taskID = &req.TaskID
	}

	call := &model.ToolCall{
		TaskID:   taskID,
		AgentID:  req.AgentID,
		ToolName: req.ToolName,
		Input:    model.JSON(inputBytes),
		Status:   model.ToolCallStatusRunning,
	}
	if err := g.callSvc.Create(ctx, call); err != nil {
		return nil, fmt.Errorf("record tool call: %w", err)
	}

	if err := g.checkPermission(ctx, req.AgentID, tool); err != nil {
		g.completeCall(ctx, call, model.ToolCallStatusDenied, nil, err.Error(), 0)
		g.auditSvc.LogEvent(ctx, "agent", req.AgentID, "tool_call.denied",
			fmt.Sprintf("工具「%s」调用被拒绝: %s", req.ToolName, err.Error()),
			"tool_call", call.ID, nil, call)
		return &ToolResult{Status: "denied", Error: err.Error()}, nil
	}

	start := time.Now()
	result, execErr := tool.Execute(ctx, req.Input)
	durationMs := int(time.Since(start).Milliseconds())

	if execErr != nil {
		g.completeCall(ctx, call, model.ToolCallStatusFailed, nil, execErr.Error(), durationMs)
		g.auditSvc.LogEvent(ctx, "agent", req.AgentID, "tool_call.failed",
			fmt.Sprintf("工具「%s」执行失败: %s", req.ToolName, execErr.Error()),
			"tool_call", call.ID, nil, call)
		return &ToolResult{Status: "failed", Error: execErr.Error()}, nil
	}

	status := model.ToolCallStatusSuccess
	if result.Status == "failed" {
		status = model.ToolCallStatusFailed
	}
	g.completeCall(ctx, call, status, result.Output, result.Error, durationMs)
	g.auditSvc.LogEvent(ctx, "agent", req.AgentID, "tool_call.completed",
		fmt.Sprintf("工具「%s」执行完成（%dms）", req.ToolName, durationMs),
		"tool_call", call.ID, nil, call)

	return result, nil
}

func (g *Gateway) checkPermission(ctx context.Context, agentID string, tool Tool) error {
	agent, err := g.agentSvc.GetByID(ctx, agentID)
	if err != nil {
		return fmt.Errorf("agent lookup failed: %w", err)
	}
	if agent == nil {
		return fmt.Errorf("agent %s not found", agentID)
	}
	if agent.Status != model.AgentStatusActive {
		return fmt.Errorf("agent %s is not active", agentID)
	}

	required := tool.RequiredPermissions()
	if len(required) == 0 {
		return nil
	}

	var capabilities []string
	if len(agent.Capabilities) > 0 {
		_ = json.Unmarshal([]byte(agent.Capabilities), &capabilities)
	}

	return matchCapabilities(capabilities, required)
}

// matchCapabilities checks whether the given capabilities satisfy all required permissions.
// "tool.*" acts as a wildcard granting access to all tool permissions.
func matchCapabilities(capabilities []string, required []string) error {
	if len(required) == 0 {
		return nil
	}

	capSet := make(map[string]bool)
	for _, c := range capabilities {
		capSet[c] = true
	}
	if capSet["tool.*"] {
		return nil
	}

	for _, perm := range required {
		if !capSet[perm] {
			return fmt.Errorf("agent lacks permission: %s", perm)
		}
	}
	return nil
}

func (g *Gateway) completeCall(ctx context.Context, call *model.ToolCall, status model.ToolCallStatus, output map[string]any, errMsg string, durationMs int) {
	outputBytes, _ := json.Marshal(output)
	if output == nil {
		outputBytes = []byte("{}")
	}
	call.Status = status
	call.Output = model.JSON(outputBytes)
	call.ErrorMessage = errMsg
	call.DurationMs = durationMs

	if err := g.callSvc.Complete(ctx, call.ID, status, model.JSON(outputBytes), errMsg, durationMs); err != nil {
		g.logger.Error("update tool call failed", "id", call.ID, "error", err)
	}
}

// RegisterDefaults registers the built-in MVP tools.
// artifactTool should be created via NewArtifactStorageTool with proper config;
// when nil a mock fallback is used.
func (g *Gateway) RegisterDefaults(artifactTool *ArtifactStorageTool) {
	g.RegisterTool(&DocGeneratorTool{})
	if artifactTool != nil {
		g.RegisterTool(artifactTool)
	} else {
		g.RegisterTool(NewMockArtifactStorageTool())
	}
	g.RegisterTool(&TestRunnerTool{})
}

package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/service"
)

// Gateway is the central dispatcher for all tool calls.
// It handles permission checking, execution, recording, and auditing.
type Gateway struct {
	tools       map[string]Tool
	callSvc     *service.ToolCallService
	agentSvc    *service.AgentService
	contractSvc *service.TaskContractService
	auditSvc    *service.AuditService
	logger      *slog.Logger
}

func NewGateway(callSvc *service.ToolCallService, agentSvc *service.AgentService, contractSvc *service.TaskContractService, auditSvc *service.AuditService, logger *slog.Logger) *Gateway {
	return &Gateway{
		tools:       make(map[string]Tool),
		callSvc:     callSvc,
		agentSvc:    agentSvc,
		contractSvc: contractSvc,
		auditSvc:    auditSvc,
		logger:      logger,
	}
}

func (g *Gateway) RegisterTool(tool Tool) {
	g.tools[tool.Name()] = tool
}

func (g *Gateway) ListTools() []ToolInfo {
	list := make([]ToolInfo, 0, len(g.tools))
	for _, t := range g.tools {
		info := ToolInfo{
			Name:                t.Name(),
			Description:         t.Description(),
			RequiredPermissions: t.RequiredPermissions(),
			RiskLevel:           RiskNormal,
		}
		if at, ok := t.(ActionAwareTool); ok {
			info.Actions = at.Actions()
		}
		list = append(list, info)
	}
	return list
}

// Execute dispatches a tool request through the gateway.
// Permission checking order: (1) agent capabilities (2) task contract tool_permissions (3) risk level.
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
		Action:   req.Action,
		Input:    model.JSON(inputBytes),
		Status:   model.ToolCallStatusRunning,
	}
	if err := g.callSvc.Create(ctx, call); err != nil {
		return nil, fmt.Errorf("record tool call: %w", err)
	}

	// --- Step 1: Check agent-level capabilities ---
	if err := g.checkAgentPermission(ctx, req.AgentID, tool, req.Action); err != nil {
		reason := fmt.Sprintf("agent capability denied: %s", err.Error())
		g.denyCall(ctx, call, req.AgentID, reason)
		return &ToolResult{Status: "denied", Error: reason}, nil
	}

	// --- Step 2: Check task contract tool_permissions (when task context exists) ---
	if req.TaskID != "" {
		if err := g.checkContractPermission(ctx, req.TaskID, req.ToolName); err != nil {
			reason := fmt.Sprintf("contract denied: %s", err.Error())
			g.denyCall(ctx, call, req.AgentID, reason)
			return &ToolResult{Status: "denied", Error: reason}, nil
		}
	}

	// --- Step 3: Check risk level ---
	if at, ok := tool.(ActionAwareTool); ok {
		risk := at.RiskLevel(req.Action)
		if risk == RiskCritical {
			reason := fmt.Sprintf("tool %s action %q is critical and requires approval", req.ToolName, req.Action)
			g.escalateCall(ctx, call, req.AgentID, reason)
			return &ToolResult{Status: "escalated", Error: reason}, nil
		}
		if risk == RiskSensitive {
			if !g.hasElevatedPermission(ctx, req.AgentID, req.ToolName) {
				reason := fmt.Sprintf("tool %s action %q is sensitive; requires elevated permission", req.ToolName, req.Action)
				g.denyCall(ctx, call, req.AgentID, reason)
				return &ToolResult{Status: "denied", Error: reason}, nil
			}
		}
	}

	// --- Execute tool ---
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

// checkAgentPermission validates the agent has the required capabilities for this tool + action.
func (g *Gateway) checkAgentPermission(ctx context.Context, agentID string, tool Tool, action string) error {
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
	if len(required) == 0 && action == "" {
		return nil
	}

	var capabilities []string
	if len(agent.Capabilities) > 0 {
		_ = json.Unmarshal([]byte(agent.Capabilities), &capabilities)
	}

	if err := matchCapabilities(capabilities, required); err != nil {
		return err
	}

	// Action-level check: if action is specified and tool supports actions, check tool.<name>:<action>
	if action != "" {
		if _, ok := tool.(ActionAwareTool); ok {
			actionPerm := fmt.Sprintf("tool.%s:%s", tool.Name(), action)
			if err := matchActionPermission(capabilities, tool.Name(), actionPerm); err != nil {
				return err
			}
		}
	}

	return nil
}

// checkContractPermission validates the tool is allowed by the task contract.
func (g *Gateway) checkContractPermission(ctx context.Context, taskID string, toolName string) error {
	contract, err := g.contractSvc.GetLatestByTaskID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("contract lookup failed: %w", err)
	}
	if contract == nil {
		// No contract = no restriction
		return nil
	}

	var allowedTools []string
	if len(contract.ToolPermissions) > 0 {
		_ = json.Unmarshal([]byte(contract.ToolPermissions), &allowedTools)
	}
	if len(allowedTools) == 0 {
		// Empty permission list = no restriction
		return nil
	}

	toolPerm := "tool." + toolName
	for _, allowed := range allowedTools {
		if allowed == "tool.*" || allowed == toolPerm {
			return nil
		}
		// support prefix wildcard: "tool.artifact_storage:*" allows all actions
		if strings.HasPrefix(allowed, toolPerm+":") || strings.HasPrefix(allowed, toolPerm) {
			return nil
		}
	}
	return fmt.Errorf("tool %q not permitted by task contract", toolName)
}

// hasElevatedPermission checks if the agent has explicit elevated access to a sensitive tool.
func (g *Gateway) hasElevatedPermission(ctx context.Context, agentID string, toolName string) bool {
	agent, err := g.agentSvc.GetByID(ctx, agentID)
	if err != nil || agent == nil {
		return false
	}
	var capabilities []string
	if len(agent.Capabilities) > 0 {
		_ = json.Unmarshal([]byte(agent.Capabilities), &capabilities)
	}
	elevated := fmt.Sprintf("tool.%s.elevated", toolName)
	for _, c := range capabilities {
		if c == elevated || c == "tool.*" {
			return true
		}
	}
	return false
}

// denyCall records a denied tool call with reason and audit event.
func (g *Gateway) denyCall(ctx context.Context, call *model.ToolCall, agentID, reason string) {
	call.Status = model.ToolCallStatusDenied
	call.DeniedReason = reason
	if err := g.callSvc.UpdateDenied(ctx, call.ID, reason); err != nil {
		g.logger.Error("update denied tool call failed", "id", call.ID, "error", err)
	}
	g.auditSvc.LogEvent(ctx, "agent", agentID, "tool_call.denied",
		fmt.Sprintf("工具「%s」调用被拒绝: %s", call.ToolName, reason),
		"tool_call", call.ID, nil, call)
}

// escalateCall records an escalated tool call (critical risk) with audit event.
func (g *Gateway) escalateCall(ctx context.Context, call *model.ToolCall, agentID, reason string) {
	call.Status = model.ToolCallStatusEscalated
	call.DeniedReason = reason
	outputBytes, _ := json.Marshal(map[string]string{"escalation_reason": reason})
	if err := g.callSvc.Complete(ctx, call.ID, model.ToolCallStatusEscalated, model.JSON(outputBytes), reason, 0); err != nil {
		g.logger.Error("update escalated tool call failed", "id", call.ID, "error", err)
	}
	g.auditSvc.LogEvent(ctx, "agent", agentID, "tool_call.escalated",
		fmt.Sprintf("工具「%s」调用需要审批: %s", call.ToolName, reason),
		"tool_call", call.ID, nil, call)
}

// matchCapabilities checks whether the given capabilities satisfy all required permissions.
// Supports wildcards: "tool.*" grants all tool permissions.
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

// matchActionPermission checks if the agent has permission for a specific tool action.
// Permission hierarchy:
//   - "tool.*" grants everything
//   - "tool.<name>" grants all actions on that tool
//   - "tool.<name>:*" grants all actions on that tool
//   - "tool.<name>:<action>" grants specific action
func matchActionPermission(capabilities []string, toolName string, actionPerm string) error {
	for _, c := range capabilities {
		if c == "tool.*" {
			return nil
		}
		if c == "tool."+toolName {
			return nil
		}
		if c == "tool."+toolName+":*" {
			return nil
		}
		if c == actionPerm {
			return nil
		}
	}
	return fmt.Errorf("agent lacks action permission: %s", actionPerm)
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
// when nil, artifact storage calls fail explicitly instead of returning mock URIs.
func (g *Gateway) RegisterDefaults(artifactTool *ArtifactStorageTool) {
	g.RegisterTool(&DocGeneratorTool{})
	if artifactTool != nil {
		g.RegisterTool(artifactTool)
	} else {
		g.RegisterTool(NewMockArtifactStorageTool())
	}
	g.RegisterTool(&TestRunnerTool{})
}

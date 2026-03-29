package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/lingchou/lingchoubot/backend/internal/gateway"
	"github.com/lingchou/lingchoubot/backend/internal/middleware"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
	"github.com/lingchou/lingchoubot/backend/internal/service"
)

type ToolCallHandler struct {
	svc     *service.ToolCallService
	gateway *gateway.Gateway
}

func NewToolCallHandler(svc *service.ToolCallService, gw *gateway.Gateway) *ToolCallHandler {
	return &ToolCallHandler{svc: svc, gateway: gw}
}

func (h *ToolCallHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/tools/call", h.CallTool)
	mux.HandleFunc("GET /api/v1/tools", h.ListTools)
	mux.HandleFunc("GET /api/v1/tool-calls", h.ListCalls)
	mux.HandleFunc("GET /api/v1/tool-calls/{id}", h.GetCall)
}

// CallTool dispatches a tool call through the gateway.
func (h *ToolCallHandler) CallTool(w http.ResponseWriter, r *http.Request) {
	var req gateway.ToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.ToolName == "" {
		middleware.ErrorJSON(w, http.StatusBadRequest, "MISSING_FIELD", "tool_name is required")
		return
	}
	if req.AgentID == "" {
		middleware.ErrorJSON(w, http.StatusBadRequest, "MISSING_FIELD", "agent_id is required")
		return
	}

	result, err := h.gateway.Execute(r.Context(), &req)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "GATEWAY_ERROR", err.Error())
		return
	}

	middleware.JSON(w, http.StatusOK, result)
}

// ListTools returns available tools registered in the gateway.
func (h *ToolCallHandler) ListTools(w http.ResponseWriter, r *http.Request) {
	tools := h.gateway.ListTools()
	middleware.JSON(w, http.StatusOK, map[string]interface{}{
		"items": tools,
		"total": len(tools),
	})
}

// ListCalls returns tool call records with optional filters.
func (h *ToolCallHandler) ListCalls(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	params := repository.ToolCallListParams{
		TaskID:   q.Get("task_id"),
		AgentID:  q.Get("agent_id"),
		ToolName: q.Get("tool_name"),
		Status:   q.Get("status"),
		Limit:    limit,
		Offset:   offset,
	}

	items, total, err := h.svc.List(r.Context(), params)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "LIST_ERROR", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"total": total,
	})
}

// GetCall returns a single tool call by ID.
func (h *ToolCallHandler) GetCall(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tc, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "GET_ERROR", err.Error())
		return
	}
	if tc == nil {
		middleware.ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "tool call not found")
		return
	}
	middleware.JSON(w, http.StatusOK, tc)
}

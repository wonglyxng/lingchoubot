package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/lingchou/lingchoubot/backend/internal/middleware"
	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/service"
)

type AgentHandler struct {
	svc *service.AgentService
}

func NewAgentHandler(svc *service.AgentService) *AgentHandler {
	return &AgentHandler{svc: svc}
}

func (h *AgentHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/agents", h.Create)
	mux.HandleFunc("GET /api/v1/agents", h.List)
	mux.HandleFunc("GET /api/v1/agents/org-tree", h.OrgTree)
	mux.HandleFunc("GET /api/v1/agents/{id}", h.Get)
	mux.HandleFunc("GET /api/v1/agents/{id}/subordinates", h.Subordinates)
	mux.HandleFunc("PUT /api/v1/agents/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/agents/{id}", h.Delete)
}

func (h *AgentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var a model.Agent
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "INVALID_BODY", "请求体解析失败")
		return
	}
	if err := h.svc.Create(r.Context(), &a); err != nil {
		if errors.Is(err, service.ErrAgentRoleCodeConflict) {
			middleware.ErrorJSON(w, http.StatusConflict, "ROLE_CODE_CONFLICT", err.Error())
			return
		}
		middleware.ErrorJSON(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusCreated, a)
}

func (h *AgentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	if a == nil {
		middleware.ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "Agent 不存在")
		return
	}
	middleware.JSON(w, http.StatusOK, a)
}

func (h *AgentHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	list, total, err := h.svc.List(r.Context(), limit, offset)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, map[string]interface{}{
		"items": list,
		"total": total,
	})
}

func (h *AgentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var a model.Agent
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "INVALID_BODY", "请求体解析失败")
		return
	}
	a.ID = id
	if err := h.svc.Update(r.Context(), &a); err != nil {
		if errors.Is(err, service.ErrAgentRoleCodeConflict) {
			middleware.ErrorJSON(w, http.StatusConflict, "ROLE_CODE_CONFLICT", err.Error())
			return
		}
		middleware.ErrorJSON(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, a)
}

func (h *AgentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "DELETE_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, map[string]string{"deleted": id})
}

func (h *AgentHandler) Subordinates(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	list, err := h.svc.GetSubordinates(r.Context(), id)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, map[string]interface{}{
		"items": list,
		"total": len(list),
	})
}

// OrgTree returns a flat list of agents ordered by depth.
// Optional query param: root_id (defaults to full tree).
func (h *AgentHandler) OrgTree(w http.ResponseWriter, r *http.Request) {
	rootID := r.URL.Query().Get("root_id")
	list, err := h.svc.GetOrgTree(r.Context(), rootID)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, map[string]interface{}{
		"items": list,
		"total": len(list),
	})
}

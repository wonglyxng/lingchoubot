package handler

import (
	"encoding/json"
	"net/http"

	"github.com/lingchou/lingchoubot/backend/internal/middleware"
	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/service"
)

type PhaseHandler struct {
	svc *service.PhaseService
}

func NewPhaseHandler(svc *service.PhaseService) *PhaseHandler {
	return &PhaseHandler{svc: svc}
}

func (h *PhaseHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/phases", h.Create)
	mux.HandleFunc("GET /api/v1/projects/{projectId}/phases", h.ListByProject)
	mux.HandleFunc("GET /api/v1/phases/{id}", h.Get)
	mux.HandleFunc("PUT /api/v1/phases/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/phases/{id}", h.Delete)
}

func (h *PhaseHandler) Create(w http.ResponseWriter, r *http.Request) {
	var p model.Phase
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "INVALID_BODY", "请求体解析失败")
		return
	}
	if err := h.svc.Create(r.Context(), &p); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusCreated, p)
}

func (h *PhaseHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	if p == nil {
		middleware.ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "阶段不存在")
		return
	}
	middleware.JSON(w, http.StatusOK, p)
}

func (h *PhaseHandler) ListByProject(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectId")
	list, err := h.svc.ListByProject(r.Context(), projectID)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, map[string]interface{}{
		"items": list,
		"total": len(list),
	})
}

func (h *PhaseHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var p model.Phase
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "INVALID_BODY", "请求体解析失败")
		return
	}
	p.ID = id
	if err := h.svc.Update(r.Context(), &p); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, p)
}

func (h *PhaseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "DELETE_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, map[string]string{"deleted": id})
}

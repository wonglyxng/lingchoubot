package handler

import (
	"net/http"

	"github.com/lingchou/lingchoubot/backend/internal/middleware"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
	"github.com/lingchou/lingchoubot/backend/internal/service"
)

type AuditHandler struct {
	svc *service.AuditService
}

func NewAuditHandler(svc *service.AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

func (h *AuditHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/audit-logs", h.List)
	mux.HandleFunc("GET /api/v1/projects/{projectId}/timeline", h.ProjectTimeline)
	mux.HandleFunc("GET /api/v1/tasks/{taskId}/timeline", h.TaskTimeline)
}

func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	p := repository.AuditListParams{
		TargetType: r.URL.Query().Get("target_type"),
		TargetID:   r.URL.Query().Get("target_id"),
		EventType:  r.URL.Query().Get("event_type"),
		Limit:      limit,
		Offset:     offset,
	}
	list, total, err := h.svc.List(r.Context(), p)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, map[string]interface{}{
		"items": list,
		"total": total,
	})
}

func (h *AuditHandler) ProjectTimeline(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectId")
	limit, offset := parsePagination(r)
	list, total, err := h.svc.ProjectTimeline(r.Context(), projectID, limit, offset)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, map[string]interface{}{
		"items": list,
		"total": total,
	})
}

func (h *AuditHandler) TaskTimeline(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	limit, offset := parsePagination(r)
	list, total, err := h.svc.TaskTimeline(r.Context(), taskID, limit, offset)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, map[string]interface{}{
		"items": list,
		"total": total,
	})
}

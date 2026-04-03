package handler

import (
	"encoding/json"
	"net/http"

	"github.com/lingchou/lingchoubot/backend/internal/middleware"
	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
	"github.com/lingchou/lingchoubot/backend/internal/service"
)

type ApprovalRequestHandler struct {
	svc *service.ApprovalRequestService
}

func NewApprovalRequestHandler(svc *service.ApprovalRequestService) *ApprovalRequestHandler {
	return &ApprovalRequestHandler{svc: svc}
}

func (h *ApprovalRequestHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/approvals", h.Create)
	mux.HandleFunc("GET /api/v1/approvals", h.List)
	mux.HandleFunc("GET /api/v1/approvals/{id}", h.Get)
	mux.HandleFunc("POST /api/v1/approvals/{id}/decide", h.Decide)
}

func (h *ApprovalRequestHandler) Create(w http.ResponseWriter, r *http.Request) {
	var a model.ApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "INVALID_BODY", "请求体解析失败")
		return
	}
	if err := h.svc.Create(r.Context(), &a); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusCreated, a)
}

func (h *ApprovalRequestHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	if a == nil {
		middleware.ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "审批请求不存在")
		return
	}
	middleware.JSON(w, http.StatusOK, a)
}

func (h *ApprovalRequestHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	p := repository.ApprovalListParams{
		ProjectID: r.URL.Query().Get("project_id"),
		TaskID:    r.URL.Query().Get("task_id"),
		Status:    r.URL.Query().Get("status"),
		Limit:     limit,
		Offset:    offset,
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

func (h *ApprovalRequestHandler) Decide(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Status       model.ApprovalStatus `json:"status"`
		Note         string               `json:"note"`
		DecisionNote string               `json:"decision_note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "INVALID_BODY", "请求体解析失败")
		return
	}
	note := body.Note
	if note == "" {
		note = body.DecisionNote
	}
	result, err := h.svc.Decide(r.Context(), id, body.Status, note)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "DECIDE_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, result)
}

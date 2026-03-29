package handler

import (
	"encoding/json"
	"net/http"

	"github.com/lingchou/lingchoubot/backend/internal/middleware"
	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
	"github.com/lingchou/lingchoubot/backend/internal/service"
)

type ReviewReportHandler struct {
	svc *service.ReviewReportService
}

func NewReviewReportHandler(svc *service.ReviewReportService) *ReviewReportHandler {
	return &ReviewReportHandler{svc: svc}
}

func (h *ReviewReportHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/reviews", h.Create)
	mux.HandleFunc("GET /api/v1/reviews", h.List)
	mux.HandleFunc("GET /api/v1/reviews/{id}", h.Get)
}

func (h *ReviewReportHandler) Create(w http.ResponseWriter, r *http.Request) {
	var rr model.ReviewReport
	if err := json.NewDecoder(r.Body).Decode(&rr); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "INVALID_BODY", "请求体解析失败")
		return
	}
	if err := h.svc.Create(r.Context(), &rr); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusCreated, rr)
}

func (h *ReviewReportHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rr, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	if rr == nil {
		middleware.ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "评审报告不存在")
		return
	}
	middleware.JSON(w, http.StatusOK, rr)
}

func (h *ReviewReportHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	p := repository.ReviewListParams{
		TaskID:     r.URL.Query().Get("task_id"),
		ReviewerID: r.URL.Query().Get("reviewer_id"),
		Verdict:    r.URL.Query().Get("verdict"),
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

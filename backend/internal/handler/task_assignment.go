package handler

import (
	"encoding/json"
	"net/http"

	"github.com/lingchou/lingchoubot/backend/internal/middleware"
	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
	"github.com/lingchou/lingchoubot/backend/internal/service"
)

type TaskAssignmentHandler struct {
	svc *service.TaskAssignmentService
}

func NewTaskAssignmentHandler(svc *service.TaskAssignmentService) *TaskAssignmentHandler {
	return &TaskAssignmentHandler{svc: svc}
}

func (h *TaskAssignmentHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/task-assignments", h.Create)
	mux.HandleFunc("GET /api/v1/task-assignments", h.List)
	mux.HandleFunc("GET /api/v1/task-assignments/{id}", h.Get)
	mux.HandleFunc("PATCH /api/v1/task-assignments/{id}/status", h.UpdateStatus)
}

func (h *TaskAssignmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var a model.TaskAssignment
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

func (h *TaskAssignmentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	if a == nil {
		middleware.ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "分派记录不存在")
		return
	}
	middleware.JSON(w, http.StatusOK, a)
}

func (h *TaskAssignmentHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	p := repository.AssignmentListParams{
		TaskID:  r.URL.Query().Get("task_id"),
		AgentID: r.URL.Query().Get("agent_id"),
		Status:  r.URL.Query().Get("status"),
		Limit:   limit,
		Offset:  offset,
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

func (h *TaskAssignmentHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Status model.AssignmentStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "INVALID_BODY", "请求体解析失败")
		return
	}
	if err := h.svc.UpdateStatus(r.Context(), id, body.Status); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, map[string]string{"id": id, "status": string(body.Status)})
}

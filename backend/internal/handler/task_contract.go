package handler

import (
	"encoding/json"
	"net/http"

	"github.com/lingchou/lingchoubot/backend/internal/middleware"
	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/service"
)

type TaskContractHandler struct {
	svc *service.TaskContractService
}

func NewTaskContractHandler(svc *service.TaskContractService) *TaskContractHandler {
	return &TaskContractHandler{svc: svc}
}

func (h *TaskContractHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/task-contracts", h.Create)
	mux.HandleFunc("GET /api/v1/task-contracts/{id}", h.Get)
	mux.HandleFunc("PUT /api/v1/task-contracts/{id}", h.Update)
	mux.HandleFunc("GET /api/v1/tasks/{taskId}/contracts", h.ListByTask)
	mux.HandleFunc("GET /api/v1/tasks/{taskId}/contracts/latest", h.GetLatestByTask)
}

func (h *TaskContractHandler) Create(w http.ResponseWriter, r *http.Request) {
	var c model.TaskContract
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "INVALID_BODY", "请求体解析失败")
		return
	}
	if err := h.svc.Create(r.Context(), &c); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusCreated, c)
}

func (h *TaskContractHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	if c == nil {
		middleware.ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "任务契约不存在")
		return
	}
	middleware.JSON(w, http.StatusOK, c)
}

func (h *TaskContractHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var c model.TaskContract
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "INVALID_BODY", "请求体解析失败")
		return
	}
	c.ID = id
	if err := h.svc.Update(r.Context(), &c); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, c)
}

func (h *TaskContractHandler) ListByTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	list, err := h.svc.ListByTaskID(r.Context(), taskID)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, map[string]interface{}{
		"items": list,
		"total": len(list),
	})
}

func (h *TaskContractHandler) GetLatestByTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	c, err := h.svc.GetLatestByTaskID(r.Context(), taskID)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	if c == nil {
		middleware.ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "该任务暂无契约")
		return
	}
	middleware.JSON(w, http.StatusOK, c)
}

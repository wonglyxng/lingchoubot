package handler

import (
	"encoding/json"
	"net/http"

	"github.com/lingchou/lingchoubot/backend/internal/middleware"
	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
	"github.com/lingchou/lingchoubot/backend/internal/service"
)

type HandoffSnapshotHandler struct {
	svc *service.HandoffSnapshotService
}

func NewHandoffSnapshotHandler(svc *service.HandoffSnapshotService) *HandoffSnapshotHandler {
	return &HandoffSnapshotHandler{svc: svc}
}

func (h *HandoffSnapshotHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/handoff-snapshots", h.Create)
	mux.HandleFunc("GET /api/v1/handoff-snapshots", h.List)
	mux.HandleFunc("GET /api/v1/handoff-snapshots/{id}", h.Get)
	mux.HandleFunc("GET /api/v1/tasks/{taskId}/handoff-snapshots/latest", h.GetLatestByTask)
}

func (h *HandoffSnapshotHandler) Create(w http.ResponseWriter, r *http.Request) {
	var snap model.HandoffSnapshot
	if err := json.NewDecoder(r.Body).Decode(&snap); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "INVALID_BODY", "请求体解析失败")
		return
	}
	if err := h.svc.Create(r.Context(), &snap); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusCreated, snap)
}

func (h *HandoffSnapshotHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	snap, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	if snap == nil {
		middleware.ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "交接快照不存在")
		return
	}
	middleware.JSON(w, http.StatusOK, snap)
}

func (h *HandoffSnapshotHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	p := repository.HandoffListParams{
		TaskID:  r.URL.Query().Get("task_id"),
		AgentID: r.URL.Query().Get("agent_id"),
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

func (h *HandoffSnapshotHandler) GetLatestByTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	snap, err := h.svc.GetLatestByTaskID(r.Context(), taskID)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	if snap == nil {
		middleware.ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "该任务暂无交接快照")
		return
	}
	middleware.JSON(w, http.StatusOK, snap)
}

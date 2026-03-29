package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/lingchou/lingchoubot/backend/internal/middleware"
	"github.com/lingchou/lingchoubot/backend/internal/orchestrator"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

type OrchestratorHandler struct {
	engine *orchestrator.Engine
}

func NewOrchestratorHandler(engine *orchestrator.Engine) *OrchestratorHandler {
	return &OrchestratorHandler{engine: engine}
}

func (h *OrchestratorHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/orchestrator/runs", h.StartRun)
	mux.HandleFunc("GET /api/v1/orchestrator/runs", h.ListRuns)
	mux.HandleFunc("GET /api/v1/orchestrator/runs/{id}", h.GetRun)
}

// StartRun triggers a workflow run for a given project.
func (h *OrchestratorHandler) StartRun(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if body.ProjectID == "" {
		middleware.ErrorJSON(w, http.StatusBadRequest, "MISSING_FIELD", "project_id is required")
		return
	}

	run, err := h.engine.Run(r.Context(), body.ProjectID)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "WORKFLOW_ERROR", err.Error())
		return
	}

	middleware.JSON(w, http.StatusCreated, run)
}

// ListRuns returns paginated workflow runs from the database.
func (h *OrchestratorHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 {
		limit = 20
	}

	params := repository.WorkflowRunListParams{
		ProjectID: q.Get("project_id"),
		Status:    q.Get("status"),
		Limit:     limit,
		Offset:    offset,
	}

	runs, total, err := h.engine.ListRuns(r.Context(), params)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "LIST_ERROR", err.Error())
		return
	}

	middleware.JSON(w, http.StatusOK, map[string]interface{}{
		"items": runs,
		"total": total,
	})
}

// GetRun returns a single workflow run by ID with its steps.
func (h *OrchestratorHandler) GetRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	run, err := h.engine.GetRun(r.Context(), id)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "GET_ERROR", err.Error())
		return
	}
	if run == nil {
		middleware.ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "workflow run not found")
		return
	}
	middleware.JSON(w, http.StatusOK, run)
}

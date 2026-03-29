package handler

import (
	"encoding/json"
	"net/http"

	"github.com/lingchou/lingchoubot/backend/internal/middleware"
	"github.com/lingchou/lingchoubot/backend/internal/orchestrator"
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

// ListRuns returns all workflow runs (in-memory).
func (h *OrchestratorHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	runs := h.engine.Store().List()
	middleware.JSON(w, http.StatusOK, map[string]interface{}{
		"items": runs,
		"total": len(runs),
	})
}

// GetRun returns a single workflow run by ID.
func (h *OrchestratorHandler) GetRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	run := h.engine.Store().Get(id)
	if run == nil {
		middleware.ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "workflow run not found")
		return
	}
	middleware.JSON(w, http.StatusOK, run)
}

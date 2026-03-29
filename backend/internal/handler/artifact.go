package handler

import (
	"encoding/json"
	"net/http"

	"github.com/lingchou/lingchoubot/backend/internal/middleware"
	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
	"github.com/lingchou/lingchoubot/backend/internal/service"
)

type ArtifactHandler struct {
	svc *service.ArtifactService
}

func NewArtifactHandler(svc *service.ArtifactService) *ArtifactHandler {
	return &ArtifactHandler{svc: svc}
}

func (h *ArtifactHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/artifacts", h.Create)
	mux.HandleFunc("GET /api/v1/artifacts", h.List)
	mux.HandleFunc("GET /api/v1/artifacts/{id}", h.Get)
	mux.HandleFunc("POST /api/v1/artifacts/{id}/versions", h.AddVersion)
	mux.HandleFunc("GET /api/v1/artifacts/{id}/versions", h.ListVersions)
}

func (h *ArtifactHandler) Create(w http.ResponseWriter, r *http.Request) {
	var a model.Artifact
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

func (h *ArtifactHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	if a == nil {
		middleware.ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "工件不存在")
		return
	}
	middleware.JSON(w, http.StatusOK, a)
}

func (h *ArtifactHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	p := repository.ArtifactListParams{
		ProjectID:    r.URL.Query().Get("project_id"),
		TaskID:       r.URL.Query().Get("task_id"),
		ArtifactType: r.URL.Query().Get("artifact_type"),
		Limit:        limit,
		Offset:       offset,
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

func (h *ArtifactHandler) AddVersion(w http.ResponseWriter, r *http.Request) {
	artifactID := r.PathValue("id")
	var v model.ArtifactVersion
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "INVALID_BODY", "请求体解析失败")
		return
	}
	v.ArtifactID = artifactID
	if err := h.svc.AddVersion(r.Context(), &v); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusCreated, v)
}

func (h *ArtifactHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	artifactID := r.PathValue("id")
	list, err := h.svc.ListVersions(r.Context(), artifactID)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, map[string]interface{}{
		"items": list,
		"total": len(list),
	})
}

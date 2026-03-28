package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/lingchou/lingchoubot/backend/internal/middleware"
	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/service"
)

type ProjectHandler struct {
	svc *service.ProjectService
}

func NewProjectHandler(svc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

func (h *ProjectHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/projects", h.Create)
	mux.HandleFunc("GET /api/v1/projects", h.List)
	mux.HandleFunc("GET /api/v1/projects/{id}", h.Get)
	mux.HandleFunc("PUT /api/v1/projects/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/projects/{id}", h.Delete)
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var p model.Project
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

func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	if p == nil {
		middleware.ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "项目不存在")
		return
	}
	middleware.JSON(w, http.StatusOK, p)
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	list, total, err := h.svc.List(r.Context(), limit, offset)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, map[string]interface{}{
		"items": list,
		"total": total,
	})
}

func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var p model.Project
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

func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "DELETE_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, map[string]string{"deleted": id})
}

func parsePagination(r *http.Request) (int, int) {
	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}

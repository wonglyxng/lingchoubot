package handler

import (
	"encoding/json"
	"net/http"

	"github.com/lingchou/lingchoubot/backend/internal/middleware"
	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/service"
)

type LLMProviderHandler struct {
	svc *service.LLMProviderService
}

func NewLLMProviderHandler(svc *service.LLMProviderService) *LLMProviderHandler {
	return &LLMProviderHandler{svc: svc}
}

func (h *LLMProviderHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/llm-providers", h.CreateProvider)
	mux.HandleFunc("GET /api/v1/llm-providers", h.ListProviders)
	mux.HandleFunc("GET /api/v1/llm-providers/{id}", h.GetProvider)
	mux.HandleFunc("PUT /api/v1/llm-providers/{id}", h.UpdateProvider)
	mux.HandleFunc("DELETE /api/v1/llm-providers/{id}", h.DeleteProvider)
	mux.HandleFunc("POST /api/v1/llm-providers/{id}/models", h.CreateModel)
	mux.HandleFunc("GET /api/v1/llm-providers/{id}/models", h.ListModels)
	mux.HandleFunc("PUT /api/v1/llm-models/{id}", h.UpdateModel)
	mux.HandleFunc("DELETE /api/v1/llm-models/{id}", h.DeleteModel)
}

// --- Provider endpoints ---

func (h *LLMProviderHandler) CreateProvider(w http.ResponseWriter, r *http.Request) {
	var p model.LLMProvider
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "INVALID_BODY", "请求体解析失败")
		return
	}
	if err := h.svc.Create(r.Context(), &p); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	service.MaskProviderAPIKey(&p)
	middleware.JSON(w, http.StatusCreated, p)
}

func (h *LLMProviderHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	enabledOnly := r.URL.Query().Get("enabled_only") == "true"
	providers, err := h.svc.ListWithModels(r.Context(), enabledOnly)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	service.MaskProvidersAPIKey(providers)
	middleware.JSON(w, http.StatusOK, map[string]interface{}{
		"items": providers,
	})
}

func (h *LLMProviderHandler) GetProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	if p == nil {
		middleware.ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "供应商不存在")
		return
	}
	// populate models
	models, _ := h.svc.ListModelsByProvider(r.Context(), id)
	p.Models = models
	service.MaskProviderAPIKey(p)
	middleware.JSON(w, http.StatusOK, p)
}

func (h *LLMProviderHandler) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var p model.LLMProvider
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "INVALID_BODY", "请求体解析失败")
		return
	}
	p.ID = id
	if err := h.svc.Update(r.Context(), &p); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	service.MaskProviderAPIKey(&p)
	middleware.JSON(w, http.StatusOK, p)
}

func (h *LLMProviderHandler) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, nil)
}

// --- Model endpoints ---

func (h *LLMProviderHandler) CreateModel(w http.ResponseWriter, r *http.Request) {
	providerID := r.PathValue("id")
	var m model.LLMModel
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "INVALID_BODY", "请求体解析失败")
		return
	}
	m.ProviderID = providerID
	if err := h.svc.CreateModel(r.Context(), &m); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusCreated, m)
}

func (h *LLMProviderHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	providerID := r.PathValue("id")
	models, err := h.svc.ListModelsByProvider(r.Context(), providerID)
	if err != nil {
		middleware.ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, map[string]interface{}{
		"items": models,
	})
}

func (h *LLMProviderHandler) UpdateModel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var m model.LLMModel
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "INVALID_BODY", "请求体解析失败")
		return
	}
	m.ID = id
	if err := h.svc.UpdateModel(r.Context(), &m); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, m)
}

func (h *LLMProviderHandler) DeleteModel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.DeleteModel(r.Context(), id); err != nil {
		middleware.ErrorJSON(w, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	middleware.JSON(w, http.StatusOK, nil)
}

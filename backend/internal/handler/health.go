package handler

import (
	"database/sql"
	"net/http"

	"github.com/lingchou/lingchoubot/backend/internal/middleware"
)

type HealthHandler struct {
	db *sql.DB
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	middleware.JSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Ping(); err != nil {
		middleware.ErrorJSON(w, http.StatusServiceUnavailable, "DB_UNAVAILABLE", "database is not reachable")
		return
	}
	middleware.JSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}

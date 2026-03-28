package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lingchou/lingchoubot/backend/internal/config"
	"github.com/lingchou/lingchoubot/backend/internal/handler"
	"github.com/lingchou/lingchoubot/backend/internal/middleware"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
	"github.com/lingchou/lingchoubot/backend/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()

	db, err := repository.NewDB(cfg.Database.DSN(), logger)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// --- repositories ---
	projectRepo := repository.NewProjectRepo(db)
	phaseRepo := repository.NewPhaseRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	taskRepo := repository.NewTaskRepo(db)
	auditRepo := repository.NewAuditRepo(db)

	// --- services ---
	auditSvc := service.NewAuditService(auditRepo, logger)
	projectSvc := service.NewProjectService(projectRepo, auditSvc)
	phaseSvc := service.NewPhaseService(phaseRepo, projectSvc, auditSvc)
	agentSvc := service.NewAgentService(agentRepo, auditSvc)
	taskSvc := service.NewTaskService(taskRepo, auditSvc)

	// --- handlers ---
	mux := http.NewServeMux()

	healthH := handler.NewHealthHandler(db)
	mux.HandleFunc("GET /healthz", healthH.Healthz)
	mux.HandleFunc("GET /readyz", healthH.Readyz)

	mux.HandleFunc("GET /api/v1/ping", func(w http.ResponseWriter, r *http.Request) {
		middleware.JSON(w, http.StatusOK, map[string]string{
			"message": "pong",
			"version": "0.1.0",
		})
	})

	handler.NewProjectHandler(projectSvc).Register(mux)
	handler.NewPhaseHandler(phaseSvc).Register(mux)
	handler.NewAgentHandler(agentSvc).Register(mux)
	handler.NewTaskHandler(taskSvc).Register(mux)
	handler.NewAuditHandler(auditSvc).Register(mux)

	// --- middleware chain ---
	var chain http.Handler = mux
	chain = middleware.CORS(chain)
	chain = middleware.Logging(logger)(chain)
	chain = middleware.Recovery(logger)(chain)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      chain,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}
	logger.Info("server stopped")
}

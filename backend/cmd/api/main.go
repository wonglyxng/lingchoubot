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
	"github.com/lingchou/lingchoubot/backend/internal/gateway"
	"github.com/lingchou/lingchoubot/backend/internal/handler"
	"github.com/lingchou/lingchoubot/backend/internal/middleware"
	"github.com/lingchou/lingchoubot/backend/internal/orchestrator"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
	"github.com/lingchou/lingchoubot/backend/internal/runtime"
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
	contractRepo := repository.NewTaskContractRepo(db)
	assignmentRepo := repository.NewTaskAssignmentRepo(db)
	handoffRepo := repository.NewHandoffSnapshotRepo(db)
	artifactRepo := repository.NewArtifactRepo(db)
	artifactVerRepo := repository.NewArtifactVersionRepo(db)
	reviewRepo := repository.NewReviewReportRepo(db)
	approvalRepo := repository.NewApprovalRequestRepo(db)
	toolCallRepo := repository.NewToolCallRepo(db)

	// --- services ---
	auditSvc := service.NewAuditService(auditRepo, logger)
	projectSvc := service.NewProjectService(projectRepo, auditSvc)
	phaseSvc := service.NewPhaseService(phaseRepo, projectSvc, auditSvc)
	agentSvc := service.NewAgentService(agentRepo, auditSvc)
	taskSvc := service.NewTaskService(taskRepo, auditSvc)
	contractSvc := service.NewTaskContractService(contractRepo, auditSvc)
	assignmentSvc := service.NewTaskAssignmentService(assignmentRepo, auditSvc)
	handoffSvc := service.NewHandoffSnapshotService(handoffRepo, auditSvc)
	artifactSvc := service.NewArtifactService(artifactRepo, artifactVerRepo, auditSvc)
	reviewSvc := service.NewReviewReportService(reviewRepo, taskSvc, auditSvc)
	approvalSvc := service.NewApprovalRequestService(approvalRepo, taskSvc, auditSvc)
	toolCallSvc := service.NewToolCallService(toolCallRepo, auditSvc)

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
	handler.NewTaskContractHandler(contractSvc).Register(mux)
	handler.NewTaskAssignmentHandler(assignmentSvc).Register(mux)
	handler.NewHandoffSnapshotHandler(handoffSvc).Register(mux)
	handler.NewArtifactHandler(artifactSvc).Register(mux)
	handler.NewReviewReportHandler(reviewSvc).Register(mux)
	handler.NewApprovalRequestHandler(approvalSvc).Register(mux)
	handler.NewAuditHandler(auditSvc).Register(mux)

	// --- tool gateway ---
	gw := gateway.NewGateway(toolCallSvc, agentSvc, contractSvc, auditSvc, logger)
	artifactTool := gateway.NewArtifactStorageTool(cfg.MinIO, logger)
	gw.RegisterDefaults(artifactTool)
	handler.NewToolCallHandler(toolCallSvc, gw).Register(mux)

	// --- agent runtime & orchestrator ---
	reg := runtime.NewRegistry()

	if cfg.LLM.Enabled && cfg.LLM.APIKey != "" {
		defaultClient := runtime.NewLLMClient(runtime.LLMClientConfig{
			BaseURL: cfg.LLM.BaseURL,
			APIKey:  cfg.LLM.APIKey,
			Model:   cfg.LLM.Model,
		})

		// 为有独立配置的角色创建专属 LLM 客户端
		roleClients := make(map[string]*runtime.LLMClient)
		for _, role := range []string{"pm", "supervisor", "worker", "reviewer"} {
			baseURL, apiKey, model := cfg.LLM.ResolveForRole(role)
			if baseURL != cfg.LLM.BaseURL || apiKey != cfg.LLM.APIKey || model != cfg.LLM.Model {
				roleClients[role] = runtime.NewLLMClient(runtime.LLMClientConfig{
					BaseURL: baseURL,
					APIKey:  apiKey,
					Model:   model,
				})
				logger.Info("role-specific LLM configured", "role", role, "model", model, "base_url", baseURL)
			}
		}

		runtime.RegisterLLMRunners(reg, defaultClient, roleClients, logger)
		logger.Info("LLM agent runners registered", "model", cfg.LLM.Model, "base_url", cfg.LLM.BaseURL)
	} else {
		reg.RegisterDefaults()
		logger.Info("mock agent runners registered (set LLM_ENABLED=true to use LLM)")
	}

	workflowRunRepo := repository.NewWorkflowRunRepo(db)
	workflowStepRepo := repository.NewWorkflowStepRepo(db)
	workflowSvc := service.NewWorkflowService(workflowRunRepo, workflowStepRepo, auditSvc)

	orchServices := &orchestrator.Services{
		Project:    projectSvc,
		Phase:      phaseSvc,
		Agent:      agentSvc,
		Task:       taskSvc,
		Contract:   contractSvc,
		Assignment: assignmentSvc,
		Artifact:   artifactSvc,
		Handoff:    handoffSvc,
		Review:     reviewSvc,
		Approval:   approvalSvc,
		Audit:      auditSvc,
	}

	// Choose workflow engine: Temporal or local
	var workflowEngine orchestrator.WorkflowEngine

	if cfg.Temporal.Enabled {
		temporalClient, err := orchestrator.StartTemporalWorker(
			orchestrator.TemporalWorkerConfig{
				HostPort:  cfg.Temporal.HostPort,
				Namespace: cfg.Temporal.Namespace,
				TaskQueue: cfg.Temporal.TaskQueue,
			},
			reg, orchServices, workflowSvc, logger,
		)
		if err != nil {
			logger.Error("failed to start Temporal worker", "error", err)
			os.Exit(1)
		}
		defer temporalClient.Close()

		workflowEngine = orchestrator.NewTemporalEngine(temporalClient, cfg.Temporal.TaskQueue, workflowSvc, logger)
		logger.Info("Temporal workflow engine active", "host_port", cfg.Temporal.HostPort)
	} else {
		engine := orchestrator.NewEngine(reg, orchServices, workflowSvc, logger)
		workflowEngine = engine
		logger.Info("local workflow engine active")
	}

	handler.NewOrchestratorHandler(workflowEngine).Register(mux)

	// --- middleware chain ---
	var chain http.Handler = mux
	chain = middleware.Auth(cfg.APIKey)(chain)
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

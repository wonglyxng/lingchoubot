package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/lingchou/lingchoubot/backend/internal/config"
	"github.com/lingchou/lingchoubot/backend/internal/orchestrator"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
	"github.com/lingchou/lingchoubot/backend/internal/runtime"
	"github.com/lingchou/lingchoubot/backend/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()

	if !cfg.Temporal.Enabled {
		logger.Error("TEMPORAL_ENABLED must be true to run the Temporal worker")
		os.Exit(1)
	}

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
	workflowRunRepo := repository.NewWorkflowRunRepo(db)
	workflowStepRepo := repository.NewWorkflowStepRepo(db)

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
	_ = service.NewToolCallService(toolCallRepo, auditSvc)
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

	// --- agent runtime ---
	reg := runtime.NewRegistry()

	if cfg.LLM.Enabled {
		defaultClient := runtime.NewLLMClient(runtime.LLMClientConfig{
			BaseURL: cfg.LLM.BaseURL,
			APIKey:  cfg.LLM.APIKey,
			Model:   cfg.LLM.Model,
		})

		roleClients := make(map[string]*runtime.LLMClient)
		for _, role := range []string{"pm", "supervisor", "worker", "reviewer"} {
			baseURL, apiKey, model := cfg.LLM.ResolveForRole(role)
			if baseURL != cfg.LLM.BaseURL || apiKey != cfg.LLM.APIKey || model != cfg.LLM.Model {
				roleClients[role] = runtime.NewLLMClient(runtime.LLMClientConfig{
					BaseURL: baseURL,
					APIKey:  apiKey,
					Model:   model,
				})
				logger.Info("role-specific LLM configured", "role", role, "model", model)
			}
		}

		providerConfigs := make(map[string]runtime.LLMClientConfig, len(cfg.LLM.Providers))
		for provider, providerCfg := range cfg.LLM.Providers {
			providerConfigs[provider] = runtime.LLMClientConfig{
				BaseURL: providerCfg.BaseURL,
				APIKey:  providerCfg.APIKey,
			}
		}

		runtime.RegisterLLMRunners(reg, defaultClient, roleClients, providerConfigs, logger)
		logger.Info("LLM agent runners registered", "model", cfg.LLM.Model, "base_url", cfg.LLM.BaseURL)
	} else {
		reg.RegisterDefaults()
		logger.Info("mock agent runners registered")
	}

	// --- start Temporal worker ---
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

	logger.Info("Temporal worker process started, waiting for tasks...")

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Temporal worker shutting down")
}

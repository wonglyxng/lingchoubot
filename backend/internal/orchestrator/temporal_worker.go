package orchestrator

import (
	"fmt"
	"log/slog"

	"github.com/lingchou/lingchoubot/backend/internal/runtime"
	"github.com/lingchou/lingchoubot/backend/internal/service"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// TemporalWorkerConfig holds Temporal worker configuration.
type TemporalWorkerConfig struct {
	HostPort  string
	Namespace string
	TaskQueue string
}

// StartTemporalWorker creates a Temporal client, registers the workflow and activities,
// and starts the worker. Returns the client (for shutdown) and any error.
func StartTemporalWorker(
	cfg TemporalWorkerConfig,
	reg *runtime.Registry,
	svc *Services,
	workflowSvc *service.WorkflowService,
	logger *slog.Logger,
) (client.Client, error) {
	c, err := client.Dial(client.Options{
		HostPort:  cfg.HostPort,
		Namespace: cfg.Namespace,
		Logger:    newTemporalLogger(logger),
	})
	if err != nil {
		return nil, fmt.Errorf("dial Temporal: %w", err)
	}

	w := worker.New(c, cfg.TaskQueue, worker.Options{})

	// Register workflow
	w.RegisterWorkflow(ProjectWorkflow)

	// Register activities
	acts := &Activities{
		Registry: reg,
		Services: svc,
		Workflow: workflowSvc,
		Logger:   logger,
	}
	w.RegisterActivity(acts.ActivityPM)
	w.RegisterActivity(acts.ActivityListPhaseTasks)
	w.RegisterActivity(acts.ActivitySupervisor)
	w.RegisterActivity(acts.ActivityWorker)
	w.RegisterActivity(acts.ActivityReviewer)
	w.RegisterActivity(acts.ActivityCompleteRun)
	w.RegisterActivity(acts.ActivityFailRun)

	// Start worker in background
	if err := w.Start(); err != nil {
		c.Close()
		return nil, fmt.Errorf("start Temporal worker: %w", err)
	}

	logger.Info("Temporal worker started",
		"host_port", cfg.HostPort,
		"namespace", cfg.Namespace,
		"task_queue", cfg.TaskQueue,
	)

	return c, nil
}

// temporalLogger adapts slog to Temporal's log.Logger interface.
type temporalLogger struct {
	logger *slog.Logger
}

func newTemporalLogger(l *slog.Logger) *temporalLogger {
	return &temporalLogger{logger: l.With("component", "temporal")}
}

func (l *temporalLogger) Debug(msg string, keyvals ...interface{}) {
	l.logger.Debug(msg, keyvals...)
}

func (l *temporalLogger) Info(msg string, keyvals ...interface{}) {
	l.logger.Info(msg, keyvals...)
}

func (l *temporalLogger) Warn(msg string, keyvals ...interface{}) {
	l.logger.Warn(msg, keyvals...)
}

func (l *temporalLogger) Error(msg string, keyvals ...interface{}) {
	l.logger.Error(msg, keyvals...)
}

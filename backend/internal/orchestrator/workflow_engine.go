package orchestrator

import (
	"context"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

// WorkflowEngine abstracts workflow execution.
// The local Engine and TemporalEngine both implement this interface.
type WorkflowEngine interface {
	RunAsync(ctx context.Context, projectID string) (*model.WorkflowRun, error)
	GetRun(ctx context.Context, id string) (*model.WorkflowRun, error)
	ListRuns(ctx context.Context, params repository.WorkflowRunListParams) ([]*model.WorkflowRun, int, error)
	ResumeRun(ctx context.Context, id string) error
	CancelRun(ctx context.Context, id string) error
}

package service

import (
	"context"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

// approvalRepo defines the repository methods ApprovalRequestService depends on.
type approvalRepo interface {
	Create(ctx context.Context, a *model.ApprovalRequest) error
	GetByID(ctx context.Context, id string) (*model.ApprovalRequest, error)
	List(ctx context.Context, p repository.ApprovalListParams) ([]*model.ApprovalRequest, int, error)
	Decide(ctx context.Context, id string, status model.ApprovalStatus, note string) error
}

type ApprovalRequestService struct {
	repo    approvalRepo
	taskSvc *TaskService
	audit   *AuditService
}

func NewApprovalRequestService(repo *repository.ApprovalRequestRepo, taskSvc *TaskService, audit *AuditService) *ApprovalRequestService {
	return &ApprovalRequestService{repo: repo, taskSvc: taskSvc, audit: audit}
}

func (s *ApprovalRequestService) Create(ctx context.Context, a *model.ApprovalRequest) error {
	if a.ProjectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if a.RequestedBy == "" {
		return fmt.Errorf("requested_by is required")
	}
	if a.Title == "" {
		return fmt.Errorf("title is required")
	}
	if a.ApproverType == "" {
		a.ApproverType = "user"
	}
	if a.Status == "" {
		a.Status = model.ApprovalStatusPending
	}
	if len(a.Metadata) == 0 {
		a.Metadata = model.JSON("{}")
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return fmt.Errorf("create approval request: %w", err)
	}
	s.audit.LogEvent(ctx, "agent", a.RequestedBy, "approval_request.created",
		fmt.Sprintf("审批请求「%s」已发起", a.Title),
		"approval_request", a.ID, nil, a)
	return nil
}

func (s *ApprovalRequestService) GetByID(ctx context.Context, id string) (*model.ApprovalRequest, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ApprovalRequestService) List(ctx context.Context, p repository.ApprovalListParams) ([]*model.ApprovalRequest, int, error) {
	return s.repo.List(ctx, p)
}

// Decide approves or rejects a pending approval request.
// If approved and linked to a task in in_review, advances it to completed.
// If rejected and linked to a task in in_review, moves it to revision_required.
func (s *ApprovalRequestService) Decide(ctx context.Context, id string, status model.ApprovalStatus, note string) error {
	if status != model.ApprovalStatusApproved && status != model.ApprovalStatusRejected {
		return fmt.Errorf("decision must be 'approved' or 'rejected'")
	}

	old, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if old == nil {
		return fmt.Errorf("approval request not found")
	}
	if old.Status != model.ApprovalStatusPending {
		return fmt.Errorf("approval already decided")
	}

	if err := s.repo.Decide(ctx, id, status, note); err != nil {
		return err
	}

	s.audit.LogEvent(ctx, "user", "", "approval_request.decided",
		fmt.Sprintf("审批请求「%s」已%s", old.Title, decisionLabel(status)),
		"approval_request", id,
		map[string]string{"status": string(old.Status)},
		map[string]string{"status": string(status), "decision_note": note},
	)

	if old.TaskID != nil {
		switch status {
		case model.ApprovalStatusApproved:
			_ = s.taskSvc.TransitionStatus(ctx, *old.TaskID, model.TaskStatusCompleted)
		case model.ApprovalStatusRejected:
			_ = s.taskSvc.TransitionStatus(ctx, *old.TaskID, model.TaskStatusRevisionRequired)
		}
	}

	return nil
}

func decisionLabel(s model.ApprovalStatus) string {
	if s == model.ApprovalStatusApproved {
		return "批准"
	}
	return "拒绝"
}

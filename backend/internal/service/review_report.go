package service

import (
	"context"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

type ReviewReportService struct {
	repo        ReviewReportRepository
	taskSvc     *TaskService
	approvalSvc *ApprovalRequestService
	audit       *AuditService
}

func NewReviewReportService(repo ReviewReportRepository, taskSvc *TaskService, audit *AuditService) *ReviewReportService {
	return &ReviewReportService{repo: repo, taskSvc: taskSvc, audit: audit}
}

// SetApprovalService injects the approval service for auto-creating approval
// requests when a review verdict is approved. Called after both services are
// constructed to avoid circular init dependencies.
func (s *ReviewReportService) SetApprovalService(approvalSvc *ApprovalRequestService) {
	s.approvalSvc = approvalSvc
}

// Create persists a review report and, if the verdict is negative, transitions
// the associated task to revision_required.
func (s *ReviewReportService) Create(ctx context.Context, rr *model.ReviewReport) error {
	if rr.TaskID == "" {
		return fmt.Errorf("task_id is required")
	}
	if rr.ReviewerID == "" {
		return fmt.Errorf("reviewer_id is required")
	}
	if rr.Verdict == "" {
		return fmt.Errorf("verdict is required")
	}
	if len(rr.Findings) == 0 {
		rr.Findings = model.JSON("[]")
	}
	if len(rr.Recommendations) == 0 {
		rr.Recommendations = model.JSON("[]")
	}
	if len(rr.Metadata) == 0 {
		rr.Metadata = model.JSON("{}")
	}

	if err := s.repo.Create(ctx, rr); err != nil {
		return fmt.Errorf("create review report: %w", err)
	}

	s.audit.LogEvent(ctx, "agent", rr.ReviewerID, "review_report.created",
		fmt.Sprintf("评审报告已创建，结论: %s", rr.Verdict),
		"review_report", rr.ID, nil, rr)

	if rr.Verdict == model.ReviewVerdictRejected || rr.Verdict == model.ReviewVerdictNeedsRevision {
		if err := s.taskSvc.TransitionStatus(ctx, rr.TaskID, model.TaskStatusRevisionRequired); err != nil {
			return fmt.Errorf("auto-transition task to revision_required: %w", err)
		}
	}

	// Approved: advance task to pending_approval and auto-create an approval request.
	if rr.Verdict == model.ReviewVerdictApproved {
		if err := s.taskSvc.TransitionStatus(ctx, rr.TaskID, model.TaskStatusPendingApproval); err != nil {
			return fmt.Errorf("auto-transition task to pending_approval: %w", err)
		}
		if s.approvalSvc != nil {
			task, _ := s.taskSvc.GetByID(ctx, rr.TaskID)
			if task != nil {
				ar := &model.ApprovalRequest{
					ProjectID:   task.ProjectID,
					TaskID:      &rr.TaskID,
					RequestedBy: rr.ReviewerID,
					Title:       fmt.Sprintf("任务「%s」评审通过，请审批", task.Title),
					Description: fmt.Sprintf("评审报告 %s 结论为通过，等待审批确认", rr.ID),
				}
				if err := s.approvalSvc.Create(ctx, ar); err != nil {
					return fmt.Errorf("auto-create approval request: %w", err)
				}
			}
		}
	}

	return nil
}

func (s *ReviewReportService) GetByID(ctx context.Context, id string) (*model.ReviewReport, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ReviewReportService) List(ctx context.Context, p repository.ReviewListParams) ([]*model.ReviewReport, int, error) {
	return s.repo.List(ctx, p)
}

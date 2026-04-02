package service

import (
	"context"
	"encoding/json"
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
				metadata, err := approvalMetadataFromReview(rr, task)
				if err != nil {
					return fmt.Errorf("build approval metadata: %w", err)
				}
				if rr.RunID != nil {
					metadata["run_id"] = *rr.RunID
				}
				if task.PhaseID != nil {
					metadata["phase_id"] = *task.PhaseID
				}
				metaJSON := model.JSON("{}")
				if len(metadata) > 0 {
					if b, err := json.Marshal(metadata); err == nil {
						metaJSON = model.JSON(b)
					}
				}
				ar := &model.ApprovalRequest{
					ProjectID:   task.ProjectID,
					TaskID:      &rr.TaskID,
					RequestedBy: rr.ReviewerID,
					Title:       fmt.Sprintf("任务「%s」评审通过，请审批", task.Title),
					Description: approvalDescription(rr, metadata),
					Metadata:    metaJSON,
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

func approvalMetadataFromReview(rr *model.ReviewReport, task *model.Task) (map[string]any, error) {
	metadata := map[string]any{
		"review_id":      rr.ID,
		"review_verdict": rr.Verdict,
		"review_summary": rr.Summary,
		"task_title":     task.Title,
	}
	if rr.ArtifactVersionID != nil {
		metadata["artifact_version_id"] = *rr.ArtifactVersionID
	}
	findings, err := jsonArrayToStrings(rr.Findings)
	if err != nil {
		return nil, err
	}
	recommendations, err := jsonArrayToStrings(rr.Recommendations)
	if err != nil {
		return nil, err
	}
	metadata["findings"] = findings
	metadata["recommendations"] = recommendations

	if len(rr.Metadata) == 0 {
		return metadata, nil
	}
	reviewMeta := map[string]any{}
	if err := json.Unmarshal(rr.Metadata, &reviewMeta); err != nil {
		return nil, fmt.Errorf("parse review metadata: %w", err)
	}
	for key, value := range reviewMeta {
		metadata[key] = value
	}
	return metadata, nil
}

func approvalDescription(rr *model.ReviewReport, metadata map[string]any) string {
	description := fmt.Sprintf("评审报告 %s 已通过。结论摘要：%s", rr.ID, rr.Summary)
	if artifactCount, ok := metadata["artifact_count"].(int); ok && artifactCount > 0 {
		description = fmt.Sprintf("%s。关联交付物 %d 个，等待审批确认。", description, artifactCount)
	}
	if artifactCountFloat, ok := metadata["artifact_count"].(float64); ok && int(artifactCountFloat) > 0 {
		description = fmt.Sprintf("%s。关联交付物 %d 个，等待审批确认。", description, int(artifactCountFloat))
	}
	if description == fmt.Sprintf("评审报告 %s 已通过。结论摘要：%s", rr.ID, rr.Summary) {
		return description + "，等待审批确认。"
	}
	return description
}

func jsonArrayToStrings(raw model.JSON) ([]string, error) {
	if len(raw) == 0 {
		return []string{}, nil
	}
	var items []string
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("parse json array: %w", err)
	}
	return items, nil
}

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
	if rr.Verdict == model.ReviewVerdictRejected || rr.Verdict == model.ReviewVerdictNeedsRevision {
		reworkCount, err := s.nextReworkCount(ctx, rr.TaskID)
		if err != nil {
			return fmt.Errorf("resolve rework count: %w", err)
		}
		enriched, brief, err := enrichReviewMetadataForRework(rr, reworkCount)
		if err != nil {
			return fmt.Errorf("build rework metadata: %w", err)
		}
		rr.Metadata = enriched
		if err := s.attachCurrentReworkBriefToTask(ctx, rr.TaskID, brief); err != nil {
			return fmt.Errorf("persist task rework brief: %w", err)
		}
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

func (s *ReviewReportService) nextReworkCount(ctx context.Context, taskID string) (int, error) {
	reviews, _, err := s.repo.List(ctx, repository.ReviewListParams{TaskID: taskID, Limit: 1000, Offset: 0})
	if err != nil {
		return 0, err
	}
	count := 1
	for _, review := range reviews {
		if review.Verdict == model.ReviewVerdictRejected || review.Verdict == model.ReviewVerdictNeedsRevision {
			count++
		}
	}
	return count, nil
}

func (s *ReviewReportService) attachCurrentReworkBriefToTask(ctx context.Context, taskID string, brief map[string]any) error {
	task, err := s.taskSvc.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("task not found")
	}
	metadata, err := jsonObject(task.Metadata)
	if err != nil {
		return fmt.Errorf("parse task metadata: %w", err)
	}
	metadata["current_rework_brief"] = brief
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal task metadata: %w", err)
	}
	updated := *task
	updated.Metadata = model.JSON(encoded)
	return s.taskSvc.Update(ctx, &updated)
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
	enrichApprovalScoreSummary(metadata)
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

func enrichApprovalScoreSummary(metadata map[string]any) {
	hardGateResults, ok := metadata["hard_gate_results"].([]any)
	if ok {
		passed := 0
		for _, item := range hardGateResults {
			entry, isMap := item.(map[string]any)
			if !isMap {
				continue
			}
			if passedValue, exists := entry["passed"].(bool); exists && passedValue {
				passed++
			}
		}
		metadata["hard_gate_total_count"] = len(hardGateResults)
		metadata["hard_gate_passed_count"] = passed
	}

	scoreItems, ok := metadata["score_items"].([]any)
	if !ok {
		return
	}
	summary := make([]map[string]any, 0, len(scoreItems))
	for _, item := range scoreItems {
		entry, isMap := item.(map[string]any)
		if !isMap {
			continue
		}
		summary = append(summary, map[string]any{
			"key":       entry["key"],
			"name":      entry["name"],
			"weight":    entry["weight"],
			"score":     entry["score"],
			"max_score": entry["max_score"],
		})
	}
	if len(summary) > 0 {
		metadata["score_breakdown_summary"] = summary
	}
}

func enrichReviewMetadataForRework(rr *model.ReviewReport, reworkCount int) (model.JSON, map[string]any, error) {
	metadata, err := jsonObject(rr.Metadata)
	if err != nil {
		return nil, nil, fmt.Errorf("parse review metadata: %w", err)
	}
	brief := buildReworkBrief(metadata, rr, reworkCount)
	metadata["rework_count"] = reworkCount
	metadata["rework_brief"] = brief
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal review metadata: %w", err)
	}
	return model.JSON(encoded), brief, nil
}

func buildReworkBrief(metadata map[string]any, rr *model.ReviewReport, reworkCount int) map[string]any {
	failedHardGateKeys := extractFailedHardGateKeys(metadata["hard_gate_results"])
	lowScoreItemKeys := extractLowScoreItemKeys(metadata["score_items"])
	mustFixItems := extractStringSlice(metadata["must_fix_items"])
	if len(mustFixItems) == 0 {
		if findings, err := jsonArrayToStrings(rr.Findings); err == nil {
			mustFixItems = findings
		}
	}
	suggestions := extractStringSlice(metadata["suggestions"])
	if len(suggestions) == 0 {
		if recommendations, err := jsonArrayToStrings(rr.Recommendations); err == nil {
			suggestions = recommendations
		}
	}
	return map[string]any{
		"attempt":                reworkCount,
		"failed_hard_gate_keys":  failedHardGateKeys,
		"low_score_item_keys":    lowScoreItemKeys,
		"must_fix_items":         mustFixItems,
		"suggestions":            suggestions,
		"requires_clarification": len(failedHardGateKeys) > 0,
	}
}

func jsonObject(raw model.JSON) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	parsed := map[string]any{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func extractFailedHardGateKeys(raw any) []string {
	items, ok := raw.([]any)
	if !ok {
		return []string{}
	}
	keys := make([]string, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		passed, _ := entry["passed"].(bool)
		if passed {
			continue
		}
		key, _ := entry["key"].(string)
		if key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}

func extractLowScoreItemKeys(raw any) []string {
	items, ok := raw.([]any)
	if !ok {
		return []string{}
	}
	keys := make([]string, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		score, scoreOK := numberToInt(entry["score"])
		maxScore, maxOK := numberToInt(entry["max_score"])
		key, _ := entry["key"].(string)
		if !scoreOK || !maxOK || maxScore <= 0 || key == "" {
			continue
		}
		if score*100 < maxScore*70 {
			keys = append(keys, key)
		}
	}
	return keys
}

func extractStringSlice(raw any) []string {
	items, ok := raw.([]any)
	if !ok {
		return []string{}
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if ok && text != "" {
			result = append(result, text)
		}
	}
	return result
}

func numberToInt(raw any) (int, bool) {
	switch value := raw.(type) {
	case int:
		return value, true
	case int32:
		return int(value), true
	case int64:
		return int(value), true
	case float64:
		return int(value), true
	default:
		return 0, false
	}
}

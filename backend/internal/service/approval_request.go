package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

type WorkflowResumer interface {
	ResumeRun(ctx context.Context, id string) error
}

type ApprovalRequestService struct {
	repo    ApprovalRepository
	taskSvc *TaskService
	audit   *AuditService
	resumer WorkflowResumer
}

type approvalContinuationWarning struct {
	message string
	runID   string
}

func NewApprovalRequestService(repo ApprovalRepository, taskSvc *TaskService, audit *AuditService) *ApprovalRequestService {
	return &ApprovalRequestService{repo: repo, taskSvc: taskSvc, audit: audit}
}

func (s *ApprovalRequestService) SetWorkflowResumer(resumer WorkflowResumer) {
	s.resumer = resumer
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
// If approved and linked to a task in pending_approval, advances it to completed.
// If rejected and linked to a task in pending_approval, moves it to revision_required.
func (s *ApprovalRequestService) Decide(ctx context.Context, id string, status model.ApprovalStatus, note string) (*model.ApprovalDecisionResult, error) {
	if status != model.ApprovalStatusApproved && status != model.ApprovalStatusRejected {
		return nil, fmt.Errorf("decision must be 'approved' or 'rejected'")
	}

	old, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if old == nil {
		return nil, fmt.Errorf("approval request not found")
	}
	if old.Status != model.ApprovalStatusPending {
		return nil, fmt.Errorf("approval already decided")
	}

	if err := s.repo.Decide(ctx, id, status, note); err != nil {
		return nil, err
	}

	result := &model.ApprovalDecisionResult{
		ID:                   id,
		Status:               status,
		WorkflowResumeStatus: model.WorkflowResumeStatusNotRequested,
	}

	s.audit.LogEvent(ctx, "user", "", "approval_request.decided",
		fmt.Sprintf("审批请求「%s」已%s", old.Title, decisionLabel(status)),
		"approval_request", id,
		map[string]string{"status": string(old.Status)},
		map[string]string{"status": string(status), "decision_note": note},
	)

	if old.TaskID != nil {
		var targetStatus model.TaskStatus
		switch status {
		case model.ApprovalStatusApproved:
			targetStatus = model.TaskStatusCompleted
		case model.ApprovalStatusRejected:
			targetStatus = model.TaskStatusRevisionRequired
		}
		if targetStatus != "" {
			if err := s.taskSvc.TransitionStatus(ctx, *old.TaskID, targetStatus); err != nil {
				warning := fmt.Sprintf("审批已生效，但关联任务状态未更新：%s", err.Error())
				result.Warnings = append(result.Warnings, warning)
				result.WorkflowResumeStatus = model.WorkflowResumeStatusWarning
				result.WorkflowResumeMessage = "任务状态未更新，已跳过工作流恢复"
				s.auditContinuationWarning(ctx, old.ID, "", warning)
				return result, nil
			}
			result.TaskStatus = &targetStatus
		}
		if warning := s.resumeWorkflowIfPhaseUnlocked(ctx, old, result); warning != nil {
			result.Warnings = append(result.Warnings, warning.message)
			s.auditContinuationWarning(ctx, old.ID, warning.runID, warning.message)
		}
	}

	return result, nil
}

func (s *ApprovalRequestService) resumeWorkflowIfPhaseUnlocked(ctx context.Context, approval *model.ApprovalRequest, result *model.ApprovalDecisionResult) *approvalContinuationWarning {
	if s.resumer == nil || approval == nil || approval.TaskID == nil {
		return nil
	}

	task, err := s.taskSvc.GetByID(ctx, *approval.TaskID)
	if err != nil {
		result.WorkflowResumeStatus = model.WorkflowResumeStatusWarning
		result.WorkflowResumeMessage = "读取任务阶段上下文失败，未恢复工作流"
		return &approvalContinuationWarning{message: fmt.Sprintf("审批已生效，但读取任务阶段上下文失败：%s", err.Error())}
	}
	if task == nil || task.PhaseID == nil {
		return nil
	}

	tasks, _, err := s.taskSvc.List(ctx, repository.TaskListParams{
		PhaseID: *task.PhaseID,
		Limit:   100,
		Offset:  0,
	})
	if err != nil {
		result.WorkflowResumeStatus = model.WorkflowResumeStatusWarning
		result.WorkflowResumeMessage = "读取阶段任务失败，未恢复工作流"
		return &approvalContinuationWarning{message: fmt.Sprintf("审批已生效，但读取阶段任务失败：%s", err.Error())}
	}
	for _, phaseTask := range tasks {
		if phaseTask.Status == model.TaskStatusPendingApproval {
			result.WorkflowResumeStatus = model.WorkflowResumeStatusSkipped
			result.WorkflowResumeMessage = "当前阶段仍有待审批任务，暂不恢复工作流"
			return nil
		}
	}

	runID, err := approvalRunID(approval.Metadata)
	if err != nil {
		result.WorkflowResumeStatus = model.WorkflowResumeStatusWarning
		result.WorkflowResumeMessage = "审批元数据异常，未恢复工作流"
		return &approvalContinuationWarning{message: fmt.Sprintf("审批已生效，但解析工作流标识失败：%s", err.Error())}
	}
	if runID == "" {
		result.WorkflowResumeStatus = model.WorkflowResumeStatusSkipped
		result.WorkflowResumeMessage = "审批未绑定工作流运行，无需恢复"
		return nil
	}
	result.WorkflowRunID = runID

	if err := s.resumer.ResumeRun(ctx, runID); err != nil {
		result.WorkflowResumeStatus = model.WorkflowResumeStatusWarning
		result.WorkflowResumeMessage = "审批已生效，但工作流未恢复"
		return &approvalContinuationWarning{runID: runID, message: fmt.Sprintf("审批已生效，但工作流 %s 未恢复：%s", runID, err.Error())}
	}
	result.WorkflowResumeStatus = model.WorkflowResumeStatusResumed
	result.WorkflowResumeMessage = "审批收口后已恢复工作流"
	return nil
}

func (s *ApprovalRequestService) auditContinuationWarning(ctx context.Context, approvalID, runID, message string) {
	after := map[string]string{"message": message}
	if runID != "" {
		after["run_id"] = runID
	}
	s.audit.LogEvent(ctx, "system", "", "approval_request.continuation_warning",
		message,
		"approval_request", approvalID, nil, after)
}

func approvalRunID(raw model.JSON) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	var meta struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return "", err
	}
	return meta.RunID, nil
}

func decisionLabel(s model.ApprovalStatus) string {
	if s == model.ApprovalStatusApproved {
		return "批准"
	}
	return "拒绝"
}

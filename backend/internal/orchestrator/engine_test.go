package orchestrator

import (
	"encoding/json"
	"testing"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/reviewpolicy"
	"github.com/lingchou/lingchoubot/backend/internal/runtime"
)

func TestInferSpecialization(t *testing.T) {
	tests := []struct {
		name string
		task *model.Task
		want model.AgentSpecialization
	}{
		{
			name: "backend by API keyword",
			task: &model.Task{Title: "实现用户管理 API", Description: "CRUD 接口"},
			want: model.AgentSpecBackend,
		},
		{
			name: "backend by 后端 keyword",
			task: &model.Task{Title: "后端服务开发", Description: ""},
			want: model.AgentSpecBackend,
		},
		{
			name: "backend by 数据库 keyword",
			task: &model.Task{Title: "数据库设计", Description: ""},
			want: model.AgentSpecBackend,
		},
		{
			name: "frontend by 前端 keyword",
			task: &model.Task{Title: "前端页面开发", Description: ""},
			want: model.AgentSpecFrontend,
		},
		{
			name: "frontend by React keyword",
			task: &model.Task{Title: "用户列表组件", Description: "使用 React 实现"},
			want: model.AgentSpecFrontend,
		},
		{
			name: "qa by 测试 keyword",
			task: &model.Task{Title: "单元测试编写", Description: ""},
			want: model.AgentSpecQA,
		},
		{
			name: "release by 发布 keyword",
			task: &model.Task{Title: "版本发布", Description: ""},
			want: model.AgentSpecRelease,
		},
		{
			name: "devops by Docker keyword",
			task: &model.Task{Title: "Docker 镜像构建", Description: ""},
			want: model.AgentSpecDevOps,
		},
		{
			name: "general fallback",
			task: &model.Task{Title: "项目计划梳理", Description: "整体协调"},
			want: model.AgentSpecGeneral,
		},
		{
			name: "case insensitive API",
			task: &model.Task{Title: "实现 api 端点", Description: ""},
			want: model.AgentSpecBackend,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferSpecialization(tt.task)
			if got != tt.want {
				t.Errorf("inferSpecialization(%q) = %q, want %q", tt.task.Title, got, tt.want)
			}
		})
	}
}

func TestInferExecutionDomain(t *testing.T) {
	tests := []struct {
		name string
		task *model.Task
		want model.ExecutionDomain
	}{
		{
			name: "development by API keyword",
			task: &model.Task{Title: "实现用户管理 API"},
			want: model.ExecDomainDevelopment,
		},
		{
			name: "development by 后端 keyword",
			task: &model.Task{Title: "后端服务开发"},
			want: model.ExecDomainDevelopment,
		},
		{
			name: "development by 前端 keyword",
			task: &model.Task{Title: "前端页面开发"},
			want: model.ExecDomainDevelopment,
		},
		{
			name: "qa by 测试 keyword",
			task: &model.Task{Title: "单元测试编写"},
			want: model.ExecDomainQA,
		},
		{
			name: "qa by 评审 keyword",
			task: &model.Task{Title: "代码评审"},
			want: model.ExecDomainQA,
		},
		{
			name: "general fallback",
			task: &model.Task{Title: "项目计划梳理", Description: "整体协调"},
			want: model.ExecDomainGeneral,
		},
		{
			name: "explicit domain preserved",
			task: &model.Task{Title: "项目计划梳理", ExecutionDomain: model.ExecDomainQA},
			want: model.ExecDomainQA,
		},
		{
			name: "general domain re-inferred as development",
			task: &model.Task{Title: "实现数据库 migration", ExecutionDomain: model.ExecDomainGeneral},
			want: model.ExecDomainDevelopment,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferExecutionDomain(tt.task)
			if got != tt.want {
				t.Errorf("inferExecutionDomain(%q) = %q, want %q", tt.task.Title, got, tt.want)
			}
		})
	}
}

func TestRoleCodeConstants(t *testing.T) {
	tests := []struct {
		rc   model.RoleCode
		want string
	}{
		{model.RoleCodePMSupervisor, "PM_SUPERVISOR"},
		{model.RoleCodeDevelopmentSupervisor, "DEVELOPMENT_SUPERVISOR"},
		{model.RoleCodeQASupervisor, "QA_SUPERVISOR"},
		{model.RoleCodeGeneralWorker, "GENERAL_WORKER"},
		{model.RoleCodeBackendDevWorker, "BACKEND_DEV_WORKER"},
		{model.RoleCodeFrontendDevWorker, "FRONTEND_DEV_WORKER"},
		{model.RoleCodeQAWorker, "QA_WORKER"},
		{model.RoleCodeReviewerWorker, "REVIEWER_WORKER"},
	}
	for _, tt := range tests {
		if string(tt.rc) != tt.want {
			t.Errorf("RoleCode = %q, want %q", tt.rc, tt.want)
		}
	}
}

func TestExecutionDomainConstants(t *testing.T) {
	tests := []struct {
		d    model.ExecutionDomain
		want string
	}{
		{model.ExecDomainGeneral, "general"},
		{model.ExecDomainDevelopment, "development"},
		{model.ExecDomainQA, "qa"},
	}
	for _, tt := range tests {
		if string(tt.d) != tt.want {
			t.Errorf("ExecutionDomain = %q, want %q", tt.d, tt.want)
		}
	}
}

func TestStringOrEmpty(t *testing.T) {
	s := "hello"
	if got := stringOrEmpty(&s); got != "hello" {
		t.Errorf("stringOrEmpty(&s) = %q, want %q", got, "hello")
	}
	if got := stringOrEmpty(nil); got != "" {
		t.Errorf("stringOrEmpty(nil) = %q, want empty", got)
	}
}

func TestShouldResumeFromPM(t *testing.T) {
	run := &model.WorkflowRun{Steps: []*model.WorkflowStep{{AgentRole: "pm", Status: model.WorkflowStepFailed}}}
	if !shouldResumeFromPM(run) {
		t.Fatal("expected shouldResumeFromPM to return true for failed PM step")
	}

	run = &model.WorkflowRun{Steps: []*model.WorkflowStep{{AgentRole: "worker", Status: model.WorkflowStepFailed}}}
	if shouldResumeFromPM(run) {
		t.Fatal("expected shouldResumeFromPM to return false for non-PM failure")
	}
}

func TestWorkflowRunStatusConstants(t *testing.T) {
	tests := []struct {
		s    model.WorkflowRunStatus
		want string
	}{
		{model.WorkflowRunPending, "pending"},
		{model.WorkflowRunRunning, "running"},
		{model.WorkflowRunWaitingApproval, "waiting_approval"},
		{model.WorkflowRunWaitingManual, "waiting_manual_intervention"},
		{model.WorkflowRunCompleted, "completed"},
		{model.WorkflowRunFailed, "failed"},
		{model.WorkflowRunCancelled, "cancelled"},
	}
	for _, tt := range tests {
		if string(tt.s) != tt.want {
			t.Errorf("WorkflowRunStatus = %q, want %q", tt.s, tt.want)
		}
	}
}

func TestWorkflowStepStatusConstants(t *testing.T) {
	tests := []struct {
		s    model.WorkflowStepStatus
		want string
	}{
		{model.WorkflowStepPending, "pending"},
		{model.WorkflowStepRunning, "running"},
		{model.WorkflowStepCompleted, "completed"},
		{model.WorkflowStepFailed, "failed"},
		{model.WorkflowStepSkipped, "skipped"},
	}
	for _, tt := range tests {
		if string(tt.s) != tt.want {
			t.Errorf("WorkflowStepStatus = %q, want %q", tt.s, tt.want)
		}
	}
}

func TestCheckReworkInputType(t *testing.T) {
	// Verify the CheckReworkInput type is well-formed
	input := CheckReworkInput{
		TaskID:  "task-123",
		Attempt: 2,
	}
	if input.TaskID != "task-123" {
		t.Errorf("expected task-123, got %s", input.TaskID)
	}
	if input.Attempt != 2 {
		t.Errorf("expected attempt 2, got %d", input.Attempt)
	}
}

func TestTaskChainInputSortOffset(t *testing.T) {
	input := TaskChainInput{
		RunID:      "run-1",
		ProjectID:  "proj-1",
		PhaseID:    "phase-1",
		TaskID:     "task-1",
		SortOffset: 5,
	}
	if input.SortOffset != 5 {
		t.Errorf("expected sort_offset 5, got %d", input.SortOffset)
	}
}

func TestBuildContractMetadata_PersistsTrimTraceForExtraScoreItems(t *testing.T) {
	task := &model.Task{Title: "技术可行性评估与架构设计", Description: "输出架构设计"}
	action := runtime.ContractAction{
		TaskTitle:          task.Title,
		TaskCategory:       "architecture",
		ReviewTemplateKey:  "architecture_v1",
		ReviewPolicyReason: "该任务涉及在线数据库迁移，需要突出迁移安全性",
		ReviewPolicySource: []string{"task.description", "acceptance_criteria"},
		ReviewPolicy: map[string]any{
			"score_items": []map[string]any{
				{
					"key":    "technical_feasibility",
					"name":   "技术可行性",
					"weight": 20,
				},
				{
					"key":    "tradeoff_reasoning",
					"name":   "取舍说明",
					"weight": 15,
				},
				{
					"key":    "constraint_alignment",
					"name":   "约束一致性",
					"weight": 10,
				},
				{
					"key":    "implementation_guidance",
					"name":   "实施指导性",
					"weight": 20,
				},
				{
					"key":    "risk_control",
					"name":   "风险控制",
					"weight": 15,
				},
				{
					"key":    "extra_b",
					"name":   "业务一致性",
					"weight": 10,
				},
				{
					"key":    "extra_c",
					"name":   "团队协作性",
					"weight": 1,
				},
				{
					"key":    "extra_a",
					"name":   "方案落地收益",
					"weight": 10,
				},
			},
		},
	}

	metadata, err := buildContractMetadata(task, action)
	if err != nil {
		t.Fatalf("buildContractMetadata: %v", err)
	}

	policy, ok := metadata["review_policy"].(*reviewpolicy.ResolvedPolicy)
	if !ok {
		t.Fatalf("metadata[review_policy] type = %T, want *reviewpolicy.ResolvedPolicy", metadata["review_policy"])
	}
	if policy.ResolutionTrace == nil || policy.ResolutionTrace.ExtraScoreItemsTrim == nil {
		t.Fatal("expected resolution trace in review policy")
	}
	trace := policy.ResolutionTrace.ExtraScoreItemsTrim
	if len(trace.DroppedExtraScoreItems) != 1 || trace.DroppedExtraScoreItems[0].Key != "extra_c" {
		t.Fatalf("dropped extra score items = %#v, want [extra_c]", trace.DroppedExtraScoreItems)
	}
	if trace.KeptExtraScoreItems[0].Key != "extra_a" || trace.KeptExtraScoreItems[1].Key != "extra_b" {
		t.Fatalf("kept extra score items = %#v, want extra_a then extra_b", trace.KeptExtraScoreItems)
	}

	if _, ok := metadata["review_policy_override"]; !ok {
		t.Fatal("expected review_policy_override metadata to preserve original override")
	}
	if got := metadata["review_policy_override_reason"]; got != "该任务涉及在线数据库迁移，需要突出迁移安全性" {
		t.Fatalf("metadata[review_policy_override_reason] = %#v", got)
	}
	sources, ok := metadata["review_policy_override_source"].([]string)
	if !ok {
		t.Fatalf("metadata[review_policy_override_source] type = %T, want []string", metadata["review_policy_override_source"])
	}
	if len(sources) != 2 || sources[0] != "task.description" || sources[1] != "acceptance_criteria" {
		t.Fatalf("metadata[review_policy_override_source] = %#v, want [task.description acceptance_criteria]", sources)
	}
}

func TestLoadContractReviewPolicyOverrideContext(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"review_policy_override_reason": "该任务有迁移窗口约束",
		"review_policy_override_source": []string{"task.description", "user instruction", " "},
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	reason, sources, err := loadContractReviewPolicyOverrideContext(raw)
	if err != nil {
		t.Fatalf("loadContractReviewPolicyOverrideContext: %v", err)
	}
	if reason != "该任务有迁移窗口约束" {
		t.Fatalf("reason = %q, want %q", reason, "该任务有迁移窗口约束")
	}
	if len(sources) != 2 || sources[0] != "task.description" || sources[1] != "user instruction" {
		t.Fatalf("sources = %#v, want [task.description user instruction]", sources)
	}
}

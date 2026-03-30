package orchestrator

import (
	"testing"

	"github.com/lingchou/lingchoubot/backend/internal/model"
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

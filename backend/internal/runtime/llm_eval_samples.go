package runtime

// EvalSample represents a fixed evaluation input for regression testing of LLM output quality.
type EvalSample struct {
	Name        string          `json:"name"`
	Role        string          `json:"role"`
	Spec        string          `json:"spec,omitempty"`
	Input       *AgentTaskInput `json:"input"`
	Description string          `json:"description"`
}

// EvalSamples returns the canonical set of evaluation samples for all roles.
// These are used to verify LLM output quality before and after model upgrades.
func EvalSamples() []EvalSample {
	return []EvalSample{
		pmEvalSample(),
		supervisorEvalSample(),
		workerBackendEvalSample(),
		reviewerEvalSample(),
	}
}

func pmEvalSample() EvalSample {
	return EvalSample{
		Name:        "pm_project_decompose",
		Role:        "pm",
		Description: "PM 将一个用户管理系统项目分解为阶段和任务",
		Input: &AgentTaskInput{
			RunID:       "eval-run-001",
			AgentID:     "eval-pm-001",
			AgentRole:   "pm",
			Instruction: "请将以下项目分解为阶段和任务。",
			Project: &ProjectCtx{
				ID:          "eval-proj-001",
				Name:        "用户管理系统",
				Description: "构建一个包含用户注册、登录、权限管理、用户资料编辑的后台管理系统。技术栈：Go 后端 + React 前端 + PostgreSQL 数据库。",
			},
		},
	}
}

func supervisorEvalSample() EvalSample {
	return EvalSample{
		Name:        "supervisor_contract_creation",
		Role:        "supervisor",
		Description: "Supervisor 为后端 API 开发任务创建执行契约",
		Input: &AgentTaskInput{
			RunID:       "eval-run-002",
			AgentID:     "eval-sup-001",
			AgentRole:   "supervisor",
			Instruction: "请为以下任务创建执行契约并分派执行者。",
			Project: &ProjectCtx{
				ID:          "eval-proj-001",
				Name:        "用户管理系统",
				Description: "用户管理系统后台",
			},
			Phase: &PhaseCtx{
				ID:          "eval-phase-001",
				ProjectID:   "eval-proj-001",
				Name:        "后端开发阶段",
				Description: "实现后端 API 和数据库",
				SortOrder:   1,
			},
			Task: &TaskCtx{
				ID:          "eval-task-001",
				ProjectID:   "eval-proj-001",
				PhaseID:     "eval-phase-001",
				Title:       "后端用户注册 API 开发",
				Description: "实现用户注册接口，包含参数校验、密码加密、邮箱唯一性检查，返回创建的用户信息。",
				Priority:    4,
			},
		},
	}
}

func workerBackendEvalSample() EvalSample {
	return EvalSample{
		Name:        "worker_backend_execution",
		Role:        "worker",
		Spec:        "backend",
		Description: "Worker 根据契约执行后端 API 开发并产出工件",
		Input: &AgentTaskInput{
			RunID:       "eval-run-003",
			AgentID:     "eval-wrk-001",
			AgentRole:   "worker",
			Instruction: "请根据任务契约执行后端开发并产出工件。",
			Project: &ProjectCtx{
				ID:          "eval-proj-001",
				Name:        "用户管理系统",
				Description: "用户管理系统后台",
			},
			Task: &TaskCtx{
				ID:          "eval-task-001",
				ProjectID:   "eval-proj-001",
				PhaseID:     "eval-phase-001",
				Title:       "后端用户注册 API 开发",
				Description: "实现用户注册接口",
				Priority:    4,
			},
			Contract: &ContractCtx{
				ID:                 "eval-contract-001",
				Scope:              "实现 POST /api/v1/users/register 接口",
				DoneDefinition:     "接口能接收用户名、邮箱、密码并返回 201；密码必须 bcrypt 加密后存储",
				VerificationPlan:   "使用 curl 或 httptest 验证注册成功和重复邮箱报错",
				AcceptanceCriteria: "参数校验完整；密码不以明文存储；邮箱重复返回 409",
			},
		},
	}
}

func reviewerEvalSample() EvalSample {
	return EvalSample{
		Name:        "reviewer_code_review",
		Role:        "reviewer",
		Description: "Reviewer 评审后端 Worker 产出的代码工件",
		Input: &AgentTaskInput{
			RunID:       "eval-run-004",
			AgentID:     "eval-rev-001",
			AgentRole:   "reviewer",
			Instruction: "请评审以下工件的质量。",
			Project: &ProjectCtx{
				ID:          "eval-proj-001",
				Name:        "用户管理系统",
				Description: "用户管理系统后台",
			},
			Task: &TaskCtx{
				ID:          "eval-task-001",
				ProjectID:   "eval-proj-001",
				PhaseID:     "eval-phase-001",
				Title:       "后端用户注册 API 开发",
				Description: "实现用户注册接口",
				Priority:    4,
			},
			Artifacts: []ArtifactCtx{
				{
					ID:           "eval-art-001",
					Name:         "user_register_handler.go",
					ArtifactType: "code",
					VersionURI:   "artifact://eval-proj-001/user_register_handler.go",
				},
			},
		},
	}
}

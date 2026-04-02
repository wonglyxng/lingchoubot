package runtime

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// MockWorkerAgent simulates a Worker agent.
// It produces artifacts and a handoff snapshot for the assigned task.
type MockWorkerAgent struct{}

func (a *MockWorkerAgent) Role() string { return "worker" }

func (a *MockWorkerAgent) Execute(input *AgentTaskInput) (*AgentTaskOutput, error) {
	if input.Task == nil {
		return &AgentTaskOutput{
			Status: OutputStatusFailed,
			Error:  "worker agent requires task context",
		}, nil
	}

	task := input.Task
	artifactType := inferArtifactType(task.Title)
	content := fmt.Sprintf("# %s\n\n由 Mock Worker Agent 自动生成。\n\n任务: %s\n描述: %s",
		task.Title, task.Title, task.Description)
	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))

	artifacts := []ArtifactAction{
		{
			Name:         fmt.Sprintf("%s-交付物", task.Title),
			ArtifactType: artifactType,
			Description:  fmt.Sprintf("任务「%s」的核心交付物", task.Title),
			URI:          fmt.Sprintf("mock://artifacts/%s/%s", task.ID, artifactType),
			ContentType:  "text/markdown",
			SizeBytes:    int64(len(content)),
			Content:      content,
			Metadata:     map[string]any{"checksum": checksum, "generator": "mock_worker"},
		},
	}

	handoffs := []HandoffAction{
		{
			Summary: fmt.Sprintf("任务「%s」已完成执行，交付物已生成", task.Title),
			CompletedItems: []string{
				fmt.Sprintf("完成「%s」核心内容编写", task.Title),
				"交付物已上传至工件存储",
			},
			PendingItems: []string{
				"等待独立评审",
			},
			Risks: []string{
				"Mock 生成内容仅作示例，需人工确认质量",
			},
			NextSteps: []string{
				"提交独立评审",
				"根据评审意见修订（如需要）",
			},
		},
	}

	transitions := []TransitionAction{
		{TaskTitle: task.Title, NewStatus: "in_review"},
	}

	return &AgentTaskOutput{
		Status:      OutputStatusNeedsReview,
		Summary:     fmt.Sprintf("任务「%s」执行完成，已生成交付物，等待评审", task.Title),
		Artifacts:   artifacts,
		Handoffs:    handoffs,
		Transitions: transitions,
	}, nil
}

func inferArtifactType(taskTitle string) string {
	titleTypes := map[string]string{
		"需求": "prd",
		"PRD":  "prd",
		"架构": "design",
		"设计": "design",
		"可行性": "design",
		"分析":  "design",
		"评估":  "design",
		"方案":  "design",
		"数据库": "schema_sql",
		"API":  "api_spec",
		"接口": "api_spec",
		"开发": "source_code",
		"编码": "source_code",
		"测试": "test_report",
		"部署": "deployment_plan",
		"发布": "release_note",
	}
	for keyword, artType := range titleTypes {
		if strings.Contains(taskTitle, keyword) {
			return artType
		}
	}
	return "other"
}

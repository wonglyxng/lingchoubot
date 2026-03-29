package gateway

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"
)

// DocGeneratorTool generates structured documents based on input parameters.
// In MVP this produces mock Markdown content; future versions will call an LLM.
type DocGeneratorTool struct{}

func (t *DocGeneratorTool) Name() string        { return "doc_generator" }
func (t *DocGeneratorTool) Description() string  { return "生成结构化文档（需求文档、设计文档、测试报告等）" }
func (t *DocGeneratorTool) RequiredPermissions() []string { return []string{"tool.doc_generator"} }

func (t *DocGeneratorTool) Execute(ctx context.Context, input map[string]any) (*ToolResult, error) {
	docType, _ := input["doc_type"].(string)
	title, _ := input["title"].(string)
	context_, _ := input["context"].(string)

	if title == "" {
		return &ToolResult{Status: "failed", Error: "title is required"}, nil
	}
	if docType == "" {
		docType = "general"
	}

	content := generateDocContent(docType, title, context_)
	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))

	return &ToolResult{
		Status: "success",
		Output: map[string]any{
			"title":        title,
			"doc_type":     docType,
			"content":      content,
			"content_type": "text/markdown",
			"size_bytes":   len(content),
			"checksum":     checksum,
			"generated_at": time.Now().Format(time.RFC3339),
		},
	}, nil
}

func generateDocContent(docType, title, ctx string) string {
	header := fmt.Sprintf("# %s\n\n> 由 doc_generator 工具自动生成\n> 生成时间: %s\n\n",
		title, time.Now().Format("2006-01-02 15:04:05"))

	switch docType {
	case "prd":
		return header + fmt.Sprintf("## 1. 概述\n\n%s\n\n## 2. 目标用户\n\n待补充\n\n## 3. 功能需求\n\n待补充\n\n## 4. 非功能需求\n\n待补充\n\n## 5. 验收标准\n\n待补充\n", ctx)
	case "design":
		return header + fmt.Sprintf("## 1. 设计目标\n\n%s\n\n## 2. 架构方案\n\n待补充\n\n## 3. 数据模型\n\n待补充\n\n## 4. 接口设计\n\n待补充\n\n## 5. 风险评估\n\n待补充\n", ctx)
	case "test_report":
		return header + fmt.Sprintf("## 1. 测试范围\n\n%s\n\n## 2. 测试用例\n\n| 编号 | 用例名称 | 预期结果 | 实际结果 | 状态 |\n|------|----------|----------|----------|------|\n| TC-01 | 基础功能验证 | 通过 | 通过 | PASS |\n\n## 3. 测试结论\n\n所有测试用例通过。\n", ctx)
	default:
		return header + fmt.Sprintf("## 内容\n\n%s\n", ctx)
	}
}

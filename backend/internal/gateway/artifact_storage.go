package gateway

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"
)

// ArtifactStorageTool simulates storing an artifact to object storage.
// In MVP this generates a mock URI; future versions will upload to MinIO.
type ArtifactStorageTool struct{}

func (t *ArtifactStorageTool) Name() string        { return "artifact_storage" }
func (t *ArtifactStorageTool) Description() string  { return "将工件内容存储到对象存储（MVP 模拟写入）" }
func (t *ArtifactStorageTool) RequiredPermissions() []string { return []string{"tool.artifact_storage"} }

func (t *ArtifactStorageTool) Execute(ctx context.Context, input map[string]any) (*ToolResult, error) {
	name, _ := input["name"].(string)
	content, _ := input["content"].(string)
	contentType, _ := input["content_type"].(string)

	if name == "" {
		return &ToolResult{Status: "failed", Error: "name is required"}, nil
	}
	if content == "" {
		return &ToolResult{Status: "failed", Error: "content is required"}, nil
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
	uri := fmt.Sprintf("mock://minio/lingchou-artifacts/%s/%s", time.Now().Format("20060102"), name)

	return &ToolResult{
		Status: "success",
		Output: map[string]any{
			"uri":          uri,
			"name":         name,
			"content_type": contentType,
			"size_bytes":   len(content),
			"checksum":     checksum,
			"stored_at":    time.Now().Format(time.RFC3339),
		},
	}, nil
}

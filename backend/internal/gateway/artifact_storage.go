package gateway

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/lingchou/lingchoubot/backend/internal/config"
)

// ArtifactStorageTool uploads artifact content to MinIO object storage.
// When MinIO is unreachable it falls back to a mock URI so development without
// a running MinIO instance still works.
type ArtifactStorageTool struct {
	client *minio.Client
	bucket string
	logger *slog.Logger
}

// NewArtifactStorageTool creates the tool and eagerly tries to ensure the
// target bucket exists. Failure is logged but not fatal (fallback mode).
func NewArtifactStorageTool(cfg config.MinIOConfig, logger *slog.Logger) *ArtifactStorageTool {
	t := &ArtifactStorageTool{bucket: cfg.Bucket, logger: logger}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		logger.Warn("minio client init failed, using fallback mode", "error", err)
		return t
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		logger.Warn("minio bucket check failed, using fallback mode", "error", err)
		return t
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			logger.Warn("minio make bucket failed", "bucket", cfg.Bucket, "error", err)
		} else {
			logger.Info("minio bucket created", "bucket", cfg.Bucket)
		}
	}

	t.client = client
	return t
}

func (t *ArtifactStorageTool) Name() string { return "artifact_storage" }
func (t *ArtifactStorageTool) Description() string {
	return "将工件内容存储到 MinIO 对象存储"
}
func (t *ArtifactStorageTool) RequiredPermissions() []string {
	return []string{"tool.artifact_storage"}
}

// Actions returns the available actions for the artifact storage tool.
func (t *ArtifactStorageTool) Actions() []string {
	return []string{"read", "write", "delete"}
}

// RiskLevel returns the risk level per action.
func (t *ArtifactStorageTool) RiskLevel(action string) RiskLevel {
	switch action {
	case "delete":
		return RiskCritical
	case "write":
		return RiskNormal
	default:
		return RiskNormal
	}
}

func (t *ArtifactStorageTool) Execute(ctx context.Context, input map[string]any) (*ToolResult, error) {
	name, _ := input["name"].(string)
	content, _ := input["content"].(string)
	contentType, _ := input["content_type"].(string)

	uri, sizeBytes, checksum, err := t.Store(ctx, name, content, contentType)
	if err != nil {
		return &ToolResult{Status: "failed", Error: err.Error()}, nil
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return t.buildResult(uri, name, contentType, int(sizeBytes), checksum), nil
}

func (t *ArtifactStorageTool) Store(ctx context.Context, name, content, contentType string) (string, int64, string, error) {
	if name == "" {
		return "", 0, "", fmt.Errorf("name is required")
	}
	if content == "" {
		return "", 0, "", fmt.Errorf("content is required")
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
	objectKey := fmt.Sprintf("%s/%s", time.Now().Format("20060102"), name)

	// If no MinIO client, fall back to mock URI.
	if t.client == nil {
		uri := fmt.Sprintf("mock://minio/%s/%s", t.bucket, objectKey)
		t.logger.Info("artifact stored (fallback)", "uri", uri)
		return uri, int64(len(content)), checksum, nil
	}

	reader := strings.NewReader(content)
	info, err := t.client.PutObject(ctx, t.bucket, objectKey, reader, int64(len(content)),
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", 0, "", fmt.Errorf("minio upload failed: %v", err)
	}

	uri := fmt.Sprintf("s3://%s/%s", t.bucket, info.Key)
	t.logger.Info("artifact stored", "uri", uri, "size", info.Size)
	return uri, info.Size, checksum, nil
}

func (t *ArtifactStorageTool) buildResult(uri, name, contentType string, size int, checksum string) *ToolResult {
	return &ToolResult{
		Status: "success",
		Output: map[string]any{
			"uri":          uri,
			"name":         name,
			"content_type": contentType,
			"size_bytes":   size,
			"checksum":     checksum,
			"stored_at":    time.Now().Format(time.RFC3339),
		},
	}
}

// NewMockArtifactStorageTool returns a tool instance without MinIO connectivity,
// useful for testing.
func NewMockArtifactStorageTool() *ArtifactStorageTool {
	return &ArtifactStorageTool{
		bucket: "lingchou-artifacts",
		logger: slog.Default(),
	}
}

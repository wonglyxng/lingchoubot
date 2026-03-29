package gateway

import (
	"context"
	"strings"
	"testing"
)

func TestArtifactStorageTool_Fallback(t *testing.T) {
	tool := NewMockArtifactStorageTool()

	if tool.Name() != "artifact_storage" {
		t.Fatalf("unexpected name: %s", tool.Name())
	}

	t.Run("success", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"name":         "test-report.md",
			"content":      "# Test Report\nAll good.",
			"content_type": "text/markdown",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != "success" {
			t.Fatalf("expected success, got %s: %s", result.Status, result.Error)
		}
		uri, _ := result.Output["uri"].(string)
		if !strings.HasPrefix(uri, "mock://minio/lingchou-artifacts/") {
			t.Fatalf("unexpected URI: %s", uri)
		}
		if !strings.HasSuffix(uri, "/test-report.md") {
			t.Fatalf("URI should end with filename: %s", uri)
		}
		if result.Output["checksum"] == "" {
			t.Fatal("checksum should not be empty")
		}
		sizeBytes, _ := result.Output["size_bytes"].(int)
		if sizeBytes <= 0 {
			t.Fatalf("unexpected size: %v", result.Output["size_bytes"])
		}
	})

	t.Run("missing name", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"content": "some content",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != "failed" {
			t.Fatalf("expected failed, got %s", result.Status)
		}
	})

	t.Run("missing content", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"name": "test.md",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != "failed" {
			t.Fatalf("expected failed, got %s", result.Status)
		}
	})

	t.Run("default content type", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"name":    "binary.bin",
			"content": "data",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		ct, _ := result.Output["content_type"].(string)
		if ct != "application/octet-stream" {
			t.Fatalf("expected default content type, got %s", ct)
		}
	})
}

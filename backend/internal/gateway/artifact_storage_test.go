package gateway

import (
	"context"
	"testing"
)

func TestArtifactStorageTool_StrictFailure(t *testing.T) {
	tool := NewMockArtifactStorageTool()

	if tool.Name() != "artifact_storage" {
		t.Fatalf("unexpected name: %s", tool.Name())
	}

	t.Run("storage unavailable", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"name":         "test-report.md",
			"content":      "# Test Report\nAll good.",
			"content_type": "text/markdown",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != "failed" {
			t.Fatalf("expected failed, got %s: %s", result.Status, result.Error)
		}
		if result.Error == "" {
			t.Fatal("expected explicit unavailable error")
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
		if result.Status != "failed" {
			t.Fatalf("expected failed, got %s", result.Status)
		}
	})
}

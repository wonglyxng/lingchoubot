package service

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/lingchou/lingchoubot/backend/internal/model"
)

func TestArtifactServiceAddVersion_AllowsNullMetadata(t *testing.T) {
	auditRepo := &fakeAuditRepo{}
	auditSvc := NewAuditService(auditRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	artifactRepo := &fakeArtifactRepo{}
	versionRepo := &fakeArtifactVersionRepo{}
	svc := NewArtifactService(artifactRepo, versionRepo, auditSvc)

	artifact := &model.Artifact{
		ProjectID:    "project-1",
		Name:         "demo-prd",
		ArtifactType: model.ArtifactTypePRD,
	}
	if err := svc.Create(context.Background(), artifact); err != nil {
		t.Fatalf("create artifact: %v", err)
	}

	version := &model.ArtifactVersion{
		ArtifactID:  artifact.ID,
		ContentType: "text/markdown",
		Content:     "# PRD",
		Metadata:    model.JSON("null"),
	}
	if err := svc.AddVersion(context.Background(), version); err != nil {
		t.Fatalf("add version with null metadata: %v", err)
	}

	meta := map[string]any{}
	if err := json.Unmarshal(version.Metadata, &meta); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if meta["stored_in"] != "inline" {
		t.Fatalf("expected stored_in=inline, got %#v", meta["stored_in"])
	}
	if meta["inline_content"] != "# PRD" {
		t.Fatalf("expected inline_content to be preserved, got %#v", meta["inline_content"])
	}
	if version.URI == "" {
		t.Fatal("expected inline URI to be populated")
	}
}

package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

type auditRepoStub struct{}

func (auditRepoStub) Create(context.Context, *model.AuditLog) error {
	return nil
}

func (auditRepoStub) List(context.Context, repository.AuditListParams) ([]*model.AuditLog, int, error) {
	return nil, 0, nil
}

func (auditRepoStub) ProjectTimeline(context.Context, string, int, int) ([]*model.AuditLog, int, error) {
	return nil, 0, nil
}

func (auditRepoStub) TaskTimeline(context.Context, string, int, int) ([]*model.AuditLog, int, error) {
	return nil, 0, nil
}

func TestAuditServicePublishEvent_NormalizesWorkflowTargetID(t *testing.T) {
	hub := NewEventHub()
	sub := hub.Subscribe([]string{"workflow"})
	defer hub.Unsubscribe(sub)

	svc := NewAuditService(auditRepoStub{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc.SetEventHub(hub)

	afterState := model.JSON(`{"run_id":"run-123","phase_id":"phase-1"}`)
	entry := &model.AuditLog{
		ID:        "evt-1",
		EventType: "workflow.waiting_manual_intervention",
		TargetType: "project",
		TargetID:  "project-456",
		AfterState: &afterState,
	}

	svc.Log(context.Background(), entry)

	select {
	case event := <-sub.Ch:
		if event.TargetID != "run-123" {
			t.Fatalf("expected workflow event target_id to be run id, got %q", event.TargetID)
		}
		if event.ProjectID != "project-456" {
			t.Fatalf("expected workflow event project_id to be preserved, got %q", event.ProjectID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for workflow event")
	}
}

func TestAuditServicePublishEvent_PreservesNonWorkflowTargetID(t *testing.T) {
	hub := NewEventHub()
	sub := hub.Subscribe([]string{"tool_call"})
	defer hub.Unsubscribe(sub)

	svc := NewAuditService(auditRepoStub{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc.SetEventHub(hub)

	entry := &model.AuditLog{
		ID:         "evt-2",
		EventType:  "tool_call.completed",
		TargetType: "tool_call",
		TargetID:   "tool-789",
	}

	svc.Log(context.Background(), entry)

	select {
	case event := <-sub.Ch:
		if event.TargetID != "tool-789" {
			t.Fatalf("expected non-workflow target_id to be unchanged, got %q", event.TargetID)
		}
		if event.ProjectID != "" {
			t.Fatalf("expected non-workflow event project_id to be empty, got %q", event.ProjectID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for tool_call event")
	}
}
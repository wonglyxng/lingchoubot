package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lingchou/lingchoubot/backend/internal/middleware"
	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/service"
	"github.com/lingchou/lingchoubot/backend/internal/testutil"
)

func newAgentHandlerForTest() *AgentHandler {
	repo := testutil.NewFakeAgentRepo()
	auditRepo := testutil.NewFakeAuditRepo()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditSvc := service.NewAuditService(auditRepo, logger)
	agentSvc := service.NewAgentService(repo, auditSvc)
	return NewAgentHandler(agentSvc)
}

func TestAgentHandlerCreateRoleCodeConflict(t *testing.T) {
	h := newAgentHandlerForTest()
	mux := http.NewServeMux()
	h.Register(mux)

	firstBody := []byte(`{"name":"PM Agent","role":"pm","role_code":"PM_SUPERVISOR","status":"active"}`)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(firstBody))
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("first create status = %d, want %d: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	duplicateBody := []byte(`{"name":"Another PM","role":"pm","role_code":"PM_SUPERVISOR","status":"active"}`)
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(duplicateBody))
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("duplicate create status = %d, want %d: %s", w.Code, http.StatusConflict, w.Body.String())
	}

	var resp middleware.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != "ROLE_CODE_CONFLICT" {
		t.Fatalf("error code = %+v, want ROLE_CODE_CONFLICT", resp.Error)
	}
}

func TestAgentHandlerUpdateRoleCodeConflict(t *testing.T) {
	repo := testutil.NewFakeAgentRepo()
	auditRepo := testutil.NewFakeAuditRepo()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditSvc := service.NewAuditService(auditRepo, logger)
	agentSvc := service.NewAgentService(repo, auditSvc)
	h := NewAgentHandler(agentSvc)
	mux := http.NewServeMux()
	h.Register(mux)

	ctx := context.Background()
	if err := agentSvc.Create(ctx, &model.Agent{
		Name:           "PM Agent",
		Role:           model.AgentRolePM,
		RoleCode:       model.RoleCodePMSupervisor,
		Status:         model.AgentStatusActive,
		AgentType:      model.AgentTypeMock,
		Specialization: model.AgentSpecGeneral,
	}); err != nil {
		t.Fatalf("seed PM: %v", err)
	}
	worker := &model.Agent{
		Name:           "Backend Worker",
		Role:           model.AgentRoleWorker,
		RoleCode:       model.RoleCodeBackendDevWorker,
		Status:         model.AgentStatusActive,
		AgentType:      model.AgentTypeMock,
		Specialization: model.AgentSpecBackend,
	}
	if err := agentSvc.Create(ctx, worker); err != nil {
		t.Fatalf("seed worker: %v", err)
	}

	updateBody := []byte(`{"name":"Backend Worker","role":"worker","role_code":"PM_SUPERVISOR","status":"active","agent_type":"mock","specialization":"backend"}`)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/api/v1/agents/"+worker.ID, bytes.NewReader(updateBody))
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("update conflict status = %d, want %d: %s", w.Code, http.StatusConflict, w.Body.String())
	}

	var resp middleware.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != "ROLE_CODE_CONFLICT" {
		t.Fatalf("error code = %+v, want ROLE_CODE_CONFLICT", resp.Error)
	}
}

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lingchou/lingchoubot/backend/internal/model"
	"github.com/lingchou/lingchoubot/backend/internal/orchestrator"
	"github.com/lingchou/lingchoubot/backend/internal/repository"
)

// mockWorkflowEngine implements orchestrator.WorkflowEngine for handler tests.
type mockWorkflowEngine struct {
	runAsyncFn                  func(ctx context.Context, projectID string) (*model.WorkflowRun, error)
	getRunFn                    func(ctx context.Context, id string) (*model.WorkflowRun, error)
	listRunsFn                  func(ctx context.Context, params repository.WorkflowRunListParams) ([]*model.WorkflowRun, int, error)
	resumeRunFn                 func(ctx context.Context, id string) error
	resolveManualInterventionFn func(ctx context.Context, id string, action model.ManualInterventionAction, note string) error
	cancelRunFn                 func(ctx context.Context, id string) error
}

// Verify interface compliance
var _ orchestrator.WorkflowEngine = (*mockWorkflowEngine)(nil)

func (m *mockWorkflowEngine) RunAsync(ctx context.Context, projectID string) (*model.WorkflowRun, error) {
	if m.runAsyncFn != nil {
		return m.runAsyncFn(ctx, projectID)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockWorkflowEngine) GetRun(ctx context.Context, id string) (*model.WorkflowRun, error) {
	if m.getRunFn != nil {
		return m.getRunFn(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockWorkflowEngine) ListRuns(ctx context.Context, params repository.WorkflowRunListParams) ([]*model.WorkflowRun, int, error) {
	if m.listRunsFn != nil {
		return m.listRunsFn(ctx, params)
	}
	return nil, 0, fmt.Errorf("not implemented")
}

func (m *mockWorkflowEngine) ResumeRun(ctx context.Context, id string) error {
	if m.resumeRunFn != nil {
		return m.resumeRunFn(ctx, id)
	}
	return fmt.Errorf("not implemented")
}

func (m *mockWorkflowEngine) ResolveManualIntervention(ctx context.Context, id string, action model.ManualInterventionAction, note string) error {
	if m.resolveManualInterventionFn != nil {
		return m.resolveManualInterventionFn(ctx, id, action, note)
	}
	return fmt.Errorf("not implemented")
}

func (m *mockWorkflowEngine) CancelRun(ctx context.Context, id string) error {
	if m.cancelRunFn != nil {
		return m.cancelRunFn(ctx, id)
	}
	return fmt.Errorf("not implemented")
}

func TestOrchestratorHandler_StartRun_Success(t *testing.T) {
	now := time.Now()
	engine := &mockWorkflowEngine{
		runAsyncFn: func(ctx context.Context, projectID string) (*model.WorkflowRun, error) {
			if projectID != "proj-123" {
				t.Errorf("expected project_id 'proj-123', got %q", projectID)
			}
			return &model.WorkflowRun{
				ID:        "run-001",
				ProjectID: projectID,
				Status:    model.WorkflowRunRunning,
				CreatedAt: now,
			}, nil
		},
	}

	h := NewOrchestratorHandler(engine)
	body := `{"project_id":"proj-123"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/orchestrator/runs", strings.NewReader(body))

	h.StartRun(w, r)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["success"] != true {
		t.Errorf("expected success=true, got %v", resp["success"])
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be an object, got %T", resp["data"])
	}
	if data["id"] != "run-001" {
		t.Errorf("expected run ID 'run-001', got %v", data["id"])
	}
}

func TestOrchestratorHandler_StartRun_MissingProjectID(t *testing.T) {
	h := NewOrchestratorHandler(&mockWorkflowEngine{})

	body := `{"project_id":""}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/orchestrator/runs", strings.NewReader(body))

	h.StartRun(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrchestratorHandler_StartRun_InvalidBody(t *testing.T) {
	h := NewOrchestratorHandler(&mockWorkflowEngine{})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/orchestrator/runs", strings.NewReader("not json"))

	h.StartRun(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrchestratorHandler_StartRun_EngineError(t *testing.T) {
	engine := &mockWorkflowEngine{
		runAsyncFn: func(ctx context.Context, projectID string) (*model.WorkflowRun, error) {
			return nil, fmt.Errorf("project not found")
		},
	}

	h := NewOrchestratorHandler(engine)
	body := `{"project_id":"bad-id"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/orchestrator/runs", strings.NewReader(body))

	h.StartRun(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestOrchestratorHandler_StartRun_PrecheckFailed(t *testing.T) {
	engine := &mockWorkflowEngine{
		runAsyncFn: func(ctx context.Context, projectID string) (*model.WorkflowRun, error) {
			return nil, fmt.Errorf("%w: missing active agents for roles: reviewer", orchestrator.ErrWorkflowPrecheckFailed)
		},
	}

	h := NewOrchestratorHandler(engine)
	body := `{"project_id":"proj-123"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/orchestrator/runs", strings.NewReader(body))

	h.StartRun(w, r)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestOrchestratorHandler_GetRun_Success(t *testing.T) {
	now := time.Now()
	engine := &mockWorkflowEngine{
		getRunFn: func(ctx context.Context, id string) (*model.WorkflowRun, error) {
			if id != "run-001" {
				t.Errorf("expected id 'run-001', got %q", id)
			}
			return &model.WorkflowRun{
				ID:        "run-001",
				ProjectID: "proj-123",
				Status:    model.WorkflowRunCompleted,
				CreatedAt: now,
			}, nil
		},
	}

	h := NewOrchestratorHandler(engine)

	// Use Go 1.22+ path pattern matching
	mux := http.NewServeMux()
	h.Register(mux)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/orchestrator/runs/run-001", nil)

	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOrchestratorHandler_GetRun_NotFound(t *testing.T) {
	engine := &mockWorkflowEngine{
		getRunFn: func(ctx context.Context, id string) (*model.WorkflowRun, error) {
			return nil, nil
		},
	}

	h := NewOrchestratorHandler(engine)
	mux := http.NewServeMux()
	h.Register(mux)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/orchestrator/runs/nonexistent", nil)

	mux.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOrchestratorHandler_ListRuns_Success(t *testing.T) {
	now := time.Now()
	engine := &mockWorkflowEngine{
		listRunsFn: func(ctx context.Context, params repository.WorkflowRunListParams) ([]*model.WorkflowRun, int, error) {
			if params.Limit != 10 {
				t.Errorf("expected limit 10, got %d", params.Limit)
			}
			if params.ProjectID != "proj-123" {
				t.Errorf("expected project_id 'proj-123', got %q", params.ProjectID)
			}
			return []*model.WorkflowRun{
				{ID: "run-001", ProjectID: "proj-123", Status: model.WorkflowRunCompleted, CreatedAt: now},
				{ID: "run-002", ProjectID: "proj-123", Status: model.WorkflowRunRunning, CreatedAt: now},
			}, 2, nil
		},
	}

	h := NewOrchestratorHandler(engine)
	mux := http.NewServeMux()
	h.Register(mux)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/orchestrator/runs?limit=10&project_id=proj-123", nil)

	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["success"] != true {
		t.Errorf("expected success=true")
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data object, got %T", resp["data"])
	}
	items, ok := data["items"].([]interface{})
	if !ok {
		t.Fatalf("expected items array, got %T", data["items"])
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
	total, ok := data["total"].(float64)
	if !ok || total != 2 {
		t.Errorf("expected total=2, got %v", data["total"])
	}
}

func TestOrchestratorHandler_ListRuns_DefaultLimit(t *testing.T) {
	engine := &mockWorkflowEngine{
		listRunsFn: func(ctx context.Context, params repository.WorkflowRunListParams) ([]*model.WorkflowRun, int, error) {
			if params.Limit != 20 {
				t.Errorf("expected default limit 20, got %d", params.Limit)
			}
			return nil, 0, nil
		},
	}

	h := NewOrchestratorHandler(engine)
	mux := http.NewServeMux()
	h.Register(mux)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/orchestrator/runs", nil)

	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestOrchestratorHandler_ListRuns_Error(t *testing.T) {
	engine := &mockWorkflowEngine{
		listRunsFn: func(ctx context.Context, params repository.WorkflowRunListParams) ([]*model.WorkflowRun, int, error) {
			return nil, 0, fmt.Errorf("db error")
		},
	}

	h := NewOrchestratorHandler(engine)
	mux := http.NewServeMux()
	h.Register(mux)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/orchestrator/runs", nil)

	mux.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestOrchestratorHandler_ResumeRun_Success(t *testing.T) {
	engine := &mockWorkflowEngine{
		resumeRunFn: func(ctx context.Context, id string) error {
			if id != "run-001" {
				t.Errorf("expected id 'run-001', got %q", id)
			}
			return nil
		},
	}

	h := NewOrchestratorHandler(engine)
	mux := http.NewServeMux()
	h.Register(mux)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/orchestrator/runs/run-001/resume", nil)
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOrchestratorHandler_ResumeRun_Error(t *testing.T) {
	engine := &mockWorkflowEngine{
		resumeRunFn: func(ctx context.Context, id string) error {
			return fmt.Errorf("run is not resumable")
		},
	}

	h := NewOrchestratorHandler(engine)
	mux := http.NewServeMux()
	h.Register(mux)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/orchestrator/runs/run-001/resume", nil)
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOrchestratorHandler_ResolveManualIntervention_Success(t *testing.T) {
	engine := &mockWorkflowEngine{
		resolveManualInterventionFn: func(ctx context.Context, id string, action model.ManualInterventionAction, note string) error {
			if id != "run-001" {
				t.Errorf("expected id 'run-001', got %q", id)
			}
			if action != model.ManualInterventionActionEscalateToApproval {
				t.Errorf("expected action %q, got %q", model.ManualInterventionActionEscalateToApproval, action)
			}
			if note != "人工确认当前交付可进入审批" {
				t.Errorf("unexpected note %q", note)
			}
			return nil
		},
	}

	h := NewOrchestratorHandler(engine)
	mux := http.NewServeMux()
	h.Register(mux)

	w := httptest.NewRecorder()
	body := `{"action":"escalate_to_approval","note":"人工确认当前交付可进入审批"}`
	r := httptest.NewRequest("POST", "/api/v1/orchestrator/runs/run-001/manual-intervention", strings.NewReader(body))
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOrchestratorHandler_ResolveManualIntervention_InvalidBody(t *testing.T) {
	h := NewOrchestratorHandler(&mockWorkflowEngine{})
	mux := http.NewServeMux()
	h.Register(mux)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/orchestrator/runs/run-001/manual-intervention", strings.NewReader("not-json"))
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOrchestratorHandler_ResolveManualIntervention_Error(t *testing.T) {
	engine := &mockWorkflowEngine{
		resolveManualInterventionFn: func(ctx context.Context, id string, action model.ManualInterventionAction, note string) error {
			return fmt.Errorf("action not allowed")
		},
	}

	h := NewOrchestratorHandler(engine)
	mux := http.NewServeMux()
	h.Register(mux)

	w := httptest.NewRecorder()
	body := `{"action":"escalate_to_approval","note":"人工确认当前交付可进入审批"}`
	r := httptest.NewRequest("POST", "/api/v1/orchestrator/runs/run-001/manual-intervention", strings.NewReader(body))
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOrchestratorHandler_CancelRun_Success(t *testing.T) {
	engine := &mockWorkflowEngine{
		cancelRunFn: func(ctx context.Context, id string) error {
			if id != "run-001" {
				t.Errorf("expected id 'run-001', got %q", id)
			}
			return nil
		},
	}

	h := NewOrchestratorHandler(engine)
	mux := http.NewServeMux()
	h.Register(mux)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/orchestrator/runs/run-001/cancel", nil)
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["success"] != true {
		t.Errorf("expected success=true")
	}
}

func TestOrchestratorHandler_CancelRun_Error(t *testing.T) {
	engine := &mockWorkflowEngine{
		cancelRunFn: func(ctx context.Context, id string) error {
			return fmt.Errorf("run is not running")
		},
	}

	h := NewOrchestratorHandler(engine)
	mux := http.NewServeMux()
	h.Register(mux)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/orchestrator/runs/run-001/cancel", nil)
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

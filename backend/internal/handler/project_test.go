package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lingchou/lingchoubot/backend/internal/middleware"
	"github.com/lingchou/lingchoubot/backend/internal/service"
)

// newProjectHandler creates a handler with real service that has nil repo.
// Tests that fail at service validation (before repo call) will work.
// Tests that need DB access will panic — use only for validation tests.
func newProjectHandler() *ProjectHandler {
	svc := service.NewProjectService(nil, nil)
	return NewProjectHandler(svc)
}

func TestProjectCreate_InvalidJSON(t *testing.T) {
	h := &ProjectHandler{svc: nil} // nil svc ok — fails before service call
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/projects", strings.NewReader("not json"))
	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	var resp middleware.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Success {
		t.Error("expected success=false")
	}
	if resp.Error == nil || resp.Error.Code != "INVALID_BODY" {
		t.Errorf("expected INVALID_BODY error code, got %v", resp.Error)
	}
}

func TestProjectCreate_EmptyBody(t *testing.T) {
	h := &ProjectHandler{svc: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/projects", strings.NewReader(""))
	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestProjectCreate_EmptyName(t *testing.T) {
	h := newProjectHandler()
	w := httptest.NewRecorder()
	body := `{"name": ""}`
	r := httptest.NewRequest("POST", "/api/v1/projects", strings.NewReader(body))
	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	var resp middleware.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Success {
		t.Error("expected success=false for empty name")
	}
	if resp.Error == nil || resp.Error.Code != "CREATE_FAILED" {
		t.Errorf("expected CREATE_FAILED error code, got %v", resp.Error)
	}
}

func TestProjectUpdate_InvalidJSON(t *testing.T) {
	h := &ProjectHandler{svc: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/api/v1/projects/abc", strings.NewReader("{bad"))
	h.Update(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	var resp middleware.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error == nil || resp.Error.Code != "INVALID_BODY" {
		t.Errorf("expected INVALID_BODY, got %v", resp.Error)
	}
}

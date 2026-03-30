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

func newTaskHandler() *TaskHandler {
	svc := service.NewTaskService(nil, nil)
	return NewTaskHandler(svc)
}

func TestTaskCreate_InvalidJSON(t *testing.T) {
	h := &TaskHandler{svc: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/tasks", strings.NewReader("not-json"))
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
		t.Errorf("expected INVALID_BODY, got %v", resp.Error)
	}
}

func TestTaskCreate_EmptyBody(t *testing.T) {
	h := &TaskHandler{svc: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/tasks", strings.NewReader(""))
	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestTaskCreate_MissingProjectID(t *testing.T) {
	h := newTaskHandler()
	w := httptest.NewRecorder()
	body := `{"title": "some task"}`
	r := httptest.NewRequest("POST", "/api/v1/tasks", strings.NewReader(body))
	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	var resp middleware.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error == nil || resp.Error.Code != "CREATE_FAILED" {
		t.Errorf("expected CREATE_FAILED, got %v", resp.Error)
	}
}

func TestTaskCreate_MissingTitle(t *testing.T) {
	h := newTaskHandler()
	w := httptest.NewRecorder()
	body := `{"project_id": "proj-123"}`
	r := httptest.NewRequest("POST", "/api/v1/tasks", strings.NewReader(body))
	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	var resp middleware.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error == nil || resp.Error.Code != "CREATE_FAILED" {
		t.Errorf("expected CREATE_FAILED, got %v", resp.Error)
	}
}

func TestTaskUpdate_InvalidJSON(t *testing.T) {
	h := &TaskHandler{svc: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/api/v1/tasks/task-1", strings.NewReader("{{invalid"))
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

func TestTaskTransitionStatus_InvalidJSON(t *testing.T) {
	h := &TaskHandler{svc: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PATCH", "/api/v1/tasks/task-1/status", strings.NewReader("bad"))
	h.TransitionStatus(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	var resp middleware.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error == nil || resp.Error.Code != "INVALID_BODY" {
		t.Errorf("expected INVALID_BODY, got %v", resp.Error)
	}
}

func TestTaskTransitionStatus_EmptyBody(t *testing.T) {
	h := &TaskHandler{svc: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PATCH", "/api/v1/tasks/task-1/status", strings.NewReader(""))
	h.TransitionStatus(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

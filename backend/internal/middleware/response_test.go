package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSON_SuccessResponse(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"key": "value"}

	JSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var resp APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.Error != nil {
		t.Error("expected error=nil for success response")
	}
	if resp.Data == nil {
		t.Error("expected data to be non-nil")
	}
}

func TestJSON_CreatedStatus(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusCreated, map[string]string{"id": "123"})

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}
	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Success {
		t.Error("expected success=true for 201")
	}
}

func TestJSON_NilData(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusOK, nil)

	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Success {
		t.Error("expected success=true even with nil data")
	}
}

func TestErrorJSON_ErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()
	ErrorJSON(w, http.StatusBadRequest, "INVALID_BODY", "请求体解析失败")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var resp APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Success {
		t.Error("expected success=false")
	}
	if resp.Data != nil {
		t.Error("expected data=nil for error response")
	}
	if resp.Error == nil {
		t.Fatal("expected error to be non-nil")
	}
	if resp.Error.Code != "INVALID_BODY" {
		t.Errorf("expected error code INVALID_BODY, got %s", resp.Error.Code)
	}
	if resp.Error.Message != "请求体解析失败" {
		t.Errorf("unexpected error message: %s", resp.Error.Message)
	}
}

func TestErrorJSON_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	ErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "resource not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Success {
		t.Error("expected success=false for 404")
	}
	if resp.Error.Code != "NOT_FOUND" {
		t.Errorf("expected NOT_FOUND, got %s", resp.Error.Code)
	}
}

func TestErrorJSON_InternalServerError(t *testing.T) {
	w := httptest.NewRecorder()
	ErrorJSON(w, http.StatusInternalServerError, "QUERY_FAILED", "database error")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Success {
		t.Error("expected success=false for 500")
	}
}

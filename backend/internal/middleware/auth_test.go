package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuth_PublicPaths(t *testing.T) {
	handler := Auth("secret-key")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	paths := []string{"/healthz", "/readyz", "/api/v1/ping"}
	for _, p := range paths {
		req := httptest.NewRequest("GET", p, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("public path %s returned %d, want 200", p, w.Code)
		}
	}
}

func TestAuth_MissingToken(t *testing.T) {
	handler := Auth("secret-key")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/projects", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing token returned %d, want 401", w.Code)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	handler := Auth("secret-key")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/projects", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("invalid token returned %d, want 401", w.Code)
	}
}

func TestAuth_ValidToken(t *testing.T) {
	called := false
	handler := Auth("secret-key")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		identity := IdentityFromContext(r.Context())
		if identity == nil {
			t.Error("identity not set in context")
			return
		}
		if identity.SubjectType != "user" {
			t.Errorf("identity.SubjectType = %q, want 'user'", identity.SubjectType)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/projects", nil)
	req.Header.Set("Authorization", "Bearer secret-key")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("valid token returned %d, want 200", w.Code)
	}
	if !called {
		t.Error("handler not called with valid token")
	}
}

func TestAuth_EmptyApiKey_DevMode(t *testing.T) {
	called := false
	handler := Auth("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/projects", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("dev mode returned %d, want 200", w.Code)
	}
	if !called {
		t.Error("handler not called in dev mode")
	}
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		{"Bearer abc123", "abc123"},
		{"bearer abc123", "abc123"},
		{"BEARER abc123", "abc123"},
		{"Basic abc123", ""},
		{"abc123", ""},
		{"", ""},
	}
	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/", nil)
		if tt.header != "" {
			req.Header.Set("Authorization", tt.header)
		}
		got := extractBearerToken(req)
		if got != tt.want {
			t.Errorf("extractBearerToken(%q) = %q, want %q", tt.header, got, tt.want)
		}
	}
}

func TestIsPublicPath(t *testing.T) {
	tests := []struct {
		path   string
		public bool
	}{
		{"/healthz", true},
		{"/readyz", true},
		{"/api/v1/ping", true},
		{"/api/v1/projects", false},
		{"/api/v1/tasks", false},
		{"/", false},
	}
	for _, tt := range tests {
		got := isPublicPath(tt.path)
		if got != tt.public {
			t.Errorf("isPublicPath(%q) = %v, want %v", tt.path, got, tt.public)
		}
	}
}

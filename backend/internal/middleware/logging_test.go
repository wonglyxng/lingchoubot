package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"io"
)

func TestLoggingWrappedWriterPreservesFlusher(t *testing.T) {
	handler := Logging(slog.New(slog.NewTextHandler(io.Discard, nil)))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, ok := w.(http.Flusher); !ok {
			t.Fatal("wrapped writer does not implement http.Flusher")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/stream", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", resp.Code)
	}
}

func TestLoggingWrappedWriterSupportsResponseControllerUnwrap(t *testing.T) {
	wrapped := &wrappedWriter{ResponseWriter: httptest.NewRecorder(), statusCode: http.StatusOK}
	if wrapped.Unwrap() == nil {
		t.Fatal("expected wrapped writer to expose underlying response writer")
	}
}

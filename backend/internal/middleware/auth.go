package middleware

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const (
	// ContextKeyIdentity is the key for the authenticated identity in request context.
	ContextKeyIdentity contextKey = "auth_identity"
)

// Identity represents an authenticated caller.
type Identity struct {
	Subject     string // who: user ID, agent ID, or "system"
	SubjectType string // "user", "agent", or "service"
	Token       string // the raw token (for audit trail)
}

// IdentityFromContext extracts the authenticated identity from the request context.
func IdentityFromContext(ctx context.Context) *Identity {
	v, _ := ctx.Value(ContextKeyIdentity).(*Identity)
	return v
}

// Auth returns middleware that validates API key authentication.
// Public paths (healthz, readyz, ping) are exempt.
// The apiKey parameter is the expected static token for MVP.
func Auth(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// No API key configured: pass through (development mode)
			if apiKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			token := extractBearerToken(r)
			if token == "" {
				ErrorJSON(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid Authorization header")
				return
			}

			if token != apiKey {
				ErrorJSON(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid API key")
				return
			}

			identity := &Identity{
				Subject:     "api-client",
				SubjectType: "user",
				Token:       token,
			}
			ctx := context.WithValue(r.Context(), ContextKeyIdentity, identity)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func isPublicPath(path string) bool {
	switch path {
	case "/healthz", "/readyz", "/api/v1/ping":
		return true
	}
	return false
}

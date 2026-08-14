package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/oat431/backend-challenge/internal/application"
)

// NewRouter wires all routes on a single ServeMux using Go 1.22+ method and
// path patterns. Protected routes are wrapped by the JWT middleware; the
// logging middleware wraps everything.
func NewRouter(log *slog.Logger, users *application.UserService, auth *application.AuthService) http.Handler {
	ah := NewAuthHandler(users, auth)
	uh := NewUserHandler(users)

	mux := http.NewServeMux()

	// Public
	mux.HandleFunc("POST /api/v1/auth/register", ah.Register)
	mux.HandleFunc("POST /api/v1/auth/login", ah.Login)
	mux.HandleFunc("GET /healthz", Health)

	// Protected (JWT)
	mux.Handle("POST /api/v1/users", requireAuth(auth, http.HandlerFunc(uh.Create)))
	mux.Handle("GET /api/v1/users", requireAuth(auth, http.HandlerFunc(uh.List)))
	mux.Handle("GET /api/v1/users/{id}", requireAuth(auth, http.HandlerFunc(uh.Get)))
	mux.Handle("PUT /api/v1/users/{id}", requireAuth(auth, http.HandlerFunc(uh.Update)))
	mux.Handle("DELETE /api/v1/users/{id}", requireAuth(auth, http.HandlerFunc(uh.Delete)))

	return logRequest(log, mux)
}

// requireAuth validates the Bearer token and stores claims in the request
// context. Failure is a uniform 401 UNAUTHORIZED (AC-002b, AC-004d, ...).
func requireAuth(auth *application.AuthService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorEnvelope{Error: errorBody{
				Code:    "UNAUTHORIZED",
				Message: "Missing or invalid Authorization header",
			}})
			return
		}

		claims, err := auth.VerifyToken(token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, errorEnvelope{Error: errorBody{
				Code:    "UNAUTHORIZED",
				Message: "Invalid or expired token",
			}})
			return
		}

		ctx := context.WithValue(r.Context(), claimsKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type claimsKey struct{}

// ClaimsFrom returns the authenticated claims, or false when absent.
func ClaimsFrom(ctx context.Context) (application.TokenClaims, bool) {
	c, ok := ctx.Value(claimsKey{}).(application.TokenClaims)
	return c, ok
}

func bearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return "", false
	}
	return strings.TrimSpace(h[len(prefix):]), true
}

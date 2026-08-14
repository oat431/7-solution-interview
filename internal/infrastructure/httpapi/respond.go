// Package httpapi is the REST driving adapter: routing, middleware, DTOs and
// JSON encoding. All business logic lives in the application layer.
package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/oat431/7-solution-interview/internal/domain"
)

const maxBodyBytes = 1 << 20 // 1 MB

// timeNow and timeSinceMS are indirection points that make middleware
// duration assertions deterministic in tests.
var timeNow = time.Now

func timeSinceMS(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}

// writeJSON encodes v with the given status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// errorEnvelope matches the contract in 022 §3.
type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Details []domain.FieldError `json:"details,omitempty"`
}

// writeError maps domain/application errors to the API error contract.
func writeError(w http.ResponseWriter, err error) {
	status, code, msg, details := mapError(err)
	writeJSON(w, status, errorEnvelope{Error: errorBody{
		Code:    code,
		Message: msg,
		Details: details,
	}})
}

func mapError(err error) (status int, code, msg string, details []domain.FieldError) {
	var verr domain.ValidationError
	switch {
	case errors.As(err, &verr):
		return http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", verr
	case errors.Is(err, domain.ErrInvalidCredentials):
		return http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password", nil
	case errors.Is(err, domain.ErrEmailExists):
		return http.StatusConflict, "EMAIL_ALREADY_EXISTS", "A user with this email already exists", nil
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "USER_NOT_FOUND", "User not found", nil
	case errors.Is(err, domain.ErrInvalidID):
		return http.StatusBadRequest, "INVALID_ID", "User id must be a valid ObjectID", nil
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil
	}
}

// decodeJSON reads a JSON body, enforcing size limit and rejecting unknown
// fields. On failure it writes the error response and returns false.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope{Error: errorBody{
			Code:    "VALIDATION_ERROR",
			Message: "Malformed request body",
		}})
		return false
	}
	return true
}

// logRequest is the logging middleware (challenge requirement 5): method,
// path, status and execution time on every request, via structured slog.
func logRequest(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := timeNow()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration_ms", timeSinceMS(start),
		)
	})
}

// statusWriter captures the response status for logging.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

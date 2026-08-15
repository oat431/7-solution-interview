package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

// errorCode probes the central error handler with a route that returns the
// given error and decodes the envelope code + status.
func errorCode(t *testing.T, err error) (int, string) {
	t.Helper()

	app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
	app.Get("/probe", func(c fiber.Ctx) error { return err })

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/probe", nil), fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode envelope: %v (%s)", err, raw)
	}
	return resp.StatusCode, body.Error.Code
}

// TestErrorHandlerMapsFiber413 covers DEF-001 (PO decision: Option A —
// REQUEST_TOO_LARGE instead of INTERNAL_ERROR).
func TestErrorHandlerMapsFiber413(t *testing.T) {
	status, code := errorCode(t, fiber.ErrRequestEntityTooLarge)
	if status != fiber.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", status)
	}
	if code != "REQUEST_TOO_LARGE" {
		t.Fatalf("expected REQUEST_TOO_LARGE, got %q", code)
	}
}

// TestErrorHandlerMapsFiber404 covers DEF-002 (PO decision: NOT_FOUND is a
// documented contract code for unknown routes).
func TestErrorHandlerMapsFiber404(t *testing.T) {
	status, code := errorCode(t, fiber.ErrNotFound)
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", status)
	}
	if code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND, got %q", code)
	}
}

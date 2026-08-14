package httpapi

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/oat431/7-solution-interview/internal/application"
)

// logRequest logs method, path, status and duration for every request.
func logRequest(log *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		attrs := []any{
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"duration_ms", time.Since(start).Milliseconds(),
		}
		if rid := c.GetRespHeader(fiber.HeaderXRequestID); rid != "" {
			attrs = append(attrs, "request_id", rid)
		}
		log.Info("http request", attrs...)
		return err
	}
}

// requireAuth validates the Bearer token before passing the request on.
func requireAuth(auth *application.AuthService) fiber.Handler {
	return func(c fiber.Ctx) error {
		token, ok := bearerToken(c.Get(fiber.HeaderAuthorization))
		if !ok {
			return unauthorized("Missing or invalid Authorization header")
		}
		if _, err := auth.VerifyToken(token); err != nil {
			return unauthorized("Invalid or expired token")
		}
		return c.Next()
	}
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	return strings.TrimSpace(header[len(prefix):]), true
}

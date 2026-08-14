package httpapi

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	"github.com/oat431/7-solution-interview/internal/application"
)

const maxBodyBytes = 1 << 20 // 1 MB

// NewApp wires all routes on one Fiber app; protected routes get the JWT
// middleware, everything gets request-id, logging and panic recovery.
func NewApp(log *slog.Logger, users *application.UserService, auth *application.AuthService) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "7-solution-interview-api",
		Immutable:    true,
		BodyLimit:    maxBodyBytes,
		ErrorHandler: errorHandler,
	})

	// Middleware order: RequestID → Logger → Recover → routes.
	app.Use(requestid.New())
	app.Use(logRequest(log))
	app.Use(recover.New())

	ah := NewAuthHandler(users, auth)
	uh := NewUserHandler(users)

	v1 := app.Group("/api/v1")
	v1.Post("/auth/register", ah.Register).Name("register")
	v1.Post("/auth/login", ah.Login).Name("login")

	usersGroup := v1.Group("/users")
	usersGroup.Use(requireAuth(auth))
	usersGroup.Post("", uh.Create).Name("create-user")
	usersGroup.Get("", uh.List).Name("list-users")
	usersGroup.Get("/:id", uh.Get).Name("get-user")
	usersGroup.Put("/:id", uh.Update).Name("update-user")
	usersGroup.Delete("/:id", uh.Delete).Name("delete-user")

	app.Get("/healthz", Health).Name("health")

	// Explicit 405 for paths that only exist under specific methods.
	methodNotAllowed := func(c fiber.Ctx) error {
		return &apiError{status: fiber.StatusMethodNotAllowed, code: "METHOD_NOT_ALLOWED", message: "Method not allowed"}
	}
	app.All("/healthz", methodNotAllowed)
	app.All("/api/v1/auth/register", methodNotAllowed)
	app.All("/api/v1/auth/login", methodNotAllowed)
	app.All("/api/v1/users", methodNotAllowed)
	app.All("/api/v1/users/:id", methodNotAllowed)

	// Catch-all 404 with the API error envelope.
	app.Use(func(c fiber.Ctx) error {
		return &apiError{status: fiber.StatusNotFound, code: "NOT_FOUND", message: "Route not found"}
	})

	return app
}

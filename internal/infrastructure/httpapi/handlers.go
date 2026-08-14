package httpapi

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/oat431/7-solution-interview/internal/application"
	"github.com/oat431/7-solution-interview/internal/domain"
)

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string      `json:"token"`
	TokenType string      `json:"tokenType"`
	ExpiresIn int64       `json:"expiresIn"`
	User      domain.User `json:"user"`
}

type updateRequest struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
}

type listResponse struct {
	Data []domain.User `json:"data"`
	Meta listMeta      `json:"meta"`
}

type listMeta struct {
	Count int `json:"count"`
}

type healthResponse struct {
	Status string `json:"status"`
}

type AuthHandler struct {
	users *application.UserService
	auth  *application.AuthService
}

func NewAuthHandler(users *application.UserService, auth *application.AuthService) *AuthHandler {
	return &AuthHandler{users: users, auth: auth}
}

// Register creates a user and returns 201 without a token.
func (h *AuthHandler) Register(c fiber.Ctx) error {
	return createUser(c, h.users)
}

// Login exchanges credentials for a JWT.
func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req loginRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}

	res, err := h.auth.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		return err
	}
	return c.JSON(loginResponse{
		Token:     res.Token,
		TokenType: "Bearer",
		ExpiresIn: int64(res.ExpiresIn / time.Second),
		User:      res.User,
	})
}

type UserHandler struct {
	users *application.UserService
}

func NewUserHandler(users *application.UserService) *UserHandler {
	return &UserHandler{users: users}
}

// Create mirrors register behind JWT auth (A7: one use case, two routes).
func (h *UserHandler) Create(c fiber.Ctx) error {
	return createUser(c, h.users)
}

func (h *UserHandler) Get(c fiber.Ctx) error {
	user, err := h.users.Get(c.Context(), c.Params("id"))
	if err != nil {
		return err
	}
	return c.JSON(user)
}

func (h *UserHandler) List(c fiber.Ctx) error {
	users, err := h.users.List(c.Context())
	if err != nil {
		return err
	}
	return c.JSON(listResponse{Data: users, Meta: listMeta{Count: len(users)}})
}

func (h *UserHandler) Update(c fiber.Ctx) error {
	var req updateRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}

	user, err := h.users.Update(c.Context(), c.Params("id"), application.UpdateUserInput{
		Name:  req.Name,
		Email: req.Email,
	})
	if err != nil {
		return err
	}
	return c.JSON(user)
}

func (h *UserHandler) Delete(c fiber.Ctx) error {
	if err := h.users.Delete(c.Context(), c.Params("id")); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// Health serves the liveness probe used by docker healthchecks.
func Health(c fiber.Ctx) error {
	return c.JSON(healthResponse{Status: "ok"})
}

// createUser is the shared path behind POST /auth/register and POST /users.
func createUser(c fiber.Ctx, users *application.UserService) error {
	var req registerRequest
	if err := decodeJSON(c, &req); err != nil {
		return err
	}

	user, err := users.Create(c.Context(), domain.NewUserInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(user)
}

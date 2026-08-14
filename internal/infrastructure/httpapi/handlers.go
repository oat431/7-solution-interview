package httpapi

import (
	"net/http"
	"time"

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
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	createUser(w, r, h.users)
}

// Login exchanges credentials for a JWT.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	res, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, loginResponse{
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
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	createUser(w, r, h.users)
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	user, err := h.users.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.users.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listResponse{Data: users, Meta: listMeta{Count: len(users)}})
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req updateRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	user, err := h.users.Update(r.Context(), r.PathValue("id"), application.UpdateUserInput{
		Name:  req.Name,
		Email: req.Email,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.users.Delete(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Health serves the liveness probe used by docker healthchecks.
func Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

// createUser is the shared path behind POST /auth/register and POST /users.
func createUser(w http.ResponseWriter, r *http.Request, users *application.UserService) {
	var req registerRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	user, err := users.Create(r.Context(), domain.NewUserInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

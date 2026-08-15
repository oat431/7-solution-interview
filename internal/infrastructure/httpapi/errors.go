package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/oat431/7-solution-interview/internal/domain"
)

type errorBody struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Details []domain.FieldError `json:"details,omitempty"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

// apiError carries the full error contract (022 §3). Handlers return it and
// errorHandler renders it as the API error envelope.
type apiError struct {
	status  int
	code    string
	message string
	details []domain.FieldError
}

func (e *apiError) Error() string { return e.message }

func badRequest(message string) *apiError {
	return &apiError{status: fiber.StatusBadRequest, code: "VALIDATION_ERROR", message: message}
}

func unauthorized(message string) *apiError {
	return &apiError{status: fiber.StatusUnauthorized, code: "UNAUTHORIZED", message: message}
}

// errorHandler is the central Fiber error handler: every error a handler
// returns lands here and leaves as the API error envelope.
func errorHandler(c fiber.Ctx, err error) error {
	var ae *apiError
	if errors.As(err, &ae) {
		return writeEnvelope(c, ae.status, ae.code, ae.message, ae.details)
	}

	var fe *fiber.Error
	if errors.As(err, &fe) {
		code := "INTERNAL_ERROR"
		msg := fe.Message
		switch fe.Code {
		case fiber.StatusNotFound:
			code = "NOT_FOUND"
		case fiber.StatusMethodNotAllowed:
			code = "METHOD_NOT_ALLOWED"
		case fiber.StatusBadRequest:
			code = "VALIDATION_ERROR"
		case fiber.StatusRequestEntityTooLarge:
			code = "REQUEST_TOO_LARGE"
			msg = "Request body exceeds the size limit"
		}
		return writeEnvelope(c, fe.Code, code, msg, nil)
	}

	status, code, msg, details := mapError(err)
	return writeEnvelope(c, status, code, msg, details)
}

func writeEnvelope(c fiber.Ctx, status int, code, msg string, details []domain.FieldError) error {
	return c.Status(status).JSON(errorEnvelope{Error: errorBody{
		Code:    code,
		Message: msg,
		Details: details,
	}})
}

func mapError(err error) (status int, code, msg string, details []domain.FieldError) {
	var verr domain.ValidationError
	switch {
	case errors.As(err, &verr):
		return fiber.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", verr
	case errors.Is(err, domain.ErrInvalidCredentials):
		return fiber.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password", nil
	case errors.Is(err, domain.ErrEmailExists):
		return fiber.StatusConflict, "EMAIL_ALREADY_EXISTS", "A user with this email already exists", nil
	case errors.Is(err, domain.ErrNotFound):
		return fiber.StatusNotFound, "USER_NOT_FOUND", "User not found", nil
	case errors.Is(err, domain.ErrInvalidID):
		return fiber.StatusBadRequest, "INVALID_ID", "User id must be a valid ObjectID", nil
	default:
		return fiber.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil
	}
}

// decodeJSON parses the body with unknown fields rejected — the password
// field cannot be smuggled through an update (AC-006g).
func decodeJSON(c fiber.Ctx, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(c.Body()))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return badRequest("Malformed request body")
	}
	return nil
}

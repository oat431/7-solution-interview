// Package domain contains the pure business model: the User entity and
// domain errors. It must stay free of external dependencies (stdlib only)
// and must not import application or infrastructure packages.
package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// User is the core entity. Password material never lives on it.
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

// NewUserInput carries the raw fields needed to create a user.
type NewUserInput struct {
	Name     string
	Email    string
	Password string
}

const (
	MaxNameLength     = 100
	MinPasswordLength = 8
	MaxPasswordLength = 72 // bcrypt input limit
)

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]{2,}$`)
var objectIDRe = regexp.MustCompile(`^[0-9a-fA-F]{24}$`)

// IsValidID reports whether id has the ObjectID hex shape (24 hex chars).
// The rule lives in the domain so every adapter behaves identically.
func IsValidID(id string) bool {
	return objectIDRe.MatchString(id)
}

// ValidateName checks the name rules, returning a ValidationError or nil.
func ValidateName(name string) error {
	if name == "" {
		return ValidationError{{Field: "name", Message: "is required"}}
	}
	if len(name) > MaxNameLength {
		return ValidationError{{Field: "name", Message: fmt.Sprintf("must be at most %d characters", MaxNameLength)}}
	}
	return nil
}

// NormalizeEmail returns the canonical form used for storage and lookup.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidateEmail checks the email rules, returning a ValidationError or nil.
// Callers normalize (trim + lowercase) before validating.
func ValidateEmail(email string) error {
	if email == "" {
		return ValidationError{{Field: "email", Message: "is required"}}
	}
	if !emailRe.MatchString(email) {
		return ValidationError{{Field: "email", Message: "must be a valid email address"}}
	}
	return nil
}

// Validate returns a ValidationError listing every rule violation, or nil.
func (in NewUserInput) Validate() error {
	var errs ValidationError

	in.Name = strings.TrimSpace(in.Name)
	if err := ValidateName(in.Name); err != nil {
		errs = append(errs, err.(ValidationError)...)
	}
	if err := ValidateEmail(in.Email); err != nil {
		errs = append(errs, err.(ValidationError)...)
	}
	if in.Password == "" {
		errs = append(errs, FieldError{Field: "password", Message: "is required"})
	} else if len(in.Password) < MinPasswordLength || len(in.Password) > MaxPasswordLength {
		errs = append(errs, FieldError{Field: "password", Message: fmt.Sprintf("must be between %d and %d characters", MinPasswordLength, MaxPasswordLength)})
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// ValidationError aggregates one or more field-level errors.
type ValidationError []FieldError

func (e ValidationError) Error() string {
	msgs := make([]string, 0, len(e))
	for _, fe := range e {
		msgs = append(msgs, fe.Field+": "+fe.Message)
	}
	return "validation failed: " + strings.Join(msgs, "; ")
}

// FieldError is a single field-level validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Sentinel domain errors, mapped to HTTP/gRPC status codes by the adapters.
var (
	ErrNotFound           = errors.New("not found")
	ErrEmailExists        = errors.New("email already exists")
	ErrInvalidID          = errors.New("invalid id")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

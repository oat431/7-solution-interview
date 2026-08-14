package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/oat431/7-solution-interview/internal/domain"
)

// UserService implements the user-management use cases: create, read, list,
// update, delete. It is the shared core used by both the REST and gRPC
// adapters (see ADR-04).
type UserService struct {
	repo   UserRepository
	hasher PasswordHasher
}

func NewUserService(repo UserRepository, hasher PasswordHasher) *UserService {
	return &UserService{repo: repo, hasher: hasher}
}

// Create validates input, hashes the password and persists the user.
// Emails are normalized to lowercase (unique index + login both depend on it).
func (s *UserService) Create(ctx context.Context, in domain.NewUserInput) (domain.User, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))

	if err := in.Validate(); err != nil {
		return domain.User{}, err
	}

	hash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return domain.User{}, fmt.Errorf("hash password: %w", err)
	}

	return s.repo.Create(ctx, CreateUserRecord{
		Name:         in.Name,
		Email:        in.Email,
		PasswordHash: hash,
	})
}

// Get returns a user by ID. Invalid IDs are rejected in the domain layer so
// every repository implementation behaves identically.
func (s *UserService) Get(ctx context.Context, id string) (domain.User, error) {
	if !domain.IsValidID(id) {
		return domain.User{}, domain.ErrInvalidID
	}
	return s.repo.FindByID(ctx, id)
}

// List returns all users.
func (s *UserService) List(ctx context.Context) ([]domain.User, error) {
	return s.repo.List(ctx)
}

// Update applies a partial update of name and/or email. At least one field
// must be present; the password is not mutable through this operation.
func (s *UserService) Update(ctx context.Context, id string, name, email *string) (domain.User, error) {
	if !domain.IsValidID(id) {
		return domain.User{}, domain.ErrInvalidID
	}
	if name == nil && email == nil {
		return domain.User{}, domain.ValidationError{
			{Field: "body", Message: "at least one of name or email is required"},
		}
	}

	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if err := domain.ValidateName(trimmed); err != nil {
			return domain.User{}, err
		}
		*name = trimmed
	}

	if email != nil {
		normalized := strings.ToLower(strings.TrimSpace(*email))
		if err := domain.ValidateEmail(normalized); err != nil {
			return domain.User{}, err
		}
		*email = normalized

		// Pre-check uniqueness (friendly 409); unique index remains the
		// race-proof backstop (AC-001f / AC-006e).
		existing, err := s.repo.FindByEmail(ctx, normalized)
		if err == nil && existing.ID != id {
			return domain.User{}, domain.ErrEmailExists
		}
		if err != nil && err != domain.ErrNotFound {
			return domain.User{}, err
		}
	}

	return s.repo.Update(ctx, id, name, email)
}

// Delete removes a user.
func (s *UserService) Delete(ctx context.Context, id string) error {
	if !domain.IsValidID(id) {
		return domain.ErrInvalidID
	}
	return s.repo.Delete(ctx, id)
}

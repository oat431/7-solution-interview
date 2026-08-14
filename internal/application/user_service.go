package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/oat431/7-solution-interview/internal/domain"
)

// UserService implements the user-management use cases. It is the shared
// core behind both the REST and gRPC adapters (ADR-04).
type UserService struct {
	repo   UserRepository
	hasher PasswordHasher
}

func NewUserService(repo UserRepository, hasher PasswordHasher) *UserService {
	return &UserService{repo: repo, hasher: hasher}
}

// Create normalizes the email, validates, hashes the password and persists.
func (s *UserService) Create(ctx context.Context, in domain.NewUserInput) (domain.User, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = domain.NormalizeEmail(in.Email)

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

// Get returns a user by ID.
func (s *UserService) Get(ctx context.Context, id string) (domain.User, error) {
	if !domain.IsValidID(id) {
		return domain.User{}, domain.ErrInvalidID
	}
	return s.repo.FindByID(ctx, id)
}

func (s *UserService) List(ctx context.Context) ([]domain.User, error) {
	return s.repo.List(ctx)
}

// Update applies a partial update of name/email; the password is not
// mutable here.
func (s *UserService) Update(ctx context.Context, id string, in UpdateUserInput) (domain.User, error) {
	if !domain.IsValidID(id) {
		return domain.User{}, domain.ErrInvalidID
	}
	if in.Name == nil && in.Email == nil {
		return domain.User{}, domain.ValidationError{
			{Field: "body", Message: "at least one of name or email is required"},
		}
	}
	if in.Name != nil {
		if err := s.validateName(in.Name); err != nil {
			return domain.User{}, err
		}
	}
	if in.Email != nil {
		if err := s.validateEmailAvailable(ctx, in.Email, id); err != nil {
			return domain.User{}, err
		}
	}
	return s.repo.Update(ctx, id, in)
}

func (s *UserService) Delete(ctx context.Context, id string) error {
	if !domain.IsValidID(id) {
		return domain.ErrInvalidID
	}
	return s.repo.Delete(ctx, id)
}

func (s *UserService) validateName(name *string) error {
	trimmed := strings.TrimSpace(*name)
	if err := domain.ValidateName(trimmed); err != nil {
		return err
	}
	*name = trimmed
	return nil
}

func (s *UserService) validateEmailAvailable(ctx context.Context, email *string, id string) error {
	normalized := domain.NormalizeEmail(*email)
	if err := domain.ValidateEmail(normalized); err != nil {
		return err
	}
	*email = normalized

	// Pre-check uniqueness (friendly 409); unique index remains the
	// race-proof backstop (AC-001f / AC-006e).
	existing, err := s.repo.FindByEmail(ctx, normalized)
	if err == nil && existing.ID != id {
		return domain.ErrEmailExists
	}
	if err != nil && err != domain.ErrNotFound {
		return err
	}
	return nil
}

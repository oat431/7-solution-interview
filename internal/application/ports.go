// Package application contains the use cases of the service. It depends only
// on the domain package and on the ports (interfaces) declared below — never
// on infrastructure. Adapters implement the ports.
package application

import (
	"context"
	"time"

	"github.com/oat431/7-solution-interview/internal/domain"
)

// CreateUserRecord is the persistence input for creating a user.
type CreateUserRecord struct {
	Name         string
	Email        string
	PasswordHash string
}

// StoredUser is a user with its password hash, returned only when
// credentials must be checked.
type StoredUser struct {
	domain.User
	PasswordHash string
}

// UpdateUserInput is a partial update: nil fields are left unchanged.
type UpdateUserInput struct {
	Name  *string
	Email *string
}

// UserRepository is the driven port for user persistence, implemented by
// the MongoDB adapter and by test fakes.
type UserRepository interface {
	Create(ctx context.Context, rec CreateUserRecord) (domain.User, error)
	FindByID(ctx context.Context, id string) (domain.User, error)
	FindByEmail(ctx context.Context, email string) (StoredUser, error)
	List(ctx context.Context) ([]domain.User, error)
	Update(ctx context.Context, id string, in UpdateUserInput) (domain.User, error)
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
}

// PasswordHasher is the driven port for password hashing (bcrypt adapter).
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type TokenClaims struct {
	Subject string // user ID
	Email   string
}

// TokenManager is the driven port for JWT issuing/verification.
type TokenManager interface {
	Issue(ctx context.Context, claims TokenClaims) (token string, expiresIn time.Duration, err error)
	Verify(token string) (TokenClaims, error)
}

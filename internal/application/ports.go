// Package application contains the use cases of the service. It depends only
// on the domain package and on the ports (interfaces) declared below — never
// on infrastructure. Adapters implement the ports.
package application

import (
	"context"
	"time"

	"github.com/oat431/backend-challenge/internal/domain"
)

// CreateUserRecord is the persistence input for creating a user. The domain
// entity deliberately carries no password material; the hash travels only
// between the application layer and the persistence adapter.
type CreateUserRecord struct {
	Name         string
	Email        string
	PasswordHash string
}

// StoredUser is a user plus its password hash, as returned by the repository
// when credentials need to be checked (login). The hash never reaches API
// responses — adapters strip it.
type StoredUser struct {
	domain.User
	PasswordHash string
}

// UserRepository is the driven port for user persistence (hexagonal).
// Implemented by the MongoDB adapter and by test fakes.
type UserRepository interface {
	Create(ctx context.Context, rec CreateUserRecord) (domain.User, error)
	FindByID(ctx context.Context, id string) (domain.User, error)
	FindByEmail(ctx context.Context, email string) (StoredUser, error)
	List(ctx context.Context) ([]domain.User, error)
	Update(ctx context.Context, id string, name, email *string) (domain.User, error)
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
}

// PasswordHasher is the driven port for password hashing (bcrypt adapter).
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

// TokenClaims is what token verification yields.
type TokenClaims struct {
	Subject string // user ID
	Email   string
}

// TokenManager is the driven port for JWT issuing/verification.
type TokenManager interface {
	Issue(ctx context.Context, subject, email string, ttl time.Duration) (string, error)
	Verify(token string) (TokenClaims, error)
}

package application

import (
	"context"
	"time"

	"github.com/oat431/7-solution-interview/internal/domain"
)

// LoginResult is a successful login: the user plus a fresh JWT.
type LoginResult struct {
	User      domain.User
	Token     string
	ExpiresIn time.Duration
}

// AuthService handles login and token verification. Registration lives in
// UserService.Create, shared by the public and protected routes (A7).
type AuthService struct {
	repo   UserRepository
	hasher PasswordHasher
	tokens TokenManager
}

func NewAuthService(repo UserRepository, hasher PasswordHasher, tokens TokenManager) *AuthService {
	return &AuthService{repo: repo, hasher: hasher, tokens: tokens}
}

// dummyPasswordHash is a real bcrypt hash (cost 10) of a throwaway string.
// When the email is unknown we compare against it so the unknown-email and
// wrong-password paths take comparable time — otherwise login doubles as an
// email-enumeration timing oracle (ACT-S2).
const dummyPasswordHash = "$2a$10$ReM7nZzKQqg7h3/GXbPKJetusfQauRkpfBLV4n0VqzXV5gaCzeTHW"

// Login returns the same error for wrong email and wrong password — no user
// enumeration (AC-002c/d).
func (s *AuthService) Login(ctx context.Context, email, password string) (LoginResult, error) {
	email = domain.NormalizeEmail(email)

	stored, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if err != domain.ErrNotFound {
			return LoginResult{}, err
		}
		// Flatten timing: burn one bcrypt compare before failing.
		_ = s.hasher.Compare(dummyPasswordHash, password)
		return LoginResult{}, domain.ErrInvalidCredentials
	}

	if err := s.hasher.Compare(stored.PasswordHash, password); err != nil {
		return LoginResult{}, domain.ErrInvalidCredentials
	}

	token, expiresIn, err := s.tokens.Issue(ctx, TokenClaims{Subject: stored.ID, Email: stored.Email})
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{User: stored.User, Token: token, ExpiresIn: expiresIn}, nil
}

// VerifyToken is shared by the REST middleware and the gRPC interceptor.
func (s *AuthService) VerifyToken(token string) (TokenClaims, error) {
	return s.tokens.Verify(token)
}

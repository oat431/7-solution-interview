package application

import (
	"context"
	"strings"
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
	ttl    time.Duration
}

func NewAuthService(repo UserRepository, hasher PasswordHasher, tokens TokenManager, ttl time.Duration) *AuthService {
	return &AuthService{repo: repo, hasher: hasher, tokens: tokens, ttl: ttl}
}

// Login returns the same error for wrong email and wrong password — no user
// enumeration (AC-002c/d).
func (s *AuthService) Login(ctx context.Context, email, password string) (LoginResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	stored, err := s.repo.FindByEmail(ctx, email)
	if err == domain.ErrNotFound {
		return LoginResult{}, domain.ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, err
	}

	if err := s.hasher.Compare(stored.PasswordHash, password); err != nil {
		return LoginResult{}, domain.ErrInvalidCredentials
	}

	token, err := s.tokens.Issue(ctx, stored.ID, stored.Email, s.ttl)
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{User: stored.User, Token: token, ExpiresIn: s.ttl}, nil
}

// VerifyToken is shared by the REST middleware and the gRPC interceptor.
func (s *AuthService) VerifyToken(token string) (TokenClaims, error) {
	return s.tokens.Verify(token)
}

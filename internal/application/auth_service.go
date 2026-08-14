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

// AuthService handles login and token verification. Registration is
// intentionally NOT here: both the public register endpoint and the protected
// create-user endpoint call UserService.Create — one use case, two adapters
// (assumption A7).
type AuthService struct {
	repo   UserRepository
	hasher PasswordHasher
	tokens TokenManager
	ttl    time.Duration
}

func NewAuthService(repo UserRepository, hasher PasswordHasher, tokens TokenManager, ttl time.Duration) *AuthService {
	return &AuthService{repo: repo, hasher: hasher, tokens: tokens, ttl: ttl}
}

// Login checks credentials and issues a JWT. Wrong email and wrong password
// return the same error — no user enumeration (AC-002c/d).
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

// VerifyToken validates a JWT and returns its claims. Shared by the REST
// auth middleware and the gRPC interceptor — one verifier, one policy.
func (s *AuthService) VerifyToken(token string) (TokenClaims, error) {
	return s.tokens.Verify(token)
}

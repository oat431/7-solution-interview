package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/oat431/7-solution-interview/internal/application"
)

const tokenIssuer = "sevensolutions-user-api"

// JWTManager implements application.TokenManager with HMAC-SHA256 (HS256).
// The TTL is signing policy, so it lives here rather than at call sites.
type JWTManager struct {
	secret []byte
	ttl    time.Duration
	issuer string
}

func NewJWTManager(secret []byte, ttl time.Duration) *JWTManager {
	return &JWTManager{secret: secret, ttl: ttl, issuer: tokenIssuer}
}

type claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func (m *JWTManager) Issue(_ context.Context, c application.TokenClaims) (string, time.Duration, error) {
	now := time.Now()
	claims := claims{
		Email: c.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   c.Subject,
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", 0, err
	}
	return token, m.ttl, nil
}

// Verify parses and validates a token. The algorithm is pinned to HS256 via
// WithValidMethods — a token signed with any other alg is rejected.
func (m *JWTManager) Verify(token string) (application.TokenClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &claims{}, func(t *jwt.Token) (any, error) {
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return application.TokenClaims{}, fmt.Errorf("invalid token: %w", err)
	}

	c, ok := parsed.Claims.(*claims)
	if !ok || !parsed.Valid {
		return application.TokenClaims{}, errors.New("invalid token claims")
	}
	if c.Subject == "" {
		return application.TokenClaims{}, errors.New("token missing subject")
	}

	return application.TokenClaims{Subject: c.Subject, Email: c.Email}, nil
}

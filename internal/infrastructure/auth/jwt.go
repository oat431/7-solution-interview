package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/oat431/backend-challenge/internal/application"
)

const tokenIssuer = "sevensolutions-user-api"

// JWTManager implements application.TokenManager with HMAC-SHA256 (HS256).
type JWTManager struct {
	secret []byte
	issuer string
}

func NewJWTManager(secret []byte) *JWTManager {
	return &JWTManager{secret: secret, issuer: tokenIssuer}
}

type claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// Issue signs a token with the given subject (user ID), email and TTL.
func (m *JWTManager) Issue(_ context.Context, subject, email string, ttl time.Duration) (string, error) {
	now := time.Now()
	c := claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(m.secret)
}

// Verify parses and validates a token. The algorithm is pinned to HS256 via
// WithValidMethods — a token signed with any other alg is rejected, and
// exp/iat validation happens automatically through ParseWithClaims.
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

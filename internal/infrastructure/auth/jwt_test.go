package auth

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/oat431/7-solution-interview/internal/application"
)

const testSecret = "0123456789abcdef0123456789abcdef" // 32 bytes

var testClaims = application.TokenClaims{Subject: "user-123", Email: "ada@example.com"}

func newManager(t *testing.T, ttl time.Duration) *JWTManager {
	t.Helper()
	return NewJWTManager([]byte(testSecret), ttl)
}

func TestIssueAndVerifyRoundtrip(t *testing.T) {
	m := newManager(t, time.Hour)

	token, expiresIn, err := m.Issue(context.Background(), testClaims)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if expiresIn != time.Hour {
		t.Fatalf("expected expiresIn 1h, got %v", expiresIn)
	}

	claims, err := m.Verify(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims != testClaims {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestVerifyExpiredToken(t *testing.T) {
	m := newManager(t, -time.Minute)

	token, _, err := m.Issue(context.Background(), testClaims)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	if _, err := m.Verify(token); err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestVerifyTamperedToken(t *testing.T) {
	m := newManager(t, time.Hour)
	token, _, err := m.Issue(context.Background(), testClaims)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	// Corrupt the payload segment.
	parts := strings.Split(token, ".")
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	raw[0] ^= 0xff // flip bits
	parts[1] = base64.RawURLEncoding.EncodeToString(raw)

	if _, err := m.Verify(strings.Join(parts, ".")); err == nil {
		t.Fatal("expected error for tampered token")
	}
}

func TestVerifyTokenSignedWithDifferentSecret(t *testing.T) {
	m := newManager(t, time.Hour)
	other := NewJWTManager([]byte("another-secret-0123456789abcdef!!"), time.Hour)

	token, _, err := other.Issue(context.Background(), testClaims)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if _, err := m.Verify(token); err == nil {
		t.Fatal("expected error for token signed with different secret")
	}
}

// TestVerifyRejectsNoneAlgorithm ensures alg=none tokens are rejected.
func TestVerifyRejectsNoneAlgorithm(t *testing.T) {
	m := newManager(t, time.Hour)

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"user-123","exp":4102444800}`))
	noneToken := header + "." + payload + "."

	if _, err := m.Verify(noneToken); err == nil {
		t.Fatal("expected error for alg=none token")
	}
}

func TestVerifyMissingSubject(t *testing.T) {
	m := newManager(t, time.Hour)
	token, _, err := m.Issue(context.Background(), application.TokenClaims{Subject: "", Email: "ada@example.com"})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if _, err := m.Verify(token); err == nil {
		t.Fatal("expected error for token without subject")
	}
}

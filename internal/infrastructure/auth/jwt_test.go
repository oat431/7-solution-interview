package auth

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

const testSecret = "0123456789abcdef0123456789abcdef" // 32 bytes

func TestIssueAndVerifyRoundtrip(t *testing.T) {
	m := NewJWTManager([]byte(testSecret))

	token, err := m.Issue(context.Background(), "user-123", "ada@example.com", time.Hour)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	claims, err := m.Verify(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.Subject != "user-123" || claims.Email != "ada@example.com" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestVerifyExpiredToken(t *testing.T) {
	m := NewJWTManager([]byte(testSecret))

	token, err := m.Issue(context.Background(), "user-123", "ada@example.com", -time.Minute)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	if _, err := m.Verify(token); err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestVerifyTamperedToken(t *testing.T) {
	m := NewJWTManager([]byte(testSecret))
	token, err := m.Issue(context.Background(), "user-123", "ada@example.com", time.Hour)
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
	m := NewJWTManager([]byte(testSecret))
	other := NewJWTManager([]byte("another-secret-0123456789abcdef!!"))

	token, err := other.Issue(context.Background(), "user-123", "ada@example.com", time.Hour)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if _, err := m.Verify(token); err == nil {
		t.Fatal("expected error for token signed with different secret")
	}
}

// TestVerifyRejectsNoneAlgorithm ensures alg=none tokens are rejected.
func TestVerifyRejectsNoneAlgorithm(t *testing.T) {
	m := NewJWTManager([]byte(testSecret))

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"user-123","exp":4102444800}`))
	noneToken := header + "." + payload + "."

	if _, err := m.Verify(noneToken); err == nil {
		t.Fatal("expected error for alg=none token")
	}
}

func TestVerifyMissingSubject(t *testing.T) {
	m := NewJWTManager([]byte(testSecret))
	// Issue a token whose subject is empty by direct signing.
	token, err := m.Issue(context.Background(), "", "ada@example.com", time.Hour)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if _, err := m.Verify(token); err == nil {
		t.Fatal("expected error for token without subject")
	}
}

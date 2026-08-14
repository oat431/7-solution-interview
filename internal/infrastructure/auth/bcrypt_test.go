package auth

import (
	"strings"
	"testing"
)

func TestBcryptHashAndCompare(t *testing.T) {
	h := NewBcryptHasher()

	hash, err := h.Hash("s3cret-pass")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "s3cret-pass" || strings.Contains(hash, "s3cret-pass") {
		t.Fatal("hash must not contain the plaintext password")
	}
	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") && !strings.HasPrefix(hash, "$2y$") {
		t.Fatalf("unexpected bcrypt hash format: %q", hash[:7])
	}

	if err := h.Compare(hash, "s3cret-pass"); err != nil {
		t.Fatalf("compare correct password: %v", err)
	}
	if err := h.Compare(hash, "wrong-pass"); err == nil {
		t.Fatal("expected mismatch for wrong password")
	}
}

func TestBcryptHashesAreSalted(t *testing.T) {
	h := NewBcryptHasher()
	a, err := h.Hash("same-password")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	b, err := h.Hash("same-password")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if a == b {
		t.Fatal("two hashes of the same password must differ (salting)")
	}
}

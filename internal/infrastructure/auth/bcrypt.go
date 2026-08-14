// Package auth provides the password-hashing and token-manager adapters.
package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// BcryptHasher implements application.PasswordHasher using bcrypt.
type BcryptHasher struct {
	cost int
}

func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{cost: bcrypt.DefaultCost}
}

func (h *BcryptHasher) Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Compare returns an error on mismatch; the caller maps it to
// domain.ErrInvalidCredentials so that wrong-password and wrong-email are
// indistinguishable to clients.
func (h *BcryptHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

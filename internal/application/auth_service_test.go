package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/oat431/7-solution-interview/internal/application"
	"github.com/oat431/7-solution-interview/internal/domain"
	"github.com/oat431/7-solution-interview/testutil"
)

type fakeTokens struct{}

func (fakeTokens) Issue(_ context.Context, subject, email string, ttl time.Duration) (string, error) {
	return subject + "|" + email + "|" + ttl.String(), nil
}

func (fakeTokens) Verify(token string) (application.TokenClaims, error) {
	if token == "" {
		return application.TokenClaims{}, errors.New("invalid")
	}
	return application.TokenClaims{Subject: "sub", Email: "email"}, nil
}

func newTestAuth() (*application.AuthService, *application.UserService, *testutil.FakeUserRepository) {
	repo := testutil.NewFakeUserRepository()
	hasher := testutil.FakeHasher{}
	users := application.NewUserService(repo, hasher)
	auth := application.NewAuthService(repo, hasher, fakeTokens{}, time.Hour)
	return auth, users, repo
}

func TestLoginHappyPath(t *testing.T) {
	auth, users, _ := newTestAuth()

	if _, err := users.Create(context.Background(), validInput()); err != nil {
		t.Fatalf("create: %v", err)
	}

	res, err := auth.Login(context.Background(), "ada@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if res.Token == "" {
		t.Fatal("expected token")
	}
	if res.ExpiresIn != time.Hour {
		t.Fatalf("expected expiresIn 1h, got %v", res.ExpiresIn)
	}
	if res.User.Email != "ada@example.com" {
		t.Fatalf("unexpected user: %+v", res.User)
	}
}

func TestLoginNormalizesEmail(t *testing.T) {
	auth, users, _ := newTestAuth()
	if _, err := users.Create(context.Background(), validInput()); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := auth.Login(context.Background(), "  ADA@Example.COM ", "s3cret-pass"); err != nil {
		t.Fatalf("login with unnormalized email: %v", err)
	}
}

func TestLoginWrongPasswordIsInvalidCredentials(t *testing.T) {
	auth, users, _ := newTestAuth()
	if _, err := users.Create(context.Background(), validInput()); err != nil {
		t.Fatalf("create: %v", err)
	}

	_, err := auth.Login(context.Background(), "ada@example.com", "wrong-password")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginUnknownEmailIsInvalidCredentials(t *testing.T) {
	auth, _, _ := newTestAuth()

	_, err := auth.Login(context.Background(), "ghost@example.com", "whatever")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginUnexpectedRepoErrorPropagates(t *testing.T) {
	auth, _, repo := newTestAuth()

	boom := errors.New("db exploded")
	repo.FailFindByEmail(boom)

	_, err := auth.Login(context.Background(), "ada@example.com", "s3cret-pass")
	if !errors.Is(err, boom) {
		t.Fatalf("expected repo error to propagate, got %v", err)
	}
}

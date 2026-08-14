package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/oat431/7-solution-interview/internal/application"
	"github.com/oat431/7-solution-interview/internal/domain"
	"github.com/oat431/7-solution-interview/testutil"
)

func newTestService() (*application.UserService, *testutil.FakeUserRepository) {
	repo := testutil.NewFakeUserRepository()
	return application.NewUserService(repo, testutil.FakeHasher{}), repo
}

func validInput() domain.NewUserInput {
	return domain.NewUserInput{
		Name:     "Ada Lovelace",
		Email:    "ada@example.com",
		Password: "s3cret-pass",
	}
}

func TestCreateHappyPath(t *testing.T) {
	svc, repo := newTestService()

	u, err := svc.Create(context.Background(), validInput())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if u.ID == "" || u.Name != "Ada Lovelace" || u.Email != "ada@example.com" {
		t.Fatalf("unexpected user: %+v", u)
	}
	if u.CreatedAt.IsZero() {
		t.Fatal("createdAt must be set")
	}

	// Hash must be stored, never plaintext.
	stored, err := repo.FindByEmail(context.Background(), "ada@example.com")
	if err != nil {
		t.Fatalf("find by email: %v", err)
	}
	if stored.PasswordHash != "h:s3cret-pass" {
		t.Fatalf("expected hashed password, got %q", stored.PasswordHash)
	}
}

func TestCreateNormalizesEmail(t *testing.T) {
	svc, _ := newTestService()
	in := validInput()
	in.Email = "  Ada@Example.COM "

	u, err := svc.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if u.Email != "ada@example.com" {
		t.Fatalf("email not normalized: %q", u.Email)
	}
}

func TestCreateDuplicateEmail(t *testing.T) {
	svc, _ := newTestService()
	if _, err := svc.Create(context.Background(), validInput()); err != nil {
		t.Fatalf("first create: %v", err)
	}

	in := validInput()
	in.Name = "Second Ada"
	_, err := svc.Create(context.Background(), in)
	if !errors.Is(err, domain.ErrEmailExists) {
		t.Fatalf("expected ErrEmailExists, got %v", err)
	}
}

func TestCreateValidationError(t *testing.T) {
	svc, _ := newTestService()
	in := validInput()
	in.Email = "not-an-email"

	_, err := svc.Create(context.Background(), in)
	var verr domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}

func TestGetNotFound(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.Get(context.Background(), "665f1c2d3e4f5a6b7c8d9e0f")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestList(t *testing.T) {
	svc, repo := newTestService()
	repo.Seed("A", "a@example.com", "h:x")
	repo.Seed("B", "b@example.com", "h:x")

	users, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
}

func TestUpdateNameOnly(t *testing.T) {
	svc, repo := newTestService()
	seeded := repo.Seed("Ada", "ada@example.com", "h:x")

	name := "Ada Byron"
	u, err := svc.Update(context.Background(), seeded.ID, application.UpdateUserInput{Name: &name})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if u.Name != "Ada Byron" || u.Email != "ada@example.com" {
		t.Fatalf("unexpected user: %+v", u)
	}
}

func TestUpdateEmailConflict(t *testing.T) {
	svc, repo := newTestService()
	seeded := repo.Seed("Ada", "ada@example.com", "h:x")
	repo.Seed("Grace", "grace@example.com", "h:x")

	email := "grace@example.com"
	_, err := svc.Update(context.Background(), seeded.ID, application.UpdateUserInput{Email: &email})
	if !errors.Is(err, domain.ErrEmailExists) {
		t.Fatalf("expected ErrEmailExists, got %v", err)
	}
}

func TestUpdateEmptyBody(t *testing.T) {
	svc, repo := newTestService()
	seeded := repo.Seed("Ada", "ada@example.com", "h:x")

	_, err := svc.Update(context.Background(), seeded.ID, application.UpdateUserInput{})
	var verr domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}

func TestUpdateNotFound(t *testing.T) {
	svc, _ := newTestService()
	name := "X"
	_, err := svc.Update(context.Background(), "665f1c2d3e4f5a6b7c8d9e0f", application.UpdateUserInput{Name: &name})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteThenGet(t *testing.T) {
	svc, repo := newTestService()
	seeded := repo.Seed("Ada", "ada@example.com", "h:x")

	if err := svc.Delete(context.Background(), seeded.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.Get(context.Background(), seeded.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestDeleteNotFound(t *testing.T) {
	svc, _ := newTestService()
	if err := svc.Delete(context.Background(), "665f1c2d3e4f5a6b7c8d9e0f"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetInvalidID(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.Get(context.Background(), "not-an-objectid")
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

func TestUpdateInvalidID(t *testing.T) {
	svc, _ := newTestService()
	name := "X"
	_, err := svc.Update(context.Background(), "not-an-objectid", application.UpdateUserInput{Name: &name})
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

func TestDeleteInvalidID(t *testing.T) {
	svc, _ := newTestService()
	if err := svc.Delete(context.Background(), "not-an-objectid"); !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

func TestUpdateInvalidEmail(t *testing.T) {
	svc, repo := newTestService()
	seeded := repo.Seed("Ada", "ada@example.com", "h:x")

	email := "not-an-email"
	_, err := svc.Update(context.Background(), seeded.ID, application.UpdateUserInput{Email: &email})
	var verr domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	for _, fe := range verr {
		if fe.Field == "email" {
			return
		}
	}
	t.Fatalf("expected email field error, got %v", verr)
}

func TestCreateHasherErrorWrapped(t *testing.T) {
	repo := testutil.NewFakeUserRepository()
	svc := application.NewUserService(repo, testutil.FailingHasher{})

	_, err := svc.Create(context.Background(), validInput())
	if err == nil {
		t.Fatal("expected error from failing hasher")
	}
	if !strings.Contains(err.Error(), "hash password") {
		t.Fatalf("expected wrapped hasher error, got %v", err)
	}
}

func TestUpdateUnexpectedRepoErrorPropagates(t *testing.T) {
	svc, repo := newTestService()
	seeded := repo.Seed("Ada", "ada@example.com", "h:x")

	boom := errors.New("db exploded")
	repo.FailFindByEmail(boom)

	email := "new@example.com"
	_, err := svc.Update(context.Background(), seeded.ID, application.UpdateUserInput{Email: &email})
	if !errors.Is(err, boom) {
		t.Fatalf("expected repo error to propagate, got %v", err)
	}
}

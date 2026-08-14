// Package testutil provides hand-written test doubles: an in-memory
// UserRepository fake and a deterministic password hasher. No third-party
// mock library — the repository port makes fakes trivial (ADR-03).
package testutil

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/oat431/7-solution-interview/internal/application"
	"github.com/oat431/7-solution-interview/internal/domain"
)

type fakeUser struct {
	user         domain.User
	passwordHash string
}

// FakeUserRepository implements application.UserRepository in memory.
type FakeUserRepository struct {
	mu              sync.Mutex
	users           map[string]fakeUser
	emailIndex      map[string]string
	seq             int
	failFindByEmail error
}

func NewFakeUserRepository() *FakeUserRepository {
	return &FakeUserRepository{
		users:      map[string]fakeUser{},
		emailIndex: map[string]string{},
	}
}

// Seed inserts a user directly for test setup, bypassing validation/hashing.
func (f *FakeUserRepository) Seed(name, email, hash string) domain.User {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	id := fmt.Sprintf("%024x", f.seq)
	u := domain.User{ID: id, Name: name, Email: email, CreatedAt: time.Now().UTC()}
	f.users[id] = fakeUser{user: u, passwordHash: hash}
	f.emailIndex[email] = id
	return u
}

func (f *FakeUserRepository) Create(_ context.Context, rec application.CreateUserRecord) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.emailIndex[rec.Email]; exists {
		return domain.User{}, domain.ErrEmailExists
	}
	f.seq++
	id := fmt.Sprintf("%024x", f.seq)
	u := domain.User{ID: id, Name: rec.Name, Email: rec.Email, CreatedAt: time.Now().UTC()}
	f.users[id] = fakeUser{user: u, passwordHash: rec.PasswordHash}
	f.emailIndex[rec.Email] = id
	return u, nil
}

func (f *FakeUserRepository) FindByID(_ context.Context, id string) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	su, ok := f.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return su.user, nil
}

func (f *FakeUserRepository) FindByEmail(_ context.Context, email string) (application.StoredUser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failFindByEmail != nil {
		return application.StoredUser{}, f.failFindByEmail
	}
	id, ok := f.emailIndex[email]
	if !ok {
		return application.StoredUser{}, domain.ErrNotFound
	}
	su := f.users[id]
	return application.StoredUser{User: su.user, PasswordHash: su.passwordHash}, nil
}

// FailFindByEmail makes every subsequent FindByEmail call return err.
// Used to exercise unexpected-repository-error paths in service tests.
func (f *FakeUserRepository) FailFindByEmail(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failFindByEmail = err
}

func (f *FakeUserRepository) List(_ context.Context) ([]domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := make([]string, 0, len(f.users))
	for id := range f.users {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]domain.User, 0, len(ids))
	for _, id := range ids {
		out = append(out, f.users[id].user)
	}
	return out, nil
}

func (f *FakeUserRepository) Update(_ context.Context, id string, in application.UpdateUserInput) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	su, ok := f.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	if in.Email != nil {
		if owner, exists := f.emailIndex[*in.Email]; exists && owner != id {
			return domain.User{}, domain.ErrEmailExists
		}
		delete(f.emailIndex, su.user.Email)
		su.user.Email = *in.Email
		f.emailIndex[*in.Email] = id
	}
	if in.Name != nil {
		su.user.Name = *in.Name
	}
	f.users[id] = su
	return su.user, nil
}

func (f *FakeUserRepository) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	su, ok := f.users[id]
	if !ok {
		return domain.ErrNotFound
	}
	delete(f.emailIndex, su.user.Email)
	delete(f.users, id)
	return nil
}

func (f *FakeUserRepository) Count(_ context.Context) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return int64(len(f.users)), nil
}

// FakeHasher implements application.PasswordHasher with a deterministic,
// instantly-reversible scheme for fast unit tests. Real bcrypt behavior is
// covered separately in internal/infrastructure/auth/bcrypt_test.go.
type FakeHasher struct{}

func (FakeHasher) Hash(password string) (string, error) {
	return "h:" + password, nil
}

func (FakeHasher) Compare(hash, password string) error {
	if hash != "h:"+password {
		return errors.New("hash mismatch")
	}
	return nil
}

// FailingHasher always fails, for exercising hasher-error paths.
type FailingHasher struct{}

func (FailingHasher) Hash(string) (string, error) {
	return "", errors.New("hasher exploded")
}

func (FailingHasher) Compare(string, string) error {
	return errors.New("hasher exploded")
}

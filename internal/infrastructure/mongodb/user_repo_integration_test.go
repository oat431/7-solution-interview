//go:build integration

// Package mongodb integration tests (ACT-Q1 / ACT-Q3 from 072 polish
// handoff). These run against a REAL MongoDB — opt-in via build tag so the
// default unit gate (`make test`, DoD) stays hermetic:
//
//	go test -tags integration -race ./internal/infrastructure/mongodb/
//
// MONGO_URI defaults to mongodb://localhost:27017 (the compose mongo
// service publishes that port). Each test gets its own throwaway database,
// dropped after the test.
package mongodb_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"github.com/oat431/7-solution-interview/internal/application"
	"github.com/oat431/7-solution-interview/internal/domain"
	"github.com/oat431/7-solution-interview/internal/infrastructure/auth"
	"github.com/oat431/7-solution-interview/internal/infrastructure/mongodb"
)

var dbSeq int

func mongoURI() string {
	if uri := os.Getenv("MONGO_URI"); uri != "" {
		return uri
	}
	return "mongodb://localhost:27017"
}

// newTestRepo connects to real Mongo and gives the test a fresh database
// (dropped on cleanup) plus a repository on the standard "users" collection.
func newTestRepo(t *testing.T) (*mongodb.UserRepository, *mongo.Database) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(mongoURI()).SetServerSelectionTimeout(5 * time.Second))
	if err != nil {
		t.Fatalf("mongo connect: %v", err)
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		t.Fatalf("mongo ping (%s): %v", mongoURI(), err)
	}

	dbSeq++
	dbName := fmt.Sprintf("qa_it_%d_%d", time.Now().UnixNano()%1_000_000_000_000, dbSeq)
	db := client.Database(dbName)
	t.Cleanup(func() {
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer dropCancel()
		_ = db.Drop(dropCtx)
		_ = client.Disconnect(dropCtx)
	})

	repo := mongodb.NewUserRepository(db)
	idxCtx, idxCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer idxCancel()
	if err := repo.EnsureIndexes(idxCtx); err != nil {
		t.Fatalf("ensure indexes: %v", err)
	}
	return repo, db
}

func ctxOK(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func createRecord(name, email string) application.CreateUserRecord {
	return application.CreateUserRecord{
		Name:         name,
		Email:        email,
		PasswordHash: "$2a$10$" + strings.Repeat("q", 53), // bcrypt-shaped placeholder
	}
}

// TC-QINT-01 — Create persists the document and returns the mapped user.
func TestIntegrationCreate(t *testing.T) {
	repo, db := newTestRepo(t)
	ctx := ctxOK(t)

	user, err := repo.Create(ctx, createRecord("Ada Lovelace", "ada@example.com"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !domain.IsValidID(user.ID) {
		t.Fatalf("Create returned non-ObjectID id %q", user.ID)
	}
	if user.Name != "Ada Lovelace" || user.Email != "ada@example.com" {
		t.Fatalf("Create returned wrong user: %+v", user)
	}
	if user.CreatedAt.IsZero() || user.CreatedAt.After(time.Now().Add(time.Minute)) {
		t.Fatalf("Create returned implausible CreatedAt: %v", user.CreatedAt)
	}

	// The persisted document must carry a password hash, never plaintext,
	// and the adapter must never expose it on domain.User (compile-time:
	// domain.User has no password field).
	var raw bson.M
	if err := db.Collection("users").FindOne(ctx, bson.M{"email": "ada@example.com"}).Decode(&raw); err != nil {
		t.Fatalf("raw doc lookup: %v", err)
	}
	hash, _ := raw["password_hash"].(string)
	if !strings.HasPrefix(hash, "$2") {
		t.Fatalf("persisted password_hash is not bcrypt-shaped: %q", hash)
	}
	if _, leaked := raw["password"]; leaked {
		t.Fatal("persisted document contains a plaintext password field")
	}
}

// TC-QINT-02 — duplicate email on Create maps unique-index violation to
// domain.ErrEmailExists (AC-001e/f backstop).
func TestIntegrationCreateDuplicateEmail(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := ctxOK(t)

	if _, err := repo.Create(ctx, createRecord("First", "dup@example.com")); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err := repo.Create(ctx, createRecord("Second", "dup@example.com"))
	if !errors.Is(err, domain.ErrEmailExists) {
		t.Fatalf("duplicate Create: want ErrEmailExists, got %v", err)
	}
}

// TC-QINT-03 — EnsureIndexes is idempotent (A12: called on every startup).
func TestIntegrationEnsureIndexesIdempotent(t *testing.T) {
	repo, db := newTestRepo(t) // newTestRepo already called it once
	ctx := ctxOK(t)

	if err := repo.EnsureIndexes(ctx); err != nil {
		t.Fatalf("second EnsureIndexes call: %v", err)
	}

	names, err := db.Collection("users").Indexes().List(ctx)
	if err != nil {
		t.Fatalf("list indexes: %v", err)
	}
	var cursorDocs []bson.M
	if err := names.All(ctx, &cursorDocs); err != nil {
		t.Fatalf("decode indexes: %v", err)
	}
	found := false
	for _, doc := range cursorDocs {
		if doc["name"] == "ux_users_email" {
			found = true
			if doc["unique"] != true {
				t.Fatalf("ux_users_email is not unique: %+v", doc)
			}
		}
	}
	if !found {
		t.Fatal("unique index ux_users_email not found")
	}
}

// TC-QINT-04 — FindByID: happy path, unknown id, malformed id.
func TestIntegrationFindByID(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := ctxOK(t)

	created, err := repo.Create(ctx, createRecord("Grace Hopper", "grace@example.com"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.ID != created.ID || got.Name != created.Name || got.Email != created.Email {
		t.Fatalf("FindByID mismatch:\n got  %+v\n want %+v", got, created)
	}
	// BSON datetime stores millisecond precision; allow the round-trip
	// truncation (a repo read-back can never carry sub-ms digits).
	if d := got.CreatedAt.Sub(created.CreatedAt); d < -time.Millisecond || d > 0 {
		t.Fatalf("FindByID CreatedAt drift beyond ms truncation: got %v want ~%v", got.CreatedAt, created.CreatedAt)
	}

	if _, err := repo.FindByID(ctx, "666666666666666666666666"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("FindByID unknown: want ErrNotFound, got %v", err)
	}
	if _, err := repo.FindByID(ctx, "zzz"); !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("FindByID malformed: want ErrInvalidID, got %v", err)
	}
}

// TC-QINT-05 — FindByEmail returns the hash for auth; unknown → ErrNotFound.
func TestIntegrationFindByEmail(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := ctxOK(t)

	rec := createRecord("Alan Turing", "alan@example.com")
	if _, err := repo.Create(ctx, rec); err != nil {
		t.Fatalf("Create: %v", err)
	}

	stored, err := repo.FindByEmail(ctx, "alan@example.com")
	if err != nil {
		t.Fatalf("FindByEmail: %v", err)
	}
	if stored.PasswordHash != rec.PasswordHash {
		t.Fatalf("FindByEmail hash mismatch: got %q want %q", stored.PasswordHash, rec.PasswordHash)
	}
	if stored.Email != "alan@example.com" || stored.Name != "Alan Turing" {
		t.Fatalf("FindByEmail wrong user: %+v", stored.User)
	}

	if _, err := repo.FindByEmail(ctx, "nobody@example.com"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("FindByEmail unknown: want ErrNotFound, got %v", err)
	}
}

// TC-QINT-06 — List returns every user (no pagination, A10).
func TestIntegrationList(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := ctxOK(t)

	if users, err := repo.List(ctx); err != nil || len(users) != 0 {
		t.Fatalf("List on empty collection: want ([], nil), got (%v, %v)", users, err)
	}

	want := map[string]bool{"a@example.com": false, "b@example.com": false, "c@example.com": false}
	for email := range want {
		if _, err := repo.Create(ctx, createRecord("User", email)); err != nil {
			t.Fatalf("Create %s: %v", email, err)
		}
	}

	users, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(users) != len(want) {
		t.Fatalf("List: want %d users, got %d", len(want), len(users))
	}
	for _, u := range users {
		if _, ok := want[u.Email]; !ok {
			t.Fatalf("List returned unexpected user %+v", u)
		}
		want[u.Email] = true
	}
	for email, seen := range want {
		if !seen {
			t.Fatalf("List missed user %s", email)
		}
	}
}

// TC-QINT-07 — Update: name/email happy paths, duplicate-email conflict via
// the unique index (AC-006e backstop), unknown id, malformed id.
func TestIntegrationUpdate(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := ctxOK(t)

	u1, err := repo.Create(ctx, createRecord("One", "one@example.com"))
	if err != nil {
		t.Fatalf("Create u1: %v", err)
	}
	u2, err := repo.Create(ctx, createRecord("Two", "two@example.com"))
	if err != nil {
		t.Fatalf("Create u2: %v", err)
	}

	newName := "One Renamed"
	got, err := repo.Update(ctx, u1.ID, application.UpdateUserInput{Name: &newName})
	if err != nil {
		t.Fatalf("Update name: %v", err)
	}
	if got.Name != newName || got.Email != "one@example.com" {
		t.Fatalf("Update name wrong result: %+v", got)
	}

	newEmail := "two-new@example.com"
	got, err = repo.Update(ctx, u2.ID, application.UpdateUserInput{Email: &newEmail})
	if err != nil {
		t.Fatalf("Update email: %v", err)
	}
	if got.Email != newEmail || got.Name != "Two" {
		t.Fatalf("Update email wrong result: %+v", got)
	}

	conflict := "one@example.com"
	if _, err := repo.Update(ctx, u2.ID, application.UpdateUserInput{Email: &conflict}); !errors.Is(err, domain.ErrEmailExists) {
		t.Fatalf("Update to taken email: want ErrEmailExists, got %v", err)
	}

	if _, err := repo.Update(ctx, "666666666666666666666666", application.UpdateUserInput{Name: &newName}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Update unknown id: want ErrNotFound, got %v", err)
	}
	if _, err := repo.Update(ctx, "zzz", application.UpdateUserInput{Name: &newName}); !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("Update malformed id: want ErrInvalidID, got %v", err)
	}
}

// TC-QINT-08 — Delete: happy path removes the doc; unknown and malformed ids
// map to ErrNotFound / ErrInvalidID.
func TestIntegrationDelete(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := ctxOK(t)

	user, err := repo.Create(ctx, createRecord("Del", "del@example.com"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.Delete(ctx, user.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.FindByID(ctx, user.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("FindByID after Delete: want ErrNotFound, got %v", err)
	}
	if err := repo.Delete(ctx, user.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Delete twice: want ErrNotFound, got %v", err)
	}
	if err := repo.Delete(ctx, "zzz"); !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("Delete malformed: want ErrInvalidID, got %v", err)
	}
}

// TC-QINT-09 — Count tracks inserts and deletes.
func TestIntegrationCount(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := ctxOK(t)

	if n, err := repo.Count(ctx); err != nil || n != 0 {
		t.Fatalf("Count empty: want (0, nil), got (%d, %v)", n, err)
	}
	users := make([]domain.User, 0, 3)
	for i := 0; i < 3; i++ {
		u, err := repo.Create(ctx, createRecord("U", fmt.Sprintf("count-%d@example.com", i)))
		if err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
		users = append(users, u)
	}
	if n, err := repo.Count(ctx); err != nil || n != 3 {
		t.Fatalf("Count after 3 inserts: want (3, nil), got (%d, %v)", n, err)
	}
	if err := repo.Delete(ctx, users[0].ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if n, err := repo.Count(ctx); err != nil || n != 2 {
		t.Fatalf("Count after delete: want (2, nil), got (%d, %v)", n, err)
	}
}

// TC-QINT-10 — AC-001f at the service level against real Mongo (ACT-Q3):
// N goroutines register the same email concurrently through the real use
// case (pre-check + bcrypt + unique index); exactly one succeeds and every
// other attempt surfaces ErrEmailExists.
func TestIntegrationConcurrentRegisterSameEmail(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := ctxOK(t)

	svc := application.NewUserService(repo, auth.NewBcryptHasher())

	const goroutines = 16
	var wg sync.WaitGroup
	wg.Add(goroutines)

	results := make(chan error, goroutines)
	start := make(chan struct{}) // maximize concurrency at the starting line
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			<-start
			_, err := svc.Create(ctx, domain.NewUserInput{
				Name:     "Racer",
				Email:    "race@example.com",
				Password: "race-pass-123",
			})
			results <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	var successes, conflicts, other int
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, domain.ErrEmailExists):
			conflicts++
		default:
			other++
			t.Logf("unexpected error: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("race: want exactly 1 success, got %d (conflicts=%d other=%d)", successes, conflicts, other)
	}
	if conflicts != goroutines-1 || other != 0 {
		t.Fatalf("race: want %d conflicts and 0 other, got conflicts=%d other=%d", goroutines-1, conflicts, other)
	}

	if n, err := repo.Count(ctx); err != nil || n != 1 {
		t.Fatalf("after race: want exactly 1 persisted user, got (%d, %v)", n, err)
	}
}

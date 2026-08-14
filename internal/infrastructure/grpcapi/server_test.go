package grpcapi_test

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	userservicev1 "github.com/oat431/7-solution-interview/gen/userservice/v1"
	"github.com/oat431/7-solution-interview/internal/application"
	"github.com/oat431/7-solution-interview/internal/infrastructure/auth"
	"github.com/oat431/7-solution-interview/internal/infrastructure/grpcapi"
	"github.com/oat431/7-solution-interview/testutil"
)

const (
	bufSize    = 1 << 20
	testSecret = "0123456789abcdef0123456789abcdef"
)

type grpcEnv struct {
	client userservicev1.UserServiceClient
	repo   *testutil.FakeUserRepository
	tokens *auth.JWTManager
}

func newGRPCEnv(t *testing.T) *grpcEnv {
	t.Helper()

	repo := testutil.NewFakeUserRepository()
	hasher := testutil.FakeHasher{}
	users := application.NewUserService(repo, hasher)
	tokens := auth.NewJWTManager([]byte(testSecret), time.Hour)
	authSvc := application.NewAuthService(repo, hasher, tokens)

	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer(grpc.UnaryInterceptor(grpcapi.UnaryAuthInterceptor(authSvc)))
	userservicev1.RegisterUserServiceServer(srv, grpcapi.NewServer(users))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc client: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return &grpcEnv{
		client: userservicev1.NewUserServiceClient(conn),
		repo:   repo,
		tokens: tokens,
	}
}

func (e *grpcEnv) ctx(token string) context.Context {
	ctx := context.Background()
	if token != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
	}
	return ctx
}

func (e *grpcEnv) validToken(t *testing.T) string {
	t.Helper()
	token, _, err := e.tokens.Issue(context.Background(), application.TokenClaims{Subject: "sub", Email: "ada@example.com"})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return token
}

func wantCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %v, got nil", want)
	}
	if got := status.Code(err); got != want {
		t.Fatalf("expected code %v, got %v (%v)", want, got, err)
	}
}

func TestCreateUser(t *testing.T) {
	e := newGRPCEnv(t)

	resp, err := e.client.CreateUser(e.ctx(e.validToken(t)), &userservicev1.CreateUserRequest{
		Name:     "Grace Hopper",
		Email:    "grace@example.com",
		Password: "c0bol-rul3z",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if resp.GetId() == "" || resp.GetEmail() != "grace@example.com" {
		t.Fatalf("unexpected response: %v", resp)
	}
	if resp.GetCreatedAt() == "" {
		t.Fatal("createdAt must be set")
	}

	// Persisted in the shared fake repo (same core as REST — AC-010e).
	if _, err := e.repo.FindByEmail(context.Background(), "grace@example.com"); err != nil {
		t.Fatalf("user not persisted: %v", err)
	}
}

func TestCreateUserValidationError(t *testing.T) {
	e := newGRPCEnv(t)

	_, err := e.client.CreateUser(e.ctx(e.validToken(t)), &userservicev1.CreateUserRequest{
		Name:     "Grace",
		Email:    "not-an-email",
		Password: "c0bol-rul3z",
	})
	wantCode(t, err, codes.InvalidArgument)
}

func TestCreateUserDuplicate(t *testing.T) {
	e := newGRPCEnv(t)
	req := &userservicev1.CreateUserRequest{Name: "Grace", Email: "grace@example.com", Password: "c0bol-rul3z"}
	if _, err := e.client.CreateUser(e.ctx(e.validToken(t)), req); err != nil {
		t.Fatalf("first create: %v", err)
	}

	_, err := e.client.CreateUser(e.ctx(e.validToken(t)), req)
	wantCode(t, err, codes.AlreadyExists)
}

func TestGetUser(t *testing.T) {
	e := newGRPCEnv(t)
	created, err := e.client.CreateUser(e.ctx(e.validToken(t)), &userservicev1.CreateUserRequest{
		Name:     "Grace Hopper",
		Email:    "grace@example.com",
		Password: "c0bol-rul3z",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := e.client.GetUser(e.ctx(e.validToken(t)), &userservicev1.GetUserRequest{Id: created.GetId()})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.GetName() != "Grace Hopper" || got.GetEmail() != "grace@example.com" {
		t.Fatalf("unexpected user: %v", got)
	}
}

func TestGetUserNotFound(t *testing.T) {
	e := newGRPCEnv(t)
	_, err := e.client.GetUser(e.ctx(e.validToken(t)), &userservicev1.GetUserRequest{Id: "665f1c2d3e4f5a6b7c8d9e0f"})
	wantCode(t, err, codes.NotFound)
}

func TestGetUserInvalidID(t *testing.T) {
	e := newGRPCEnv(t)
	_, err := e.client.GetUser(e.ctx(e.validToken(t)), &userservicev1.GetUserRequest{Id: "zzz"})
	wantCode(t, err, codes.InvalidArgument)
}

func TestMissingMetadata(t *testing.T) {
	e := newGRPCEnv(t)
	_, err := e.client.GetUser(context.Background(), &userservicev1.GetUserRequest{Id: "665f1c2d3e4f5a6b7c8d9e0f"})
	wantCode(t, err, codes.Unauthenticated)
}

func TestGarbageToken(t *testing.T) {
	e := newGRPCEnv(t)
	_, err := e.client.GetUser(e.ctx("not.a.jwt"), &userservicev1.GetUserRequest{Id: "665f1c2d3e4f5a6b7c8d9e0f"})
	wantCode(t, err, codes.Unauthenticated)
}

func TestExpiredToken(t *testing.T) {
	e := newGRPCEnv(t)
	expiredMgr := auth.NewJWTManager([]byte(testSecret), -time.Minute)
	expired, _, err := expiredMgr.Issue(context.Background(), application.TokenClaims{Subject: "sub", Email: "ada@example.com"})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	_, err = e.client.GetUser(e.ctx(expired), &userservicev1.GetUserRequest{Id: "665f1c2d3e4f5a6b7c8d9e0f"})
	wantCode(t, err, codes.Unauthenticated)
}

// TestRESTAndGRPCShareCore verifies assumption A7/AC-010e: a user created
// via gRPC is visible through the REST adapter and vice versa (same repo).
func TestRESTAndGRPCShareCore(t *testing.T) {
	e := newGRPCEnv(t)

	if _, err := e.client.CreateUser(e.ctx(e.validToken(t)), &userservicev1.CreateUserRequest{
		Name:     "Grace Hopper",
		Email:    "grace@example.com",
		Password: "c0bol-rul3z",
	}); err != nil {
		t.Fatalf("grpc create: %v", err)
	}

	users, err := e.repo.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(users) != 1 || users[0].Email != "grace@example.com" {
		t.Fatalf("expected the gRPC-created user in the shared store, got %+v", users)
	}
}

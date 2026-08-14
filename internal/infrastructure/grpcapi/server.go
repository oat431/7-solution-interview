// Package grpcapi is the gRPC driving adapter: it re-exposes the same
// application-layer use cases as the REST adapter (ADR-04), secured by a
// JWT metadata interceptor.
package grpcapi

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	userservicev1 "github.com/oat431/7-solution-interview/gen/userservice/v1"
	"github.com/oat431/7-solution-interview/internal/application"
	"github.com/oat431/7-solution-interview/internal/domain"
)

// Server implements userservicev1.UserServiceServer using the application
// core — no business logic lives here.
type Server struct {
	userservicev1.UnimplementedUserServiceServer
	users *application.UserService
}

func NewServer(users *application.UserService) *Server {
	return &Server{users: users}
}

func (s *Server) CreateUser(ctx context.Context, req *userservicev1.CreateUserRequest) (*userservicev1.User, error) {
	user, err := s.users.Create(ctx, domain.NewUserInput{
		Name:     req.GetName(),
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return toProto(user), nil
}

func (s *Server) GetUser(ctx context.Context, req *userservicev1.GetUserRequest) (*userservicev1.User, error) {
	user, err := s.users.Get(ctx, req.GetId())
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return toProto(user), nil
}

func toProto(u domain.User) *userservicev1.User {
	return &userservicev1.User{
		Id:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05.000Z07:00"),
	}
}

// mapGRPCError translates domain/application errors to gRPC status codes.
func mapGRPCError(err error) error {
	var verr domain.ValidationError
	switch {
	case errors.As(err, &verr):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrEmailExists):
		return status.Error(codes.AlreadyExists, "a user with this email already exists")
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, "user not found")
	case errors.Is(err, domain.ErrInvalidID):
		return status.Error(codes.InvalidArgument, "user id must be a valid ObjectID")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

// UnaryAuthInterceptor enforces JWT authentication from gRPC metadata. It
// shares the same verifier as the REST middleware (AC-010c/d).
func UnaryAuthInterceptor(auth *application.AuthService) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		values := md.Get("authorization")
		if len(values) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization metadata")
		}

		token, ok := bearerFromMetadata(values[0])
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "authorization must be a Bearer token")
		}

		if _, err := auth.VerifyToken(token); err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}

		return handler(ctx, req)
	}
}

func bearerFromMetadata(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	return strings.TrimSpace(header[len(prefix):]), true
}

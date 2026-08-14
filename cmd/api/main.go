// Command api wires configuration, adapters, servers and the worker, and
// handles graceful shutdown.
package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/oat431/7-solution-interview/gen/userservice/v1"
	"github.com/oat431/7-solution-interview/internal/application"
	"github.com/oat431/7-solution-interview/internal/infrastructure/auth"
	"github.com/oat431/7-solution-interview/internal/infrastructure/config"
	"github.com/oat431/7-solution-interview/internal/infrastructure/grpcapi"
	"github.com/oat431/7-solution-interview/internal/infrastructure/httpapi"
	"github.com/oat431/7-solution-interview/internal/infrastructure/logger"
	"github.com/oat431/7-solution-interview/internal/infrastructure/mongodb"
	"github.com/oat431/7-solution-interview/internal/worker"
)

const (
	mongoConnectTimeout    = 10 * time.Second
	mongoDisconnectTimeout = 5 * time.Second
	// Pool bounds replace the driver defaults (max 100 / min 0) so a
	// misbehaving caller cannot open unbounded connections, and an
	// unreachable Mongo fails fast instead of hanging 30s (ACT-D2).
	mongoMaxPoolSize            = 50
	mongoMinPoolSize            = 5
	mongoServerSelectionTimeout = 5 * time.Second
	shutdownTimeout             = 10 * time.Second
	grpcStopTimeout             = 5 * time.Second
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	logg := logger.New()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client, err := connectMongo(ctx, cfg.MongoURI)
	if err != nil {
		fatal(logg, "mongo connect failed", err)
	}
	repo := mongodb.NewUserRepository(client.Database(cfg.DBName))
	if err := ensureIndexes(ctx, repo); err != nil {
		fatal(logg, "ensure indexes failed", err)
	}

	hasher := auth.NewBcryptHasher()
	tokens := auth.NewJWTManager([]byte(cfg.JWTSecret), cfg.TokenTTL)
	users := application.NewUserService(repo, hasher)
	authSvc := application.NewAuthService(repo, hasher, tokens)

	go worker.NewUserCountWorker(repo, logg).Run(ctx, cfg.WorkerInterval)

	serverErr := make(chan error, 1)
	httpApp := serveFiber(logg, httpapi.NewApp(logg, users, authSvc), cfg.HTTPPort, serverErr)
	grpcSrv := serveGRPC(logg, users, authSvc, cfg.GRPCPort, serverErr)

	logg.Info("service started",
		"db", cfg.DBName,
		"http_port", cfg.HTTPPort,
		"grpc_port", cfg.GRPCPort,
		"worker_interval", cfg.WorkerInterval.String(),
	)

	select {
	case err := <-serverErr:
		logg.Error("server failed", "error", err)
	case <-ctx.Done():
		logg.Info("shutdown signal received")
	}
	stop()

	shutdown(logg, httpApp, grpcSrv, client)
}

func connectMongo(ctx context.Context, uri string) (*mongo.Client, error) {
	opts := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(mongoMaxPoolSize).
		SetMinPoolSize(mongoMinPoolSize).
		SetServerSelectionTimeout(mongoServerSelectionTimeout)

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, mongoConnectTimeout)
	defer cancel()
	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}
	return client, nil
}

func ensureIndexes(ctx context.Context, repo *mongodb.UserRepository) error {
	idxCtx, cancel := context.WithTimeout(ctx, mongoConnectTimeout)
	defer cancel()
	return repo.EnsureIndexes(idxCtx)
}

func serveFiber(logg *slog.Logger, app *fiber.App, port string, serverErr chan<- error) *fiber.App {
	go func() {
		logg.Info("http server listening", "port", port)
		// Fiber listens on tcp4 by default; "tcp" restores the dual-stack
		// listener the pre-Fiber net/http server had, so in-container
		// localhost (::1) healthchecks can reach it.
		if err := app.Listen(":"+port, fiber.ListenConfig{ListenerNetwork: fiber.NetworkTCP}); err != nil {
			serverErr <- err
		}
	}()
	return app
}

func serveGRPC(logg *slog.Logger, users *application.UserService, authSvc *application.AuthService, port string, serverErr chan<- error) *grpc.Server {
	// Reflection lets clients like grpcurl discover the service without a
	// local proto file.
	grpcSrv := grpc.NewServer(grpc.UnaryInterceptor(grpcapi.UnaryAuthInterceptor(authSvc)))
	userservicev1.RegisterUserServiceServer(grpcSrv, grpcapi.NewServer(users))
	reflection.Register(grpcSrv)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		serverErr <- err
		return grpcSrv
	}
	go func() {
		logg.Info("grpc server listening", "port", port)
		if serveErr := grpcSrv.Serve(lis); serveErr != nil {
			serverErr <- serveErr
		}
	}()
	return grpcSrv
}

func shutdown(logg *slog.Logger, httpApp *fiber.App, grpcSrv *grpc.Server, client *mongo.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := httpApp.ShutdownWithContext(ctx); err != nil {
		logg.Error("http shutdown failed", "error", err)
	}

	// GracefulStop waits for in-flight RPCs; fall back to hard Stop after 5s.
	grpcStopped := make(chan struct{})
	go func() {
		grpcSrv.GracefulStop()
		close(grpcStopped)
	}()
	select {
	case <-grpcStopped:
	case <-time.After(grpcStopTimeout):
		logg.Warn("grpc graceful stop timed out, forcing stop")
		grpcSrv.Stop()
	}

	disconnectCtx, cancelDisc := context.WithTimeout(context.Background(), mongoDisconnectTimeout)
	defer cancelDisc()
	if err := client.Disconnect(disconnectCtx); err != nil {
		logg.Error("mongo disconnect failed", "error", err)
	}

	logg.Info("shutdown complete")
}

func fatal(logg *slog.Logger, msg string, err error) {
	logg.Error(msg, "error", err)
	os.Exit(1)
}

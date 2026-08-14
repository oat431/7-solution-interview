// Command api wires configuration, adapters, servers and the worker, and
// handles graceful shutdown.
package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	shutdownTimeout        = 10 * time.Second
	grpcStopTimeout        = 5 * time.Second
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
	httpSrv := serveHTTP(logg, httpapi.NewRouter(logg, users, authSvc), cfg.HTTPPort, serverErr)
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

	shutdown(logg, httpSrv, grpcSrv, client)
}

func connectMongo(ctx context.Context, uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
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

func serveHTTP(logg *slog.Logger, handler http.Handler, port string, serverErr chan<- error) *http.Server {
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logg.Info("http server listening", "port", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()
	return srv
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

func shutdown(logg *slog.Logger, httpSrv *http.Server, grpcSrv *grpc.Server, client *mongo.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := httpSrv.Shutdown(ctx); err != nil {
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

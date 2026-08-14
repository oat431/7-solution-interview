// Command api is the composition root of the user management service: it
// loads configuration, wires the hexagonal layers, starts the HTTP server
// and the user-count worker, and shuts everything down gracefully.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"github.com/oat431/backend-challenge/internal/application"
	"github.com/oat431/backend-challenge/internal/infrastructure/auth"
	"github.com/oat431/backend-challenge/internal/infrastructure/config"
	"github.com/oat431/backend-challenge/internal/infrastructure/httpapi"
	"github.com/oat431/backend-challenge/internal/infrastructure/logger"
	"github.com/oat431/backend-challenge/internal/infrastructure/mongodb"
	"github.com/oat431/backend-challenge/internal/worker"
)

const shutdownTimeout = 10 * time.Second

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	logg := logger.New()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ---- driven adapter: MongoDB ----
	client, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		logg.Error("mongo connect failed", "error", err)
		os.Exit(1)
	}

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	if err := client.Ping(connectCtx, readpref.Primary()); err != nil {
		cancel()
		logg.Error("mongo ping failed", "error", err)
		os.Exit(1)
	}
	cancel()

	repo := mongodb.NewUserRepository(client.Database(cfg.DBName))
	idxCtx, cancelIdx := context.WithTimeout(ctx, 10*time.Second)
	if err := repo.EnsureIndexes(idxCtx); err != nil {
		cancelIdx()
		logg.Error("ensure indexes failed", "error", err)
		os.Exit(1)
	}
	cancelIdx()

	// ---- application core ----
	hasher := auth.NewBcryptHasher()
	tokens := auth.NewJWTManager([]byte(cfg.JWTSecret))
	users := application.NewUserService(repo, hasher)
	authSvc := application.NewAuthService(repo, hasher, tokens, cfg.TokenTTL)

	// ---- background worker (challenge requirement 6) ----
	worker := worker.NewUserCountWorker(repo, logg)
	go worker.Run(ctx, cfg.WorkerInterval)

	// ---- driving adapter: HTTP ----
	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           httpapi.NewRouter(logg, users, authSvc),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		logg.Info("http server listening", "port", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	logg.Info("service started",
		"db", cfg.DBName,
		"http_port", cfg.HTTPPort,
		"worker_interval", cfg.WorkerInterval.String(),
	)

	// ---- graceful shutdown ----
	select {
	case err := <-serverErr:
		logg.Error("http server failed", "error", err)
		stop()
	case <-ctx.Done():
		logg.Info("shutdown signal received")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logg.Error("http shutdown failed", "error", err)
	}

	// ctx cancellation also stops the worker goroutine.
	stop()

	disconnectCtx, cancelDisc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelDisc()
	if err := client.Disconnect(disconnectCtx); err != nil {
		logg.Error("mongo disconnect failed", "error", err)
	}

	logg.Info("shutdown complete")
}

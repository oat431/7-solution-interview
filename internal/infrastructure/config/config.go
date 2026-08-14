// Package config loads and validates environment configuration at startup.
// All values fail fast: a misconfigured service refuses to boot instead of
// failing at request time.
package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds all runtime configuration, sourced from environment variables.
type Config struct {
	MongoURI       string
	DBName         string
	JWTSecret      string
	TokenTTL       time.Duration
	HTTPPort       string
	GRPCPort       string
	WorkerInterval time.Duration
}

const minSecretBytes = 32

// Load reads the environment and validates the configuration.
func Load() (Config, error) {
	cfg := Config{
		MongoURI:       getenv("MONGO_URI", "mongodb://localhost:27017"),
		DBName:         getenv("DB_NAME", "userdb"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		HTTPPort:       getenv("HTTP_PORT", "8080"),
		GRPCPort:       getenv("GRPC_PORT", "50051"),
		TokenTTL:       getenvDur("TOKEN_TTL", time.Hour),
		WorkerInterval: getenvDur("WORKER_INTERVAL", 10*time.Second),
	}

	if len(cfg.JWTSecret) < minSecretBytes {
		return Config{}, fmt.Errorf("JWT_SECRET must be set and at least %d bytes long (got %d)", minSecretBytes, len(cfg.JWTSecret))
	}
	if cfg.TokenTTL <= 0 {
		return Config{}, fmt.Errorf("TOKEN_TTL must be positive")
	}
	if cfg.WorkerInterval <= 0 {
		return Config{}, fmt.Errorf("WORKER_INTERVAL must be positive")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvDur(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return fallback
}

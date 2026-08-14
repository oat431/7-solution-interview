package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadValidConfig(t *testing.T) {
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("MONGO_URI", "mongodb://localhost:27017")
	t.Setenv("TOKEN_TTL", "30m")
	t.Setenv("WORKER_INTERVAL", "5s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.DBName != "userdb" {
		t.Fatalf("expected default db name, got %q", cfg.DBName)
	}
	if cfg.HTTPPort != "8080" || cfg.GRPCPort != "50051" {
		t.Fatalf("unexpected ports: %s/%s", cfg.HTTPPort, cfg.GRPCPort)
	}
	if cfg.TokenTTL != 30*time.Minute {
		t.Fatalf("unexpected ttl: %v", cfg.TokenTTL)
	}
	if cfg.WorkerInterval != 5*time.Second {
		t.Fatalf("unexpected worker interval: %v", cfg.WorkerInterval)
	}
}

func TestLoadMissingSecretFails(t *testing.T) {
	t.Setenv("JWT_SECRET", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("expected JWT_SECRET error, got %v", err)
	}
}

func TestLoadShortSecretFails(t *testing.T) {
	t.Setenv("JWT_SECRET", "too-short")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "at least 32") {
		t.Fatalf("expected short-secret error, got %v", err)
	}
}

func TestLoadInvalidDurationsFallBack(t *testing.T) {
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("TOKEN_TTL", "not-a-duration")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.TokenTTL != time.Hour {
		t.Fatalf("expected default ttl on bad input, got %v", cfg.TokenTTL)
	}
}

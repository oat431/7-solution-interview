// Package logger sets up the structured application logger (stdlib slog).
package logger

import (
	"log/slog"
	"os"
)

// New returns a JSON slog.Logger writing to stdout, the idiomatic Go
// structured-logging setup with zero third-party dependencies.
func New() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

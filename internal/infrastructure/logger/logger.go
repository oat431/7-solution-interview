// Package logger sets up the structured application logger (stdlib slog).
package logger

import (
	"log/slog"
	"os"
)

// New returns a JSON slog.Logger writing to stdout.
func New() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

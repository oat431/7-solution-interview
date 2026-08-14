// Package worker contains background jobs.
package worker

import (
	"context"
	"log/slog"
	"time"
)

// Counter is the narrow dependency the worker needs from persistence.
type Counter interface {
	Count(ctx context.Context) (int64, error)
}

type UserCountWorker struct {
	counter Counter
	log     *slog.Logger
}

func NewUserCountWorker(counter Counter, log *slog.Logger) *UserCountWorker {
	return &UserCountWorker{counter: counter, log: log}
}

// Run logs total_users every interval until ctx is cancelled. It is a
// blocking call — start it in a goroutine owned by main.
func (w *UserCountWorker) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("user count worker stopped")
			return
		case <-ticker.C:
			n, err := w.counter.Count(ctx)
			if err != nil {
				w.log.Error("user count query failed", "error", err)
				continue
			}
			w.log.Info("user count", "total_users", n)
		}
	}
}

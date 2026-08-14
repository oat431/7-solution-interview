package worker

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

type counterStub struct {
	mu sync.Mutex
	n  int64
}

func (c *counterStub) Count(_ context.Context) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n, nil
}

func TestWorkerLogsCountPeriodically(t *testing.T) {
	counter := &counterStub{n: 42}
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	w := NewUserCountWorker(counter, log)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		w.Run(ctx, 20*time.Millisecond)
		close(done)
	}()

	// Let several ticks fire, then cancel.
	time.Sleep(70 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not stop after context cancel")
	}

	out := buf.String()
	if !strings.Contains(out, "total_users") {
		t.Fatalf("expected total_users log line, got: %s", out)
	}
}

func TestWorkerStopsImmediatelyWhenContextAlreadyCancelled(t *testing.T) {
	counter := &counterStub{}
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	w := NewUserCountWorker(counter, log)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before starting

	done := make(chan struct{})
	go func() {
		w.Run(ctx, time.Hour)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker should exit immediately on cancelled context")
	}
}

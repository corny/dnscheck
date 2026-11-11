package main

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// every runs fn immediately and then every interval until ctx is done.
// Errors are logged and do not stop the loop. wg is done when the loop has
// stopped, so the caller can wait for a running fn on shutdown.
func every(ctx context.Context, wg *sync.WaitGroup, interval time.Duration, name string, fn func() error) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			if err := fn(); err != nil {
				slog.Error("task failed", "task", name, "error", err)
			}

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

package main

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEvery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var (
		wg    sync.WaitGroup
		calls atomic.Int32
	)

	every(ctx, &wg, 10*time.Millisecond, "test", func() error {
		calls.Add(1)
		return errors.New("failures do not stop the loop")
	})

	assert.Eventually(t, func() bool { return calls.Load() >= 3 }, time.Second, time.Millisecond)

	cancel()
	wg.Wait()
	stopped := calls.Load()
	time.Sleep(30 * time.Millisecond)
	assert.Equal(t, stopped, calls.Load(), "no calls after cancel")
}

func TestEveryRunsImmediately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	called := make(chan struct{})

	every(ctx, &wg, time.Hour, "test", func() error {
		close(called)
		return nil
	})

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("not called before the first interval")
	}
}

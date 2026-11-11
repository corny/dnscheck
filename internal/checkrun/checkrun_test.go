package checkrun

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/corny/dnscheck/internal/check"
)

// fakeChecker blocks in Enqueue until release is closed, like a full job
// queue, and records whether Enqueue returned after Stop.
type fakeChecker struct {
	entered     chan struct{} // closed on the first Enqueue
	enteredOnce sync.Once
	release     chan struct{}
	results     chan *check.Job

	mu               sync.Mutex
	stopped          bool
	enqueueAfterStop bool
}

func newFakeChecker() *fakeChecker {
	return &fakeChecker{
		entered: make(chan struct{}),
		release: make(chan struct{}),
		results: make(chan *check.Job),
	}
}

func (*fakeChecker) ReadDomains(string) error { return nil }
func (*fakeChecker) Start() error             { return nil }

func (f *fakeChecker) Stop() {
	f.mu.Lock()
	f.stopped = true
	f.mu.Unlock()
	close(f.results)
}

func (f *fakeChecker) Enqueue(int, net.IP) {
	f.enteredOnce.Do(func() { close(f.entered) })
	<-f.release

	// the real checker panics here with "send on closed channel"
	f.mu.Lock()
	if f.stopped {
		f.enqueueAfterStop = true
	}
	f.mu.Unlock()
}

func (f *fakeChecker) Results() <-chan *check.Job { return f.results }

type fakeStore struct{}

func (fakeStore) EachAddress(_ int, fn func(int, net.IP) bool) error {
	for id := 1; id <= 10; id++ {
		if !fn(id, net.IPv4(192, 0, 2, byte(id))) {
			return nil
		}
	}
	return nil
}

func (fakeStore) SaveResult(*check.Job) error { return nil }

func TestStopWhileEnqueueing(t *testing.T) {
	checker := newFakeChecker()
	runner := New(checker, fakeStore{}, "", time.Hour, false)

	runDone := make(chan error, 1)
	go func() { runDone <- runner.Run() }()
	<-checker.entered

	stopDone := make(chan struct{})
	go func() {
		runner.Stop()
		close(stopDone)
	}()

	// give Stop the chance to stop the checker while Enqueue is blocked
	time.Sleep(100 * time.Millisecond)
	close(checker.release)

	for _, done := range []<-chan struct{}{stopDone, wait(runDone)} {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("runner does not shut down")
		}
	}

	assert.False(t, checker.enqueueAfterStop, "checker stopped while a job was being enqueued")
}

func wait(ch <-chan error) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		<-ch
		close(done)
	}()
	return done
}

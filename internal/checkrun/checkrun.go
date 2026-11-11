// Package checkrun periodically checks all nameservers and stores the results.
package checkrun

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/corny/dnscheck/internal/check"
)

const batchSize = 1000

// Checker checks nameservers, implemented by *check.Checker.
type Checker interface {
	ReadDomains(path string) error
	Start() error
	Stop()
	Enqueue(id int, address net.IP)
	Results() <-chan *check.Job
}

// Store provides the nameservers and saves the results, implemented by
// *store.Store.
type Store interface {
	EachAddress(batchSize int, fn func(id int, address net.IP) bool) error
	SaveResult(job *check.Job) error
}

// Runner checks all nameservers of the store in a fixed interval.
type Runner struct {
	checker  Checker
	store    Store
	domains  string // path to the domain list
	interval time.Duration
	debug    bool

	shutdown chan struct{} // closed by Stop
	done     chan struct{} // closed when Run returns
	finished sync.WaitGroup
}

// New creates a Runner. domains is the path to the domain list.
func New(checker Checker, store Store, domains string, interval time.Duration, debug bool) *Runner {
	return &Runner{
		checker:  checker,
		store:    store,
		domains:  domains,
		interval: interval,
		debug:    debug,
		shutdown: make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Run starts the checker and checks all nameservers every interval until
// Stop is called. Before returning, it stops the checker and waits until all
// results are written.
func (r *Runner) Run() error {
	defer close(r.done)

	if err := r.checker.ReadDomains(r.domains); err != nil {
		return fmt.Errorf("unable to read domain list: %w", err)
	}

	if err := r.checker.Start(); err != nil {
		return fmt.Errorf("unable to start checker: %w", err)
	}

	r.finished.Add(1)
	go r.writeResults()

	// runs after the loop below has returned, so no job is enqueued anymore
	defer func() {
		r.checker.Stop()
		r.finished.Wait()
	}()

	timer := time.NewTimer(r.interval)
	for {
		r.createJobs()

		select {
		case <-r.shutdown:
			return nil
		case <-timer.C:
			timer.Reset(r.interval)
		}
	}
}

// Stop stops the checks and waits until Run has returned.
func (r *Runner) Stop() {
	close(r.shutdown)
	<-r.done
}

func (r *Runner) createJobs() {
	log.Println("create jobs")

	err := r.store.EachAddress(batchSize, func(id int, address net.IP) bool {
		select {
		case <-r.shutdown:
			return false
		default:
			r.checker.Enqueue(id, address)
			return true
		}
	})
	if err != nil {
		log.Fatal(err)
	}
}

func (r *Runner) writeResults() {
	defer r.finished.Done()

	for job := range r.checker.Results() {
		if r.debug {
			log.Println(job)
		}

		if err := r.store.SaveResult(job); err != nil {
			log.Println(err)
		}
	}
}

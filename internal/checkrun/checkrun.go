// Package checkrun periodically checks all nameservers and stores the results.
package checkrun

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/corny/dnscheck/internal/check"
	"github.com/corny/dnscheck/internal/store"
)

const batchSize = 1000

// Runner checks all nameservers of the store in a fixed interval.
type Runner struct {
	checker  *check.Checker
	store    *store.Store
	domains  string // path to the domain list
	interval time.Duration
	debug    bool

	shutdown chan struct{}
	finished sync.WaitGroup
}

// New creates a Runner. domains is the path to the domain list.
func New(checker *check.Checker, store *store.Store, domains string, interval time.Duration, debug bool) *Runner {
	return &Runner{
		checker:  checker,
		store:    store,
		domains:  domains,
		interval: interval,
		debug:    debug,
		shutdown: make(chan struct{}),
	}
}

// Run starts the checker and checks all nameservers every interval until
// Stop is called.
func (r *Runner) Run() error {
	if err := r.checker.ReadDomains(r.domains); err != nil {
		return fmt.Errorf("unable to read domain list: %w", err)
	}

	if err := r.checker.Start(); err != nil {
		return fmt.Errorf("unable to start checker: %w", err)
	}

	r.finished.Add(1)
	go r.writeResults()

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

// Stop stops the checks and waits until all results are written.
func (r *Runner) Stop() {
	close(r.shutdown)
	r.checker.Stop()
	r.finished.Wait()
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

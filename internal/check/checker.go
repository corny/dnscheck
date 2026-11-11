package check

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/corny/dnscheck/internal/geoip"
	"github.com/miekg/dns"
)

// Checker is a checker instance
type Checker struct {
	WorkersCount    uint
	ReferenceServer string
	MaxAttempts     uint
	Timeout         time.Duration
	GeoDbPathCity   string
	GeoDbPathASN    string

	// map of checked domains and their results from the reference server
	expectedResults resultMap

	// domains to be checked
	domains []string

	DNSClient dns.Client

	geoipCity *geoip.Database
	geoipASN  *geoip.Database

	pending   chan *Job
	finished  chan *Job
	pendingWg sync.WaitGroup

	ready atomic.Bool // set when Start has succeeded
}

// Ready tells whether Start has succeeded, so jobs can be executed.
func (checker *Checker) Ready() bool {
	return checker.ready.Load()
}

// Domains returns the domains to be checked.
func (checker *Checker) Domains() []string {
	return slices.Clone(checker.domains)
}

// Start checks the options and starts the check routines
func (checker *Checker) Start() error {

	if checker.ReferenceServer == "" {
		return errors.New("reference nameserver missing")
	}
	if err := checker.UpdateExectations(); err != nil {
		return errors.New("unable to query reference nameserver")
	}

	if path := checker.GeoDbPathCity; path != "" {
		db, err := geoip.New(path)
		if err != nil {
			return err
		}
		checker.geoipCity = db
	}

	if path := checker.GeoDbPathASN; path != "" {
		db, err := geoip.New(path)
		if err != nil {
			return err
		}
		checker.geoipASN = db
	}

	checker.pending = make(chan *Job, 100)
	checker.finished = make(chan *Job, 100)

	// Start workers
	checker.pendingWg.Add(int(checker.WorkersCount))
	for i := uint(0); i < checker.WorkersCount; i++ {
		go checker.worker()
	}

	checker.ready.Store(true)
	return nil
}

// Stop closes input channel and waits for workers to finish
func (checker *Checker) Stop() {
	close(checker.pending)

	if checker.geoipCity != nil {
		checker.geoipCity.Close()
	}

	if checker.geoipASN != nil {
		checker.geoipASN.Close()
	}

	checker.pendingWg.Wait()
	close(checker.finished)
}

// UpdateExectations checks the domain list against the references nameserver and saves the responses
func (checker *Checker) UpdateExectations() error {
	res, _, err := checker.resolveDomains(checker.ReferenceServer)
	if err != nil {
		return fmt.Errorf("error resolving domains from reference server: %w", err)
	}
	checker.expectedResults = res
	return nil
}

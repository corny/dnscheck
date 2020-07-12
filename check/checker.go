package check

import (
	"sync"
	"time"

	"github.com/corny/dnscheck/geoip"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
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
		return errors.Wrapf(err, "error resolving domains from reference server")
	}
	checker.expectedResults = res
	return nil
}

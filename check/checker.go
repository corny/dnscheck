package check

import (
	"sync"
	"time"

	"github.com/miekg/dns"
	geoip2 "github.com/oschwald/geoip2-golang"
	"github.com/pkg/errors"
)

// Checker is a checker instance
type Checker struct {
	WorkersCount    uint
	ReferenceServer string
	MaxAttempts     uint
	Timeout         time.Duration
	GeoDbPath       string

	// map of checked domains and their results from the reference server
	expectedResults resultMap

	// domains to be checked
	domains []string

	DNSClient dns.Client

	geoip     *geoip2.Reader
	pending   chan *Job
	finished  chan *Job
	pendingWg sync.WaitGroup
}

// Start checks the options and starts the check routines
func (checker *Checker) Start() error {

	if checker.ReferenceServer == "" {
		return errors.New("reference nameserver missing")
	}
	if checker.GeoDbPath == "" {
		return errors.New("GeoDbPath missing")
	}

	// Get results from the reference nameserver
	res, _, err := checker.resolveDomains(checker.ReferenceServer)
	if err != nil {
		return errors.Wrapf(err, "error resolving domains from reference server")
	}
	checker.expectedResults = res

	// Open the GeoDB
	geoip, err := geoip2.Open(checker.GeoDbPath)
	if err != nil {
		return err
	}

	checker.geoip = geoip
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
	checker.pendingWg.Wait()
	close(checker.finished)
}

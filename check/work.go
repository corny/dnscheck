package check

import (
	"log"
	"net"
)

// Enqueue adds a new job
func (checker *Checker) Enqueue(id int, address string) {
	checker.pending <- &Job{ID: id, Address: address}
}

// Results returns the results channel
func (checker *Checker) Results() <-chan *Job {
	return checker.finished
}

func (checker *Checker) worker() {
	for job := range checker.pending {
		checker.executeJob(job)
		checker.finished <- job

		incrementMetric(&Metrics.Processed)
	}
	checker.pendingWg.Done()
}

// consumes a job and writes the result in the given job
func (checker *Checker) executeJob(job *Job) {
	// GeoDB lookup
	var err error

	job.Country, job.City, err = checker.geoip.City(net.ParseIP(job.Address))
	if err != nil {
		log.Printf("cannot resolve IP address to location %v: %s", job.Address, err)
		return
	}

	// Run the check
	dnssec, err := checker.check(job)
	job.Name = checker.ptrName(job.Address)

	// query the bind version
	if err == nil || err.Error() != "i/o timeout" {
		job.Version = checker.version(job.Address)
	}

	if err == nil {
		job.State = "valid"
		job.Err = ""
		job.Dnssec = &dnssec

		if dnssec {
			incrementMetric(&Metrics.DNSSecSupported)
		} else {
			incrementMetric(&Metrics.DNSSecNotSupported)
		}

		incrementMetric(&Metrics.Valid)
	} else {
		job.State = "invalid"
		job.Err = err.Error()

		incrementMetric(&Metrics.Invalid)
	}
}

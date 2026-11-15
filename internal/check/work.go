package check

import (
	"net"
)

// Enqueue adds a new job
func (checker *Checker) Enqueue(id int, address net.IP) {
	checker.pending <- &Job{ID: id, Address: address}
}

// Results returns the results channel
func (checker *Checker) Results() <-chan *Job {
	return checker.finished
}

func (checker *Checker) worker() {
	for job := range checker.pending {
		checker.ExecuteJob(job)
		checker.finished <- job

		Metrics.Processed.Add(1)
	}
	checker.pendingWg.Done()
}

// consumes a job and writes the result in the given job
func (checker *Checker) ExecuteJob(job *Job) {
	// GeoDB lookup
	var err error

	if checker.geoipCity != nil {
		job.City, _ = checker.geoipCity.City(job.Address)
	}
	if checker.geoipASN != nil {
		job.ASN, _ = checker.geoipASN.ASN(job.Address)
	}

	// Run the check
	dnssec, err := checker.check(job)
	job.Name = checker.ptrName(job.Address.String())

	// query the bind version
	if err == nil || err.Error() != "i/o timeout" {
		job.Version = checker.version(job.Address.String())
	}

	if err == nil {
		job.Status = true
		job.Error = ""
		job.Dnssec = &dnssec

		if dnssec {
			Metrics.DNSSecSupported.Add(1)
		} else {
			Metrics.DNSSecNotSupported.Add(1)
		}

		Metrics.Valid.Add(1)
	} else {
		job.Status = false
		job.Error = err.Error()

		Metrics.Invalid.Add(1)
	}
}

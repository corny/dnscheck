package check

import "sync/atomic"

// Metrics holds runtime metrics
var Metrics struct {
	Queries uint64 // total amount of DNS queries

	Processed uint64 // total checks done
	Valid     uint64 // total checks with valid result
	Invalid   uint64 // total checks with invalid result

	DNSSecSupported    uint64 // valid results with dnssec
	DNSSecNotSupported uint64 // valid results without dnssec
}

func incrementMetric(ptr *uint64) {
	atomic.AddUint64(ptr, 1)
}

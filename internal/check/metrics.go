package check

import "sync/atomic"

// Metrics holds runtime metrics
var Metrics struct {
	Queries   atomic.Uint64 // total amount of DNS queries
	Processed atomic.Uint64 // total checks done
	Valid     atomic.Uint64 // total checks with valid result
	Invalid   atomic.Uint64 // total checks with invalid result
}

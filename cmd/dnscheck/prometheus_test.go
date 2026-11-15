package main

import (
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"

	"github.com/corny/dnscheck/internal/check"
)

// TestMetricsConcurrent fails with -race if the metrics are read without
// synchronization while checks are running.
func TestMetricsConcurrent(t *testing.T) {
	checker := check.Checker{ReferenceServer: "192.0.2.1"}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		checker.ExecuteJob(&check.Job{}) // increments the metrics
	}()

	ch := make(chan prometheus.Metric, 10)
	(&metricsExporter{}).Collect(ch)
	wg.Wait()
	close(ch)

	assert.Len(t, ch, 4)
}

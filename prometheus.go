package main

import (
	"log"
	"net/http"

	"github.com/corny/dnscheck/check"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metricsPrefix = "dnscheck_"
	checksDesc    = prometheus.NewDesc(metricsPrefix+"processed", "Total number of total checks", nil, nil)
	resultDesc    = prometheus.NewDesc(metricsPrefix+"result", "Result of checks", []string{"result"}, nil)
	queriesDesc   = prometheus.NewDesc(metricsPrefix+"queries", "Total number of DNS queries", nil, nil)
)

func startMetrics() {
	prometheus.MustRegister(&metricsExporter{})
	http.Handle(*metricsPath, promhttp.Handler())
	log.Fatal(http.ListenAndServe(*metricsListenAddress, nil))
}

type metricsExporter struct{}

// Describe describes metrics for Prometheus
func (ex *metricsExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- checksDesc
	ch <- queriesDesc
	ch <- resultDesc
}

// Collect collects metrics for Prometheus
func (ex *metricsExporter) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(checksDesc, prometheus.CounterValue, float64(check.Metrics.Processed))
	ch <- prometheus.MustNewConstMetric(queriesDesc, prometheus.CounterValue, float64(check.Metrics.Queries))
	ch <- prometheus.MustNewConstMetric(resultDesc, prometheus.CounterValue, float64(check.Metrics.Valid), "valid")
	ch <- prometheus.MustNewConstMetric(resultDesc, prometheus.CounterValue, float64(check.Metrics.Invalid), "invalid")
}

package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/corny/dnscheck/internal/web"
)

// startHTTP serves the web pages, the API and the metrics.
func startHTTP(config web.Config) {
	srv, err := web.New(config)
	if err != nil {
		fatal("cannot load web pages", "error", err)
	}

	prometheus.MustRegister(&metricsExporter{})

	mux := http.NewServeMux()
	mux.Handle(*metricsPath, promhttp.Handler())
	mux.Handle("/", srv.Handler())

	server := &http.Server{
		Addr:              *listenAddress,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      5 * time.Minute, // a submitted nameserver is checked synchronously
		IdleTimeout:       2 * time.Minute,
	}

	slog.Info("listening", "address", *listenAddress)
	go func() {
		fatal("HTTP server stopped", "error", server.ListenAndServe())
	}()
}

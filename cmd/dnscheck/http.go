package main

import (
	"log"
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
		log.Fatalln("cannot load web pages:", err)
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

	log.Println("listening on", *listenAddress)
	go func() {
		log.Fatal(server.ListenAndServe())
	}()
}

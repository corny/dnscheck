package main

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"

	"github.com/corny/dnscheck/internal/bogon"
	"github.com/corny/dnscheck/internal/check"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
)

var errAlreadyExists = errors.New("already exists")

func startHTTP() {
	prometheus.MustRegister(&metricsExporter{})

	c := cors.New(cors.Options{
		AllowedOrigins: *allowOrigins,
	})

	http.Handle("/api/nameservers", c.Handler(submitHandler{}))
	http.Handle(*metricsPath, promhttp.Handler())

	log.Println("listening on", *listenAddress)
	go func() {
		log.Fatal(http.ListenAndServe(*listenAddress, nil))
	}()
	log.Println("started")
}

type submitHandler struct{}

func (submitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	addrStr := r.FormValue("address")
	if addrStr == "" {
		http.Error(w, "address parameter is missing or empty", http.StatusUnprocessableEntity)
		return
	}

	address := net.ParseIP(addrStr)
	if address == nil {
		http.Error(w, "invalid IP address", http.StatusUnprocessableEntity)
		return
	}

	if bogon.IsBogon(address) {
		http.Error(w, "bogon IP address", http.StatusUnprocessableEntity)
		return
	}

	job, err := addNameserver(address)

	switch err {
	case errAlreadyExists:
		http.Error(w, err.Error(), http.StatusConflict)
	case nil:
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(&job)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func addNameserver(address net.IP) (job check.Job, err error) {
	job.Address = address

	_, found, err := db.FindID(address)
	if err != nil {
		return job, err
	}
	if found {
		return job, errAlreadyExists
	}

	checker.ExecuteJob(&job)
	if job.Error != "" {
		return job, errors.New(job.Error)
	}

	return job, db.SaveResult(&job)
}

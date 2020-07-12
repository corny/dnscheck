package main

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/corny/dnscheck/check"
	"github.com/pkg/errors"
)

var (
	shutdown   = make(chan struct{})
	finishedWg = sync.WaitGroup{}
)

func startChecks() error {
	// read domain list
	if err := checker.ReadDomains(*domains); err != nil {
		return errors.Wrap(err, "unable to read domain list")
	}

	if err := checker.Start(); err != nil {
		return errors.Wrap(err, "unable to start checker")
	}

	// Start result writer
	finishedWg.Add(1)
	go resultWriter()

	timer := time.NewTimer(*checkInterval)
	for {
		createJobs()

		select {
		case <-shutdown:
			goto abort
		case <-timer.C:
			timer.Reset(*checkInterval)
		}
	}

abort:
	return nil
}

func stopChecks() {
	close(shutdown)
	checker.Stop()
	finishedWg.Wait()
}

func createJobs() {
	log.Println("create jobs")

	id := 0
	batchSize := 1000
	found := batchSize

	for batchSize == found {
		// Read the next batch
		rows, err := dbConn.Query("SELECT id, ip_address FROM nameservers WHERE id > $1 LIMIT $2", id, batchSize)
		if err != nil {
			log.Fatalf("select batch failed: %v", err)
		}

		found = 0
		for rows.Next() {
			var address string

			select {
			case <-shutdown:
				goto abort
			default:
				// get RawBytes from data
				err = rows.Scan(&id, &address)
				if err != nil {
					log.Fatalf("scanning DB values failed: %v", err)
				}

				checker.Enqueue(id, net.ParseIP(address))
				found++
			}
		}
	abort:
		rows.Close()
	}
}

func resultWriter() {
	for job := range checker.Results() {
		if *debug {
			log.Println(job)
		}

		err := insertOrUpdate(job)
		if err != nil {
			log.Println(err)
		}
	}

	finishedWg.Done()
}

// insert or update the record
func insertOrUpdate(job *check.Job) (err error) {
	args := []interface{}{job.Name, job.Status, job.Error, job.Version, job.Dnssec, nil, nil, nil, nil, nil}

	if job.City != nil {
		args[5] = job.City.Country.IsoCode
		args[6] = job.City.City.Names["en"]
	}

	if job.ASN != nil {
		args[7] = job.ASN.AutonomousSystemNumber
		args[8] = job.ASN.AutonomousSystemOrganization
	}

	if job.ID > 0 {
		args[len(args)-1] = job.ID
		_, err = dbConn.Exec("UPDATE nameservers SET name=$1, status=$2, error=$3, version=$4, dnssec=$5, country_code=$6, city=$7, as_number=$8, as_org=$9, checked_at = NOW() at time zone 'utc' WHERE id=$10", args...)
	} else {
		args[len(args)-1] = job.Address.String()
		_, err = dbConn.Exec("INSERT INTO nameservers (name, status, error, version, dnssec, country_code, city, as_number, as_org, ip_address, reliability, checked_at, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 1, NOW() at time zone 'utc', NOW() at time zone 'utc')", args...)
	}

	return
}

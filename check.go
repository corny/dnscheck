package main

import (
	"log"

	"github.com/pkg/errors"
)

func runCheck() error {
	if metricsListenAddress != nil {
		go startMetrics()
	}

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

	createJobs()
	checker.Stop()
	finishedWg.Wait()

	return nil
}

func createJobs() {
	id := 0
	batchSize := 1000
	found := batchSize

	for batchSize == found {
		// Read the next batch
		rows, err := dbConn.Query("SELECT id, ip FROM nameservers WHERE id > ? LIMIT ?", id, batchSize)
		if err != nil {
			log.Fatalf("select batch failed: %v", err)
		}

		found = 0
		for rows.Next() {
			var address string

			// get RawBytes from data
			err = rows.Scan(&id, &address)
			if err != nil {
				log.Fatalf("scanning DB values failed: %v", err)
			}

			checker.Enqueue(id, address)
			found++
		}
		rows.Close()
	}
}

func resultWriter() {
	stm, err := dbConn.Prepare("UPDATE nameservers SET name=?, state=?, error=?, version=?, dnssec=?, checked_at=NOW(), country_id=?, city=? WHERE id=?")
	if err != nil {
		log.Fatalf("prepare statement failed: %v", err)
	}
	defer stm.Close()

	for res := range checker.Results() {
		if *debug {
			log.Println(res)
		}
		stm.Exec(res.Name, res.State, res.Err, res.Version, res.Dnssec, res.Country, res.City, res.ID)
	}

	finishedWg.Done()
}

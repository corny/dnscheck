package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/corny/dnscheck"
	"github.com/corny/dnscheck/check"
	_ "github.com/go-sql-driver/mysql"
)

const (
	// Timeout for DNS queries
	timeout = 3 * 1e9

	// maximum number of attempts for a query
	maxAttempts = 3
)

var (
	pending      = make(chan *dnscheck.Nameserver, 100)
	finished     = make(chan *dnscheck.Nameserver, 100)
	pendingWg    sync.WaitGroup
	finishedWg   sync.WaitGroup
	workersCount = 32
	driver, dsn  string
	domainArg    string
)

func main() {
	databaseArg := flag.String("database", "database.yml", "Path to file containing the database configuration")
	flag.StringVar(&domainArg, "domains", "domains.txt", "Path to file containing the domain list")
	flag.StringVar(&check.GeoDbPath, "geodb", "GeoLite2-City.mmdb", "Path to GeoDB database")
	flag.StringVar(&check.Reference, "reference", check.Reference, "The nameserver that every other is compared with")
	flag.IntVar(&workersCount, "workers", workersCount, "Number of worker routines")
	flag.Parse()

	check.DNSClient.ReadTimeout = timeout
	check.MaxAttempts = maxAttempts

	environment := os.Getenv("RAILS_ENV")
	if environment == "" {
		environment = "development"
	}

	// read domain list
	if err := check.ReadDomains(domainArg); err != nil {
		fmt.Println("unable to read domain list")
		panic(err)
	}

	// load database configuration
	driver, dsn = dnscheck.RailsConfigToDSN(*databaseArg, environment)

	// check the GeoDB
	check.GeoLocate(check.Reference)

	// Use all cores

	// Get results from the reference nameserver
	res, _, err := check.ResolveDomains(check.Reference)
	if err != nil {
		panic(err)
	}
	check.Expectations = res

	// Start result writer
	finishedWg.Add(1)
	go resultWriter()

	// Start workers
	pendingWg.Add(workersCount)
	for i := 0; i < workersCount; i++ {
		go worker()
	}

	createJobs()

	// wait for workers to finish
	pendingWg.Wait()

	close(finished)
	finishedWg.Wait()
}

func createJobs() {
	currentID := 0
	batchSize := 1000
	found := batchSize

	// Open SQL connection
	db, err := sql.Open(driver, dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	for batchSize == found {
		// Read the next batch
		rows, err := db.Query("SELECT id, ip FROM nameservers WHERE id > ? LIMIT ?", currentID, batchSize)
		if err != nil {
			panic(err)
		}

		found = 0
		for rows.Next() {
			j := new(dnscheck.Nameserver)

			// get RawBytes from data
			err = rows.Scan(&j.ID, &j.Address)
			if err != nil {
				panic(err)
			}
			pending <- j
			currentID = j.ID
			found++
		}
		if err = rows.Close(); err != nil {
			panic(err)
		}
	}
	close(pending)
}

func worker() {
	for job := range pending {
		executeJob(job)
		finished <- job
	}
	pendingWg.Done()
}

func resultWriter() {
	// Open SQL connection
	db, err := sql.Open(driver, dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	stm, err := db.Prepare("UPDATE nameservers SET name=?, state=?, error=?, version=?, dnssec=?, checked_at=NOW(), country_id=?, city=? WHERE id=?")
	if err != nil {
		panic(err)
	}
	defer stm.Close()

	for res := range finished {
		log.Println(res)
		_, err = stm.Exec(res.Name, res.State, res.Error, res.Version, res.DNSSEC, res.Country, res.City, res.ID)
		if err != nil {
			panic(err)
		}
	}

	finishedWg.Done()
}

// consumes a job and writes the result in the given job
func executeJob(job *dnscheck.Nameserver) {
	// log.Println("received job", job.id)

	// GeoDB lookup
	job.Country, job.City = check.GeoLocate(job.Address)

	// Run the check
	dnssec, err := check.Run(job)
	job.Name = check.PtrName(job.Address)

	// query the bind version
	if err == nil || err.Error() != "i/o timeout" {
		job.Version = check.Version(job.Address)
	}

	if err == nil {
		job.State = "valid"
		job.Error = ""
		job.DNSSEC = &dnssec
	} else {
		job.State = "invalid"
		job.Error = err.Error()
	}
}

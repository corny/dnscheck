package main

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"sync"

	_ "github.com/go-sql-driver/mysql"
)

const (
	// Timeout for DNS queries
	timeout = 3 * 1e9

	// maximum number of attempts for a query
	maxAttempts = 3
)

var (
	pending         = make(chan *job, 100)
	finished        = make(chan *job, 100)
	pendingWg       sync.WaitGroup
	finishedWg      sync.WaitGroup
	workersCount    = 32
	referenceServer = "8.8.8.8"
	connection      string
	domainArg       string
	verbose         bool
	syslog          bool
)

func main() {
	databaseArg := flag.String("database", "database.yml", "Path to file containing the database configuration")
	flag.StringVar(&domainArg, "domains", "domains.txt", "Path to file containing the domain list")
	flag.StringVar(&geoDbPath, "geodb", "GeoLite2-City.mmdb", "Path to GeoDB database")
	flag.StringVar(&referenceServer, "reference", referenceServer, "The nameserver that every other is compared with")
	flag.IntVar(&workersCount, "workers", workersCount, "Number of worker routines")
	flag.BoolVar(&verbose, "verbose", verbose, "Increase logging output")
	flag.BoolVar(&syslog, "syslog", syslog, "Prepare logging for syslog (print to stdout, no timestamps)")
	flag.Parse()

	if syslog {
		log.SetOutput(os.Stdout)
		log.SetFlags(0)
	}

	dnsClient.ReadTimeout = timeout

	environment := os.Getenv("RAILS_ENV")
	if environment == "" {
		environment = "development"
	}

	// read domain list
	if err := readDomains(domainArg); err != nil {
		log.Fatalf("unable to read domain list: %v", err)
	}

	// load database configuration
	connection = databasePath(*databaseArg, environment)

	// check the GeoDB
	location(referenceServer)

	// Get results from the reference nameserver
	res, _, err := resolveDomains(referenceServer)
	if err != nil {
		log.Fatalf("error resolving domains from reference server: %v", err)
	}
	expectedResults = res

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
	db, err := sql.Open("mysql", connection)
	if err != nil {
		log.Fatalf("cannot connect to database: %v", err)
	}
	defer db.Close()

	for batchSize == found {
		// Read the next batch
		rows, err := db.Query("SELECT id, ip FROM nameservers WHERE id > ? LIMIT ?", currentID, batchSize)
		if err != nil {
			log.Fatalf("select batch failed: %v", err)
		}

		found = 0
		for rows.Next() {
			j := new(job)

			// get RawBytes from data
			err = rows.Scan(&j.id, &j.address)
			if err != nil {
				log.Fatalf("scanning DB values failed: %v", err)
			}
			pending <- j
			currentID = j.id
			found++
		}
		rows.Close()
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
	db, err := sql.Open("mysql", connection)
	if err != nil {
		log.Fatalf("cannot connect to database: %v", err)
	}
	defer db.Close()

	stm, err := db.Prepare("UPDATE nameservers SET name=?, state=?, error=?, version=?, dnssec=?, checked_at=NOW(), country_id=?, city=? WHERE id=?")
	if err != nil {
		log.Fatalf("prepare statement failed: %v", err)
	}
	defer stm.Close()

	for res := range finished {
		if verbose {
			log.Println(res)
		}
		stm.Exec(res.name, res.state, res.err, res.version, res.dnssec, res.country, res.city, res.id)
	}

	finishedWg.Done()
}

// consumes a job and writes the result in the given job
func executeJob(job *job) {
	// GeoDB lookup
	job.country, job.city = location(job.address)

	// Run the check
	dnssec, err := check(job)
	job.name = ptrName(job.address)

	// query the bind version
	if err == nil || err.Error() != "i/o timeout" {
		job.version = version(job.address)
	}

	if err == nil {
		job.state = "valid"
		job.err = ""
		job.dnssec = &dnssec
	} else {
		job.state = "invalid"
		job.err = err.Error()
	}
}

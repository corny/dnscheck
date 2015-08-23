package main

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"os"
	"sync"
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
	done            sync.WaitGroup
	workersCount    = 32
	referenceServer = "8.8.8.8"
	connection      string
	domainArg       string
)

func main() {
	databaseArg := flag.String("database", "database.yml", "Path to file containing the database configuration")
	flag.StringVar(&domainArg, "domains", "domains.txt", "Path to file containing the domain list")
	flag.StringVar(&geoDbPath, "geodb", "GeoLite2-City.mmdb", "Path to GeoDB database")
	flag.StringVar(&referenceServer, "reference", referenceServer, "The nameserver that every other is compared with")
	flag.IntVar(&workersCount, "workers", workersCount, "Number of worker routines")
	flag.Parse()

	dnsClient.ReadTimeout = timeout

	environment := os.Getenv("RAILS_ENV")
	if environment == "" {
		environment = "development"
	}

	// read domain list
	if err := readDomains(domainArg); err != nil {
		fmt.Println("unable to read domain list")
		panic(err)
	}

	// load database configuration
	connection = databasePath(*databaseArg, environment)

	// check the GeoDB
	location(referenceServer)

	// Use all cores

	// Get results from the reference nameserver
	res, err := resolveDomains(referenceServer)
	if err != nil {
		panic(err)
	}
	expectedResults = res

	// Start result writer
	go resultWriter()

	// Start workers
	done.Add(workersCount)
	for i := 0; i < workersCount; i++ {
		go worker()
	}

	createJobs()

	// wait for workers to finish
	done.Wait()

	close(finished)
}

func createJobs() {
	currentId := 0
	batchSize := 1000
	found := batchSize

	// Open SQL connection
	db, err := sql.Open("mysql", connection)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	for batchSize == found {
		// Read the next batch
		rows, err := db.Query("SELECT id, ip FROM nameservers WHERE id > ? LIMIT ?", currentId, batchSize)
		if err != nil {
			panic(err)
		}

		found = 0
		for rows.Next() {
			j := new(job)

			// get RawBytes from data
			err = rows.Scan(&j.id, &j.address)
			if err != nil {
				panic(err)
			}
			pending <- j
			currentId = j.id
			found += 1
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
	done.Done()
}

func resultWriter() {
	// Open SQL connection
	db, err := sql.Open("mysql", connection)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	stm, err := db.Prepare(
		"UPDATE nameservers SET name=?, state=?, error=?, version=?, checked_at=NOW(), country_id=?, city=?," +
			"state_changed_at = (CASE WHEN ? != state THEN NOW() ELSE state_changed_at END )" +
			"WHERE id=?")
	defer stm.Close()

	for res := range finished {
		log.Println(res)
		stm.Exec(res.name, res.state, res.err, res.version, res.country, res.city, res.state, res.id)
	}
}

// consumes a job and writes the result in the given job
func executeJob(job *job) {
	// log.Println("received job", job.id)

	// GeoDB lookup
	job.country, job.city = location(job.address)

	// Run the check
	err := check(job)
	job.name = ptrName(job.address)

	// query the bind version
	if err == nil || err.Error() != "i/o timeout" {
		job.version = version(job.address)
	}

	if err == nil {
		job.state = "valid"
		job.err = ""
	} else {
		job.state = "invalid"
		job.err = err.Error()
	}
}

package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"runtime"
)

const connection = "root:@tcp(localhost:3306)/nameservers_development"
const referenceNameserver = "8.8.8.8"

var jobs = make(chan *job, 100)
var results = make(chan *result, 100)
var done = make(chan bool)
var workersLimit = 1

func main() {
	// Use all cores
	cpus := runtime.NumCPU()
	runtime.GOMAXPROCS(cpus)
	workersLimit = 8 * cpus

	// Get results from the reference nameserver
	res, err := resolveDomains(referenceNameserver)
	if err != nil {
		panic(err)
	}
	expectedResults = res

	go resultWriter()

	for i := 0; i < workersLimit; i++ {
		go worker()
	}

	createJobs()

	// wait for resultWriter to finish
	<-done
}

func createJobs() {
	currentId := 0
	batchSize := 1000

	// Open SQL connection
	db, err := sql.Open("mysql", connection)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	for {
		// Read the next batch
		rows, err := db.Query("SELECT id, ip FROM nameservers WHERE id > ? LIMIT ?", currentId, batchSize)
		if err != nil {
			panic(err)
		}

		found := 0
		for rows.Next() {
			j := new(job)

			// get RawBytes from data
			err = rows.Scan(&j.id, &j.address)
			if err != nil {
				panic(err)
			}
			jobs <- j
			currentId = j.id
			found += 1
		}
		rows.Close()

		// Last batch?
		if found < batchSize {
			close(jobs)
			return
		}
	}
}

func worker() {
	for {
		job := <-jobs
		if job != nil {
			// log.Println("received job", job.id)
			err := check(job)
			result := &result{id: job.id, name: ptrName(job.address)}

			if err == nil {
				result.state = "valid"
				result.err = ""
			} else {
				result.state = "invalid"
				result.err = err.Error()
			}
			results <- result

		} else {
			log.Println("received all jobs")
			results <- nil
			return
		}
	}
}

func resultWriter() {
	// Open SQL connection
	db, err := sql.Open("mysql", connection)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	stm, err := db.Prepare("UPDATE nameservers SET name=?, state=?, error=?, checked_at=NOW() WHERE id=?")
	defer stm.Close()

	doneCount := 0
	for doneCount < workersLimit {
		res := <-results
		// log.Println("finished job", res.id)
		if res == nil {
			doneCount++
			log.Println("worker terminated")
		} else {
			log.Printf("job id=%v state=%s name=%s err=%s", res.id, res.state, res.name, res.err)
			stm.Exec(res.name, res.state, res.err, res.id)
		}
	}
	done <- true
}

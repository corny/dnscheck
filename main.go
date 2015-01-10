package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"runtime"
)

const connection = "root:@tcp(localhost:3306)/nameservers_development"
const referenceNameserver = "8.8.8.8"

var pending = make(chan *job, 100)
var finished = make(chan *job, 100)
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
			pending <- j
			currentId = j.id
			found += 1
		}
		rows.Close()

		// Last batch?
		if found < batchSize {
			close(pending)
			return
		}
	}
}

func worker() {
	for {
		job := <-pending
		if job != nil {
			// log.Println("received job", job.id)
			err := check(job)
			job.name = ptrName(job.address)
			job.version = version(job.address)

			if err == nil {
				job.state = "valid"
				job.err = ""
			} else {
				job.state = "invalid"
				job.err = err.Error()
			}
			finished <- job

		} else {
			log.Println("received all jobs")
			finished <- nil
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

	stm, err := db.Prepare(
		"UPDATE nameservers SET name=?, state=?, error=?, version=?, checked_at=NOW()," +
			"state_changed_at = (CASE WHEN ? != state THEN NOW() ELSE state_changed_at END )" +
			"WHERE id=?")
	defer stm.Close()

	doneCount := 0
	for doneCount < workersLimit {
		res := <-finished
		// log.Println("finished job", res.id)
		if res == nil {
			doneCount++
			log.Println("worker terminated")
		} else {
			log.Printf("job id=%v state=%s name=%s err=%s", res.id, res.state, res.name, res.err)
			stm.Exec(res.name, res.state, res.err, res.version, res.state, res.id)
		}
	}
	done <- true
}

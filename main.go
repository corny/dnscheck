package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
	//"time"
)

const workersLimit = 10
const connection = "root:@tcp(localhost:3306)/nameservers_development"

var jobs = make(chan *job, 100)
var results = make(chan *result, 100)
var done = make(chan bool)

func main() {
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

	// Open SQL connection
	db, err := sql.Open("mysql", connection)
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	defer db.Close()

	for {
		// Prepare statement for reading data
		rows, err := db.Query("SELECT id, ip FROM nameservers WHERE id > ? LIMIT 1000", currentId)
		if err != nil {
			panic(err.Error()) // proper error handling instead of panic in your app
		}

		found := 0
		for rows.Next() {
			j := new(job)

			// get RawBytes from data
			err = rows.Scan(&j.id, &j.address)
			if err != nil {
				panic(err.Error()) // proper error handling instead of panic in your app
			}
			jobs <- j
			log.Println(j.id, j.address)
			currentId = j.id
			found += 1
		}
		rows.Close()
		if found == 0 {
			close(jobs)
			return
		}
	}
}

func worker() {
	for {
		job := <-jobs
		if job != nil {
			log.Println("received job", job.id)
			results <- check(job)
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
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	defer db.Close()

	stm, err := db.Prepare("UPDATE nameservers SET name=?, state=?, error=? WHERE id=?")
	defer stm.Close()

	doneCount := 0
	for doneCount < workersLimit {
		res := <-results
		log.Println("finished job", res.id)
		if res == nil {
			doneCount++
			log.Println("worker terminated")
		} else {
			log.Printf("job id=%i state=%s name=%s err=%s", res.id, res.state, res.name, res.err)
			stm.Exec(res.name, res.state, res.err, res.id)
		}
	}
	done <- true
}

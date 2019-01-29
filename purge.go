package main

import (
	"errors"
	"fmt"
	"log"
)

func runPurge() error {
	if *maxCheckAge < 1 {
		return errors.New("max-age can not be zero")
	}

	failed := false

	// purge checks older than N days
	res, err := dbConn.Exec("DELETE FROM nameserver_checks WHERE created_at < DATE_SUB(NOW(), INTERVAL ? DAY)", maxCheckAge)
	if err != nil {
		log.Println(err)
		failed = true
	} else {
		affected, _ := res.RowsAffected()
		fmt.Printf("Purged %d checks\n", affected)
	}

	// purge invalid nameservers checked within the last 7 days that has been invalid for N days
	res, err = dbConn.Exec("DELETE FROM nameservers WHERE state='invalid' AND checked_at > DATE_SUB(NOW(), INTERVAL 7 DAY) AND state_changed_at < DATE_SUB(NOW(), INTERVAL ? DAY)", maxCheckAge)
	if err != nil {
		log.Println(err)
		failed = true
	} else {
		affected, _ := res.RowsAffected()
		fmt.Printf("Purged %d nameservers\n", affected)
	}

	if failed {
		return errors.New("purging failed")
	}
	return nil
}

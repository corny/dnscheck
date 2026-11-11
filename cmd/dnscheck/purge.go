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
	affected, err := db.PurgeChecks(*maxCheckAge)
	if err != nil {
		log.Println(err)
		failed = true
	} else {
		fmt.Printf("Purged %d checks\n", affected)
	}

	// purge nameservers that have been invalid for N days
	affected, err = db.PurgeNameservers(*maxCheckAge)
	if err != nil {
		log.Println(err)
		failed = true
	} else {
		fmt.Printf("Purged %d nameservers\n", affected)
	}

	if failed {
		return errors.New("purging failed")
	}
	return nil
}

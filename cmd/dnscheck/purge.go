package main

import (
	"errors"
	"fmt"
	"log"
)

// runPurge deletes check results and invalid nameservers older than
// maxAgeDays.
func runPurge(maxAgeDays uint) error {
	if maxAgeDays < 1 {
		return errors.New("max-age can not be zero")
	}

	checks, err := db.PurgeChecks(maxAgeDays)
	if err != nil {
		return fmt.Errorf("purging checks: %w", err)
	}

	nameservers, err := db.PurgeNameservers(maxAgeDays)
	if err != nil {
		return fmt.Errorf("purging nameservers: %w", err)
	}

	log.Printf("purged %d checks and %d nameservers", checks, nameservers)
	return nil
}

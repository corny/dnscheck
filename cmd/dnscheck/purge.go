package main

import (
	"errors"
	"fmt"
	"log/slog"
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

	slog.Info("purged", "checks", checks, "nameservers", nameservers)
	return nil
}

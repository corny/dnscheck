package main

import (
	"context"
	"log"
	"time"

	"github.com/corny/dnscheck/internal/geoip"
)

// waitForGeoIP downloads the GeoLite2 databases and retries every minute
// until it succeeds. If a download fails but older databases exist, they
// are used.
func waitForGeoIP(ctx context.Context, u *geoip.Updater) error {
	for {
		err := u.Update(ctx)
		if err == nil {
			return nil
		}
		if u.Available() {
			log.Println("GeoLite2 update failed, using existing databases:", err)
			return nil
		}

		log.Println("GeoLite2 download failed, retrying in a minute:", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Minute):
		}
	}
}

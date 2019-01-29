package check

import (
	"log"
	"net"
)

func (checker *Checker) location(address string) (isocode string, city string) {
	record, err := checker.geoip.City(net.ParseIP(address))

	if err != nil {
		log.Printf("cannot resolve IP address to location %s: %s", address, err)
		return
	}

	return record.Country.IsoCode, record.City.Names["en"]
}

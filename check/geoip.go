package check

import (
	"log"
	"net"

	"github.com/oschwald/geoip2-golang"
)

// GeoDbPath points to a GeoIP mmdb file
var GeoDbPath string

// GeoLocate performs a geo location for the given address.
// You may need to setup GeoDbPath first.
func GeoLocate(address string) (isocode string, city string) {
	db, err := geoip2.Open(GeoDbPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	record, err := db.City(net.ParseIP(address))
	if err != nil {
		log.Fatal(err)
		return "", ""
	}

	return record.Country.IsoCode, record.City.Names["en"]
}

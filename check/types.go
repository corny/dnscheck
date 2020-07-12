package check

import (
	"net"

	"github.com/oschwald/geoip2-golang"
)

// Job represents a nameserver check
type Job struct {
	ID      int
	Address net.IP
	Name    string
	Version string
	Status  bool
	Error   string
	City    *geoip2.City
	ASN     *geoip2.ASN
	Dnssec  *bool
}

type resultMap map[string]ipSet

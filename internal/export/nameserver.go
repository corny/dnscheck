package export

import (
	"strconv"
	"time"

	"github.com/corny/dnscheck/internal/store"
)

// fieldString returns the string representation of the given attribute.
func fieldString(ns *store.Nameserver, attr string) string {
	nullStr := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}
	var res string
	switch attr {
	case "ip_address":
		res = ns.Address
	case "name":
		res = nullStr(ns.Name)
	case "as_number":
		if ns.ASNumber != nil {
			res = strconv.Itoa(int(*ns.ASNumber))
		}
	case "as_org":
		res = nullStr(ns.ASOrg)
	case "country_code":
		res = nullStr(ns.Country)
	case "city":
		res = nullStr(ns.City)
	case "version":
		res = nullStr(ns.Version)
	case "error":
		res = nullStr(ns.Error)
	case "dnssec":
		if ns.DNSSEC == nil {
			res = ""
		} else if *ns.DNSSEC {
			res = "true"
		} else {
			res = "false"
		}
	case "reliability":
		res = strconv.FormatFloat(ns.Reliability, 'f', 2, 64)
	case "checked_at":
		if ns.CheckedAt == nil {
			res = ""
		} else {
			res = ns.CheckedAt.Format(time.RFC3339)
		}
	case "created_at":
		res = ns.CreatedAt.Format(time.RFC3339)
	default:
		panic("invalid attribute: " + attr)
	}
	return res
}

// nameserverFields are the exported fields in CSV column order.
var nameserverFields = []string{
	"ip_address",
	"name",
	"as_number",
	"as_org",
	"country_code",
	"city",
	"version",
	"error",
	"dnssec",
	"reliability",
	"checked_at",
	"created_at",
}

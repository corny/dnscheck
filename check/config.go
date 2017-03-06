package check

import "github.com/miekg/dns"

// a place for global config variables

// DNSClient speaks DNS.
var DNSClient = &dns.Client{}

// MaxAttempts defines how often after an error a check
// shall be retried.
var MaxAttempts = 3

// Reference defines a DNS server to be queried for reference.
// You must trust this server.
var Reference = "8.8.8.8"

// Expectations maps checked domains to their results from the
// reference server
var Expectations ResultMap

// ResultMap maps different results for a domain check.
type ResultMap map[string]StringSet

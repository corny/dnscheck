package main

import (
	"fmt"
	"github.com/miekg/dns"
	"net"
)

var dnsClient = &dns.Client{}

// Query the given nameserver for all domains
func resolveDomains(nameserver string) (results resultMap, err error) {
	results = make(resultMap)

	for _, domain := range domains {
		result, err := resolve(nameserver, domain)
		if err != nil {
			return nil, err
		}
		results[domain] = result
	}

	return results, nil
}

// Query the given nameserver for a single domain
func resolve(nameserver string, domain string) (records stringSet, err error) {
	m := &dns.Msg{}
	m.RecursionDesired = true
	m.SetQuestion(dns.Fqdn(domain), dns.TypeA)

	hostPort := net.JoinHostPort(nameserver, "53")
	attempt := 1
	result := &dns.Msg{}

	// execute the query
	for {
		result, _, err = dnsClient.Exchange(m, hostPort)
		if err == nil {
			// success
			break
		} else {
			err = simplifyError(err)
			if err.Error() != "i/o timeout" || attempt == maxAttempts {
				// network problem or timeout
				return
			} else {
				err = nil
				// retry
				attempt++
			}
		}
	}

	// NXDomain rcode?
	if result.Rcode == dns.RcodeNameError {
		return
	}

	// Other erroneous rcode?
	if result.Rcode != dns.RcodeSuccess {
		err = fmt.Errorf("%v for %s", dns.RcodeToString[result.Rcode], domain)
		return
	}

	records = make(stringSet)

	// Add addresses to set
	for _, a := range result.Answer {
		if record, ok := a.(*dns.A); ok {
			records.add(record.A.String())
		}
	}

	return
}

func ptrName(address string) string {
	reverse, err := dns.ReverseAddr(address)
	if err != nil {
		return ""
	}

	m := &dns.Msg{}
	m.RecursionDesired = true
	m.SetQuestion(reverse, dns.TypePTR)

	// execute the query
	result, _, err := dnsClient.Exchange(m, net.JoinHostPort(referenceServer, "53"))
	if result == nil || result.Rcode != dns.RcodeSuccess {
		return ""
	}

	// Add addresses to set
	for _, a := range result.Answer {
		if record, ok := a.(*dns.PTR); ok {
			return record.Ptr
		}
	}
	return ""
}

// Query a nameserver for the bind version
func version(address string) string {
	m := &dns.Msg{}
	m.RecursionDesired = true
	m.Question = make([]dns.Question, 1)
	m.Question[0] = dns.Question{"version.bind.", dns.TypeTXT, dns.ClassCHAOS}

	// Execute the query
	r, _, _ := dnsClient.Exchange(m, net.JoinHostPort(address, "53"))

	// Valid response?
	if r != nil && r.Rcode == dns.RcodeSuccess {
		for _, a := range r.Answer {
			if record, ok := a.(*dns.TXT); ok {
				return record.Txt[0]
			}
		}
	}
	return ""
}

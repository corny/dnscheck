package main

import (
	"fmt"
	"github.com/miekg/dns"
	"net"
)

// maximum number of attempts for a query
const maxAttempts = 3

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

	attempt := 1
	result := &dns.Msg{}

	// execute the query
	for {
		result, _, err = dnsClient.Exchange(m, net.JoinHostPort(nameserver, "53"))
		if err == nil {
			// success
			break
		} else {
			err = simplifyError(err)
			if err.Error() != "i/o timeout" || attempt == maxAttempts {
				// network problem or timeout
				return nil, simplifyError(err)
			} else {
				// retry
				attempt++
			}
		}
	}

	// NXDomain rcode?
	if result.Rcode == dns.RcodeNameError {
		return nil, nil
	}

	// Other erroneous rcode?
	if result.Rcode != dns.RcodeSuccess {
		return nil, fmt.Errorf("%v for %s", dns.RcodeToString[result.Rcode], domain)
	}

	records = make(stringSet)

	// Add addresses to set
	for _, a := range result.Answer {
		if record, ok := a.(*dns.A); ok {
			records.add(record.A.String())
		}
	}

	return records, nil
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
	result, _, err := dnsClient.Exchange(m, net.JoinHostPort(referenceNameserver, "53"))
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

	// execute the query
	r, _, _ := dnsClient.Exchange(m, net.JoinHostPort(address, "53"))
	if r == nil || r.Rcode != dns.RcodeSuccess {
		return ""
	}

	// Add addresses to set
	for _, a := range r.Answer {
		if record, ok := a.(*dns.TXT); ok {
			return record.Txt[0]
		}
	}
	return ""
}

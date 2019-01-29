package check

import (
	"fmt"
	"net"
	"time"

	"github.com/miekg/dns"
)

func (checker *Checker) exchange(m *dns.Msg, address string) (r *dns.Msg, rtt time.Duration, err error) {
	incrementMetric(&Metrics.Queries)
	return checker.DNSClient.Exchange(m, address)
}

// Query the given nameserver for all domains
func (checker *Checker) resolveDomains(nameserver string) (results resultMap, authenticated bool, err error) {
	results = make(resultMap)

	for _, domain := range checker.domains {
		result, authenticatedValue, err := checker.resolve(nameserver, domain)
		if err != nil {
			return nil, false, err
		}
		results[domain] = result
		authenticated = authenticated || authenticatedValue
	}

	return results, authenticated, nil
}

// Query the given nameserver for a single domain
func (checker *Checker) resolve(nameserver string, domain string) (records stringSet, authenticated bool, err error) {
	m := &dns.Msg{}
	m.RecursionDesired = true
	m.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	m.AuthenticatedData = true

	hostPort := net.JoinHostPort(nameserver, "53")
	attempt := uint(1)
	result := &dns.Msg{}

	// execute the query
	for {
		result, _, err = checker.exchange(m, hostPort)
		if err == nil {
			// success
			break
		} else {
			err = simplifyError(err)
			if err.Error() != "i/o timeout" || attempt == checker.MaxAttempts {
				// network problem or timeout
				return
			}
			err = nil
			// retry
			attempt++
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

	authenticated = result.AuthenticatedData
	records = make(stringSet)

	// Add addresses to set
	for _, a := range result.Answer {
		if record, ok := a.(*dns.A); ok {
			records.add(record.A.String())
		}
	}

	return
}

func (checker *Checker) ptrName(address string) string {
	reverse, err := dns.ReverseAddr(address)
	if err != nil {
		return ""
	}

	m := &dns.Msg{}
	m.RecursionDesired = true
	m.SetQuestion(reverse, dns.TypePTR)

	// execute the query
	result, _, err := checker.exchange(m, net.JoinHostPort(checker.ReferenceServer, "53"))
	if err != nil || result == nil || result.Rcode != dns.RcodeSuccess {
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
func (checker *Checker) version(address string) string {
	m := &dns.Msg{}
	m.RecursionDesired = true
	m.Question = make([]dns.Question, 1)
	m.Question[0] = dns.Question{
		Name:   "version.bind.",
		Qtype:  dns.TypeTXT,
		Qclass: dns.ClassCHAOS,
	}

	// Execute the query
	r, _, _ := checker.exchange(m, net.JoinHostPort(address, "53"))

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

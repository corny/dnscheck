package check

import (
	"fmt"
	"net"

	"github.com/miekg/dns"
)

// ResolveDomains queries the given nameserver for all domains.
func ResolveDomains(nameserver string) (results ResultMap, authenticated bool, err error) {
	results = make(ResultMap)

	for _, domain := range domains {
		result, authenticatedValue, err := resolve(nameserver, domain)
		if err != nil {
			return nil, false, err
		}
		results[domain] = result
		authenticated = authenticated || authenticatedValue
	}

	return results, authenticated, nil
}

// Query the given nameserver for a single domain
func resolve(nameserver string, domain string) (records StringSet, authenticated bool, err error) {
	m := &dns.Msg{}
	m.RecursionDesired = true
	m.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	m.AuthenticatedData = true

	hostPort := net.JoinHostPort(nameserver, "53")
	attempt := 1
	result := &dns.Msg{}

	// execute the query
	for {
		result, _, err = DNSClient.Exchange(m, hostPort)
		if err == nil {
			// success
			break
		} else {
			err = simplifyError(err)
			if err.Error() != "i/o timeout" || attempt == MaxAttempts {
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
	records = make(StringSet)

	// Add addresses to set
	for _, a := range result.Answer {
		if record, ok := a.(*dns.A); ok {
			records.Add(record.A.String())
		}
	}

	return
}

// PtrName tries to retrieve the PTR record for the given address.
func PtrName(address string) string {
	reverse, err := dns.ReverseAddr(address)
	if err != nil {
		return ""
	}

	m := &dns.Msg{}
	m.RecursionDesired = true
	m.SetQuestion(reverse, dns.TypePTR)

	// execute the query
	result, _, err := DNSClient.Exchange(m, net.JoinHostPort(Reference, "53"))
	if result == nil || result.Rcode != dns.RcodeSuccess || err != nil {
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

// Version queries a nameserver for the bind version.
func Version(address string) string {
	m := &dns.Msg{}
	m.RecursionDesired = true
	m.Question = make([]dns.Question, 1)
	m.Question[0] = dns.Question{
		Name:   "version.bind.",
		Qtype:  dns.TypeTXT,
		Qclass: dns.ClassCHAOS,
	}

	// Execute the query
	r, _, _ := DNSClient.Exchange(m, net.JoinHostPort(address, "53"))

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

package check

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/miekg/dns"
)

var addressTypes = []uint16{dns.TypeA, dns.TypeAAAA}

func (checker *Checker) exchange(m *dns.Msg, serverAddress string) (r *dns.Msg, rtt time.Duration, err error) {
	incrementMetric(&Metrics.Queries)
	return checker.DNSClient.Exchange(m, serverAddress)
}

// exchange message with the reference nameserver
func (checker *Checker) exchangeReference(m *dns.Msg) (r *dns.Msg, rtt time.Duration, err error) {
	return checker.exchange(m, net.JoinHostPort(checker.ReferenceServer, "53"))
}

// resolves the fqdn and returns all IP addresses
func (checker *Checker) resolveAddresses(fqdn string) (result []net.IP, err error) {

	mtx := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(addressTypes))

	for _, qType := range addressTypes {
		go func() {
			resResult, _, resErr := checker.resolve(checker.ReferenceServer, fqdn, qType)
			mtx.Lock()

			if resErr != nil {
				err = resErr
			} else {
				result = append(result, resResult...)
			}

			mtx.Unlock()
			wg.Done()
		}()
	}
	wg.Wait()

	return
}

// Query the given nameserver for all domains (A records)
func (checker *Checker) resolveDomains(nameserver string) (results resultMap, authenticated bool, err error) {
	results = make(resultMap)

	for _, domain := range checker.domains {
		result, authenticatedValue, err := checker.resolve(nameserver, domain, dns.TypeA)
		if err != nil {
			return nil, false, err
		}
		results[domain] = ipSet{result}
		authenticated = authenticated || authenticatedValue
	}

	return results, authenticated, nil
}

// Query the given nameserver for a single domain
func (checker *Checker) resolve(nameserver string, domain string, qType uint16) (records []net.IP, authenticated bool, err error) {
	m := &dns.Msg{}
	m.RecursionDesired = true
	m.SetQuestion(dns.Fqdn(domain), qType)
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

	// Add addresses to set
	for _, a := range result.Answer {
		switch record := a.(type) {
		case *dns.A:
			records = append(records, record.A)
		case *dns.AAAA:
			records = append(records, record.AAAA)
		}
	}

	return
}

// ptrName does a reverse lookup
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

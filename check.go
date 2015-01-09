package main

import (
	"errors"
	"fmt"
	"github.com/miekg/dns"
	//"log"
	"net"
)

var domains = []string{
	"example.com",
	"wikileaks.org",
	"non-existent.example.com",
}

func resolve(nameserver string, domain string) (records stringSet, err error) {
	fmt.Printf("%i\n", len(domains))

	c := &dns.Client{}
	m := &dns.Msg{}
	m.RecursionDesired = true
	m.SetQuestion(dns.Fqdn(domain), dns.TypeA)

	r, _, err := c.Exchange(m, net.JoinHostPort(nameserver, "53"))
	if r == nil {
		return nil, err
	}

	// NXDomain?
	if r.Rcode == dns.RcodeNameError {
		return nil, nil
	}

	records = make(stringSet)

	// Other non-success rcode?
	if r.Rcode != dns.RcodeSuccess {
		return nil, errors.New(fmt.Sprintf("RCODE %i for %s", r.Rcode, domain))
	}

	for _, a := range r.Answer {
		if record, ok := a.(*dns.A); ok {
			records.Add(record.A.String())
		}
	}

	return records, nil
}

func check(job *job) *result {
	result := &result{id: job.id}

	result.err = ""
	result.state = "valid"
	return result
}

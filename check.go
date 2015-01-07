package main

import (
	"fmt"
	"github.com/miekg/dns"
	"log"
	"net"
)

func check(server string) (state string, result string) {
	domain := "wikileaks.org"
	c := &dns.Client{}
	m := &dns.Msg{}
	m.RecursionDesired = true
	m.SetQuestion(dns.Fqdn(domain), dns.TypeA)

	r, _, err := c.Exchange(m, net.JoinHostPort(server, "53"))
	if r == nil {
		log.Fatalf("*** error: %s\n", err.Error())
	}

	if r.Rcode != dns.RcodeSuccess {
		log.Fatalf(" *** invalid answer from %s for %s\n", server, domain)
	}

	// Stuff must be in the answer section
	for _, a := range r.Answer {
		result = fmt.Sprintf("%v\n", a)
	}

	state = "valid"
	return
}

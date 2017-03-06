package dnscheck

// A Nameserver represents a service running a DNS daemon.
type Nameserver struct {
	ID      int
	Address string
	Name    string
	Version string
	State   string
	Error   string
	Country string
	City    string
	DNSSEC  *bool
}

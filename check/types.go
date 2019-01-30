package check

// Job represents a nameserver check
type Job struct {
	ID      int
	Address string
	Name    string
	Version string
	State   string
	Err     string
	Country string
	City    string
	Dnssec  *bool
}

type stringSet map[string]struct{}

type resultMap map[string]stringSet

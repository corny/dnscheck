package check

import (
	"net"
	"strings"
)

type ipSet struct {
	list []net.IP
}

// creates a comma separated sorted list
func (set *ipSet) String() string {
	var b strings.Builder

	for i := range set.list {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(set.list[i].String())
	}

	return b.String()
}

func (set *ipSet) add(ip net.IP) {
	set.list = append(set.list, ip)
}

func (set *ipSet) contains(ip net.IP) bool {
	for i := range set.list {
		if set.list[i].Equal(ip) {
			return true
		}
	}
	return false
}

func (set *ipSet) containsSet(other *ipSet) bool {
	for i := range other.list {
		if !set.contains(other.list[i]) {
			return false
		}
	}
	return true
}

func (set *ipSet) equals(other *ipSet) bool {
	return set.containsSet(other) && other.containsSet(set)
}

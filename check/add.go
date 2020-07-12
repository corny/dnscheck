package check

import "net"

func (checker *Checker) Resolve(address string) (addresses []net.IP, err error) {

	if addr := net.ParseIP(address); addr != nil {
		addresses = []net.IP{addr}
	} else {
		addresses, err = checker.resolveAddresses(address)
	}

	return
}

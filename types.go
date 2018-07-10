package main

type job struct {
	id      int
	address string
	name    string
	version string
	state   string
	err     string
	country string
	city    string
	dnssec  *bool
}

type stringSet map[string]struct{}

type resultMap map[string]stringSet

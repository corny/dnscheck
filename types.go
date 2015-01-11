package main

type job struct {
	id      int
	address string
	name    string
	version string
	state   string
	err     string
}

type stringSet map[string]struct{}

type resultMap map[string]stringSet

type config map[string]map[string]string

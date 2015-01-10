package main

type job struct {
	id      int
	address string
	name    string
	state   string
	err     string
}

type stringSet map[string]struct{}

type resultMap map[string]stringSet

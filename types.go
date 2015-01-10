package main

type job struct {
	id      int
	address string
}

type stringSet map[string]struct{}

type resultMap map[string]stringSet

type result struct {
	id    int
	name  string
	state string
	err   string
}

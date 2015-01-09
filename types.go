package main

type job struct {
	id      int
	address string
}

type stringSet map[string]struct{}

func (set stringSet) Add(str string) {
	set[str] = struct{}{}
}

func (set stringSet) equals() bool {
	return false
}

type result struct {
	id    int
	name  string
	state string
	err   string
}

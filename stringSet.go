package main

import (
	"sort"
	"strings"
)

// creates a comma separated sorted list
func (set stringSet) String() string {
	i := 0
	result := make([]string, len(set))

	for key := range set {
		result[i] = key
		i++
	}

	sort.Strings(result)
	return strings.Join(result, ", ")
}

func (set stringSet) add(str string) {
	set[str] = struct{}{}
}

func (set stringSet) equals(other stringSet) bool {
	if len(set) != len(other) {
		return false
	}
	if len(set) == 0 {
		return true
	}
	for key := range set {
		_, ok := other[key]
		if !ok {
			return false
		}
	}
	return true
}

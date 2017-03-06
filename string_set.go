package main

import (
	"sort"
	"strings"
)

// creates a comma seperated sorted list
func (set stringSet) String() string {
	i := 0
	result := make([]string, len(set))

	for key, _ := range set {
		result[i] = key
		i++
	}

	sort.Strings(result)
	return strings.Join(result, ", ")
}

func (set stringSet) add(str string) {
	set[str] = struct{}{}
}

func (this stringSet) equals(that stringSet) bool {
	if len(this) != len(that) {
		return false
	}
	if len(this) == 0 {
		return true
	}
	for key, _ := range this {
		_, ok := that[key]
		if !ok {
			return false
		}
	}
	return true
}

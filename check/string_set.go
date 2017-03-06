package check

import (
	"sort"
	"strings"
)

// StringSet is a data type for unique strings.
type StringSet map[string]struct{}

// creates a comma seperated sorted list
func (set StringSet) String() string {
	i := 0
	result := make([]string, len(set))

	for key := range set {
		result[i] = key
		i++
	}

	sort.Strings(result)
	return strings.Join(result, ", ")
}

// Add adds a string to the set.
func (set StringSet) Add(str string) {
	set[str] = struct{}{}
}

// Equals compares two string sets.
func (set StringSet) Equals(other StringSet) bool {
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

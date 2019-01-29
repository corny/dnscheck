package check

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompare(t *testing.T) {
	assert := assert.New(t)

	set1 := make(stringSet)
	set1.add("bar")
	set1.add("foo")

	set2 := make(stringSet)
	set2.add("foo")
	set2.add("bar")

	set3 := make(stringSet)
	set3.add("foo")
	set3.add("baz")

	assert.Equal(set1, set2)
	assert.Equal(set2, set1)
	assert.NotEqual(set1, set3)
}

func TestString(t *testing.T) {
	assert := assert.New(t)

	set := make(stringSet)
	set.add("bar")
	set.add("xx")
	set.add("foo")

	assert.Equal("bar, foo, xx", set.String())
}

package check

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompare(t *testing.T) {
	assert := assert.New(t)

	set1 := make(StringSet)
	set1.Add("bar")
	set1.Add("foo")

	set2 := make(StringSet)
	set2.Add("foo")
	set2.Add("bar")

	set3 := make(StringSet)
	set3.Add("foo")
	set3.Add("baz")

	assert.Equal(set1, set2)
	assert.Equal(set2, set1)
	assert.NotEqual(set1, set3)
}

func TestString(t *testing.T) {
	assert := assert.New(t)

	set := make(StringSet)
	set.Add("bar")
	set.Add("xx")
	set.Add("foo")

	assert.Equal("bar, foo, xx", set.String())
}

package check

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompare(t *testing.T) {
	assert := assert.New(t)

	set1 := ipSet{}
	set1.add(net.ParseIP("127.0.0.1"))
	set1.add(net.ParseIP("1.2.3.4"))

	set2 := ipSet{}
	set2.add(net.ParseIP("127.0.0.1"))
	set2.add(net.ParseIP("1.2.3.4"))

	set3 := ipSet{}
	set3.add(net.ParseIP("10.8.0.1"))
	set3.add(net.ParseIP("1.2.3.4"))

	assert.Equal(set1, set2)
	assert.Equal(set2, set1)
	assert.NotEqual(set1, set3)
}

func TestString(t *testing.T) {
	assert := assert.New(t)

	set := ipSet{}
	set.add(net.ParseIP("127.0.0.1"))
	set.add(net.ParseIP("1.2.3.4"))

	assert.Equal("127.0.0.1, 1.2.3.4", set.String())
}

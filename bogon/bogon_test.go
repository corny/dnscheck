package bogon

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBogon(t *testing.T) {
	assert := assert.New(t)

	assert.True(IsBogon(net.ParseIP("127.0.0.1")))
	assert.False(IsBogon(net.ParseIP("213.1.2.3")))

	assert.True(IsBogon(net.ParseIP("fe80::123")))
	assert.False(IsBogon(net.ParseIP("2a00:1450::1")))
}

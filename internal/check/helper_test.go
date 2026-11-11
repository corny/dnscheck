package check

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimplifyTimeout(t *testing.T) {
	assert := assert.New(t)
	result := simplifyError(errors.New("read udp 91.194.211.134:53: i/o timeout"))

	assert.EqualError(result, "i/o timeout")
}

func TestSimplifyNetworkUnreachable(t *testing.T) {
	assert := assert.New(t)
	result := simplifyError(errors.New("dial udp [2002:d596:2a92:1:71:53::]:53: network is unreachable"))

	assert.EqualError(result, "network is unreachable")
}

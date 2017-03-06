package check

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExistent(t *testing.T) {
	assert := assert.New(t)
	result, _, err := resolve(Reference, "example.com")

	assert.Nil(err)
	assert.Len(result, 1)
}

func TestNotExistent(t *testing.T) {
	assert := assert.New(t)
	result, authenticated, err := resolve(Reference, "xxx.example.com")

	assert.Nil(err)
	assert.False(authenticated)
	assert.Len(result, 0)
}

func TestAuthenticated(t *testing.T) {
	assert := assert.New(t)
	result, authenticated, err := resolve(Reference, "www.dnssec-tools.org")

	assert.Nil(err)
	assert.True(authenticated)
	assert.Len(result, 1)
}

func TestUnreachable(t *testing.T) {
	assert := assert.New(t)
	_, _, err := resolve("127.1.2.3", "example.com")

	assert.EqualError(err, "connection refused")
}

func TestPtrName(t *testing.T) {
	assert := assert.New(t)
	result := PtrName("8.8.8.8")

	assert.Equal("google-public-dns-a.google.com.", result)
}

func TestVersion(t *testing.T) {
	assert := assert.New(t)
	result := Version("82.96.65.2")

	assert.Equal("Make my day", result)
}

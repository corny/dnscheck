package check

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	referenceServer = "8.8.8.8"
	testChecker     = &Checker{ReferenceServer: referenceServer}
)

func resolve(server, query string) (records stringSet, authenticated bool, err error) {
	return testChecker.resolve(server, query)
}

func TestExistent(t *testing.T) {
	assert := assert.New(t)
	result, _, err := resolve(referenceServer, "example.com")

	assert.Nil(err)
	assert.Len(result, 1)
}

func TestNotExistent(t *testing.T) {
	assert := assert.New(t)
	result, authenticated, err := resolve(referenceServer, "xxx.example.com")

	assert.Nil(err)
	assert.False(authenticated)
	assert.Len(result, 0)
}

func TestAuthenticated(t *testing.T) {
	assert := assert.New(t)
	result, authenticated, err := resolve(referenceServer, "verisignlabs.com")

	assert.Nil(err)
	assert.True(authenticated)
	assert.GreaterOrEqual(len(result), 1)
}

func TestUnreachable(t *testing.T) {
	assert := assert.New(t)
	_, _, err := resolve("127.1.2.3", "example.com")

	assert.EqualError(err, "connection refused")
}

func TestPtrName(t *testing.T) {
	assert := assert.New(t)
	result := testChecker.ptrName("8.8.8.8")

	assert.Equal("dns.google.", result)
}

func TestVersion(t *testing.T) {
	assert := assert.New(t)
	result := testChecker.version("82.96.65.2")

	assert.Equal("Make my day", result)
}

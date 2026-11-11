package check

import (
	"net"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

var (
	referenceServer = "8.8.8.8"
	testChecker     = &Checker{ReferenceServer: referenceServer}
)

// skipWithoutNetwork skips tests that query public DNS servers.
func skipWithoutNetwork(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("queries public DNS servers")
	}
}

func resolve(server, query string) (records []net.IP, authenticated bool, err error) {
	return testChecker.resolve(server, query, dns.TypeA)
}

func TestExistent(t *testing.T) {
	skipWithoutNetwork(t)
	assert := assert.New(t)
	result, _, err := resolve(referenceServer, "example.com")

	assert.NoError(err)
	assert.NotEmpty(result)
}

func TestNotExistent(t *testing.T) {
	skipWithoutNetwork(t)
	assert := assert.New(t)
	result, _, err := resolve(referenceServer, "xxx.example.com")

	assert.NoError(err)
	assert.Empty(result)
}

func TestAuthenticated(t *testing.T) {
	skipWithoutNetwork(t)
	assert := assert.New(t)
	result, authenticated, err := resolve(referenceServer, "verisignlabs.com")

	assert.NoError(err)
	assert.True(authenticated)
	assert.NotEmpty(result)
}

func TestUnreachable(t *testing.T) {
	skipWithoutNetwork(t)
	_, _, err := resolve("127.1.2.3", "example.com")

	// "connection refused" on Linux, "i/o timeout" on macOS
	assert.Error(t, err)
}

func TestPtrName(t *testing.T) {
	skipWithoutNetwork(t)
	assert := assert.New(t)
	result := testChecker.ptrName("8.8.8.8")

	assert.Equal("dns.google.", result)
}

func TestVersion(t *testing.T) {
	skipWithoutNetwork(t)
	assert := assert.New(t)
	result := testChecker.version("82.96.65.2")

	assert.Equal("Make my day", result)
}

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	result, authenticated, err := resolve(referenceServer, "www.dnssec-tools.org")

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
	result := ptrName("8.8.8.8")

	assert.Equal("google-public-dns-a.google.com.", result)
}

func TestVersion(t *testing.T) {
	assert := assert.New(t)
	result := version("82.96.65.2")

	assert.Equal("Make my day", result)
}

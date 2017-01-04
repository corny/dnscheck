package main

import "testing"

import "github.com/stretchr/testify/assert"

func TestCheckResult(t *testing.T) {
	assert := assert.New(t)
	domains = []string{"example.com"}

	correctAddr := make(stringSet)
	correctAddr.add("1.2.3.4")
	correctMap := make(resultMap)
	correctMap["example.com"] = correctAddr

	incorrectAddr := make(stringSet)
	incorrectAddr.add("23.0.0.1")
	incorrectAddr.add("23.0.0.2")
	incorrectMap := make(resultMap)
	incorrectMap["example.com"] = incorrectAddr

	// compare correct with correct
	err := checkResult(correctMap, correctMap)
	assert.NoError(err)

	// compare correct with invalid
	err = checkResult(correctMap, incorrectMap)
	assert.Error(err, "Unexpected result for example.com: 23.0.0.1, 23.0.0.2")
}

func TestCheckResultEmpty(t *testing.T) {
	assert := assert.New(t)
	domains = []string{"example.com"}

	correctAddr := make(stringSet)
	correctAddr.add("1.2.3.4")
	correctMap := make(resultMap)
	correctMap["example.com"] = correctAddr

	incorrectAddr := make(stringSet)
	incorrectMap := make(resultMap)
	incorrectMap["example.com"] = incorrectAddr

	// compare correct with invalid
	err := checkResult(correctMap, incorrectMap)
	assert.Error(err, "Unexpected result for example.com: NXDOMAIN")
}

func TestReadDomains(t *testing.T) {
	assert := assert.New(t)
	err := readDomains("domains.txt")

	assert.NoError(err)
	assert.Len(domains, 4)
}

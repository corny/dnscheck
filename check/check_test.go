package check

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckResult(t *testing.T) {
	assert := assert.New(t)
	domains := []string{"example.com"}

	correctAddr := ipSet{}
	correctAddr.add(net.ParseIP("1.2.3.4"))
	correctMap := make(resultMap)
	correctMap["example.com"] = correctAddr

	incorrectAddr := ipSet{}
	incorrectAddr.add(net.ParseIP("23.0.0.1"))
	incorrectAddr.add(net.ParseIP("23.0.0.2"))
	incorrectMap := make(resultMap)
	incorrectMap["example.com"] = incorrectAddr

	// compare correct with correct
	err := checkResult(domains, correctMap, correctMap)
	assert.NoError(err)

	// compare correct with invalid
	err = checkResult(domains, correctMap, incorrectMap)
	assert.Error(err, "Unexpected result for example.com: 23.0.0.1, 23.0.0.2")
}

func TestCheckResultEmpty(t *testing.T) {
	assert := assert.New(t)
	domains := []string{"example.com"}

	correctAddr := ipSet{}
	correctAddr.add(net.ParseIP("1.2.3.4"))
	correctMap := make(resultMap)
	correctMap["example.com"] = correctAddr

	incorrectAddr := ipSet{}
	incorrectMap := make(resultMap)
	incorrectMap["example.com"] = incorrectAddr

	// compare correct with invalid
	err := checkResult(domains, correctMap, incorrectMap)
	assert.Error(err, "Unexpected result for example.com: NXDOMAIN")
}

func TestReadDomains(t *testing.T) {
	assert := assert.New(t)
	checker := Checker{}
	err := checker.ReadDomains("../domains.txt")

	assert.NoError(err)
	assert.Len(checker.domains, 4)
}

package main

import "testing"

func TestCheckResult(t *testing.T) {
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
	if err != nil {
		t.Fatal(err)
	}

	// compare correct with invalid
	err = checkResult(correctMap, incorrectMap)
	if err.Error() != "Unexpected result for example.com: 23.0.0.1, 23.0.0.2" {
		t.Fatal(err)
	}
}

func TestCheckResultEmpty(t *testing.T) {
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
	if err.Error() != "Unexpected result for example.com: NXDOMAIN" {
		t.Fatal(err)
	}
}

func TestReadDomains(t *testing.T) {
	err := readDomains("domains.txt")
	if err != nil {
		t.Fatal(err)
	}

	if len(domains) != 4 {
		t.Fatal("unexpected domain list:", domains)
	}
}

package main

import "testing"

func TestCheckResult(t *testing.T) {
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

package main

import (
	"fmt"
)

// domains to check
var domains = []string{
	"example.com",
	"wikileaks.org",
	"non-existent.example.com",
}

var expectedResults resultMap

// Compare the result with the expectations
func checkResult(expectedMap resultMap, solvedMap resultMap) error {

	for domain, expected := range expectedMap {
		result := solvedMap[domain]
		if !expected.equals(result) {
			return fmt.Errorf("Unexpected result for %s: %v", domain, result)
		}
	}

	return nil
}

func check(job *job) error {
	solved, err := resolveDomains(job.address)

	if err != nil {
		return err
	}

	return checkResult(expectedResults, solved)
}

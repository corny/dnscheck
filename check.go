package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// domains to check
var domains = []string{}

// map of checked domains and their results from the reference server
var expectedResults resultMap

// reads the domains to check from the given file
func readDomains(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read lines
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// skip empty lines and lines with leading hash
		if len(line) > 0 && strings.Index(line, "#") == -1 {
			domains = append(domains, line)
		}
	}

	return scanner.Err()
}

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

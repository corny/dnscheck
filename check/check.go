package check

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/corny/dnscheck"
)

// domains to check
var domains = []string{}

// ReadDomains reads the domains to check from the given file.
func ReadDomains(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// create empty array
	domains = []string{}

	// Read lines
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// skip empty lines and lines with leading hash
		if len(line) > 0 && !strings.Contains(line, "#") {
			domains = append(domains, line)
		}
	}

	return scanner.Err()
}

// Compare the result with the expectations
func checkResult(expectedMap ResultMap, solvedMap ResultMap) error {
	for _, domain := range domains {
		expected := expectedMap[domain]
		result := solvedMap[domain]
		if !expected.Equals(result) {
			if len(result) == 0 {
				// empty result means NXDOMAIN
				return fmt.Errorf("Unexpected result for %s: NXDOMAIN", domain)
			}
			return fmt.Errorf("Unexpected result for %s: %v", domain, result)
		}
	}

	return nil
}

// Run performs a check job.
func Run(job *dnscheck.Nameserver) (bool, error) {
	solved, dnssec, err := ResolveDomains(job.Address)
	if err != nil {
		return dnssec, err
	}
	return dnssec, checkResult(Expectations, solved)
}

package check

import (
	"bufio"
	"os"
	"strings"
)

// ReadDomains reads the domains to be checked from the given file
func (checker *Checker) ReadDomains(path string) error {
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
			checker.AddDomain(line)
		}
	}

	return scanner.Err()
}

// AddDomain adds a domain if it does not exist yet
func (checker *Checker) AddDomain(domain string) bool {
	for i := range checker.domains {
		if checker.domains[i] == domain {
			return false
		}
	}

	checker.domains = append(checker.domains, domain)
	return true
}

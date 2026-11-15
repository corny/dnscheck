package check

import (
	"bufio"
	"os"
	"slices"
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

		// skip empty lines and comments
		if line != "" && !strings.Contains(line, "#") {
			checker.AddDomain(line)
		}
	}

	return scanner.Err()
}

// AddDomain adds a domain if it does not exist yet
func (checker *Checker) AddDomain(domain string) {
	if !slices.Contains(checker.domains, domain) {
		checker.domains = append(checker.domains, domain)
	}
}

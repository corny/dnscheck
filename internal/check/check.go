package check

import (
	"fmt"
)

func (checker *Checker) check(job *Job) (bool, error) {
	solved, dnssec, err := checker.resolveDomains(job.Address.String())

	if err != nil {
		return dnssec, err
	}

	return dnssec, checkResult(checker.domains, checker.expectedResults, solved)
}

// Compare the result with the expectations
func checkResult(domains []string, expectedMap resultMap, solvedMap resultMap) error {
	for _, domain := range domains {
		expected := expectedMap[domain]
		result := solvedMap[domain]
		if !expected.equals(&result) {
			if len(result.list) == 0 {
				// empty result means NXDOMAIN
				return fmt.Errorf("Unexpected result for %s: NXDOMAIN", domain)
			}
			return fmt.Errorf("Unexpected result for %s: %v", domain, result)
		}
	}

	return nil
}

package check

import (
	"errors"
	"strings"
)

func simplifyError(err error) error {
	str := err.Error()
	index := strings.LastIndex(str, ": ")

	if index != -1 {
		str = str[index+2:]
	}

	return errors.New(str)
}

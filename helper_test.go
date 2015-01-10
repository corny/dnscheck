package main

import (
	"errors"
	"testing"
)

func TestSimplifyTimeout(t *testing.T) {
	result := simplifyError(errors.New("read udp 91.194.211.134:53: i/o timeout"))

	if result.Error() != "i/o timeout" {
		t.Fatal("unexpected result:", result)
	}
}

func TestSimplifyNetworkUnreachable(t *testing.T) {
	result := simplifyError(errors.New("dial udp [2002:d596:2a92:1:71:53::]:53: network is unreachable"))

	if result.Error() != "network is unreachable" {
		t.Fatal("unexpected result:", result)
	}
}

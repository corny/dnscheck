package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListenAddressFromEnv(t *testing.T) {
	t.Setenv("LISTEN_ADDRESS", "127.0.0.1:8123")

	_, err := app.Parse([]string{"serve"})
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:8123", *listenAddress)
}

package main

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := newLogger(&buf, false)
	logger.Debug("hidden")
	logger.Info("shown", "count", 3)

	assert.NotContains(t, buf.String(), "hidden")
	assert.Regexp(t, `^time=\S+ level=INFO msg=shown count=3\n$`, buf.String())
}

func TestNewLoggerDebug(t *testing.T) {
	var buf bytes.Buffer
	logger := newLogger(&buf, true)
	logger.Debug("details", slog.Group("job", "id", 1))

	assert.Regexp(t, `^time=\S+ level=DEBUG msg=details job.id=1\n$`, buf.String())
}

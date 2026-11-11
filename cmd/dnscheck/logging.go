package main

import (
	"io"
	"log/slog"
	"os"
)

// newLogger creates a text logger. With debug, it logs debug messages too.
func newLogger(w io.Writer, debug bool) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	if debug {
		opts.Level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(w, opts))
}

// fatal logs an error and exits.
func fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

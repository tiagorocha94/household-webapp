package main

import (
	"fmt"
	"io"
	"log/slog"
)

// newLogger builds the application logger, which writes text records to w at
// the given level (debug, info, warn or error).
func newLogger(w io.Writer, level string) (*slog.Logger, error) {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("parse log level %q: %w", level, err)
	}
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: l})), nil
}

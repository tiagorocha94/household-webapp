package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger, err := newLogger(&buf, "warn")
	if err != nil {
		t.Fatalf("newLogger() error = %v", err)
	}
	logger.Info("hidden message")
	logger.Warn("shown message", "key", "value")
	out := buf.String()
	if strings.Contains(out, "hidden message") || !strings.Contains(out, "shown message") || !strings.Contains(out, "key=value") {
		t.Fatalf("log output = %q", out)
	}

	if _, err := newLogger(&buf, "loud"); err == nil {
		t.Fatal("newLogger(loud) succeeded, want error")
	}
}

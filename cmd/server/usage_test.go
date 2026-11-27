package main

import (
	"bytes"
	"flag"
	"strings"
	"testing"
)

func TestUsageListsTheServerFlagsAndTheDeviceCommands(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("household-app", flag.ContinueOnError)
	fs.String("config", "config.yaml", "path to the configuration file")
	fs.Bool("dry-run", false, "report what starting would change")

	var out bytes.Buffer
	printUsage(&out, fs)

	for _, want := range []string{
		"household-app [flags]",
		"household-app device add",
		"household-app device list",
		"household-app device revoke",
		"-config string",
		"-dry-run",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("usage does not mention %q:\n%s", want, out.String())
		}
	}
}

package main

import (
	"bytes"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/database"
	"github.com/tiagorocha94/household-webapp/internal/database/migrations"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

// deviceTestConfig makes a configuration file and a database with the users
// alice and bob, and returns the configuration path.
func deviceTestConfig(t *testing.T, publicURL string) string {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "household.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	if err := migrations.Run(db, testutil.Logger()); err != nil {
		t.Fatalf("migrations.Run() error = %v", err)
	}
	testutil.InsertUser(t, db, "alice")
	testutil.InsertUser(t, db, "bob")
	if err := db.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}

	cfg := "database:\n  path: " + dbPath + "\n"
	if publicURL != "" {
		cfg += "server:\n  public_url: " + publicURL + "\n"
	}
	path := filepath.Join(dir, "config.yaml")
	writeFile(t, path, cfg)
	return path
}

func device(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := runDevice(args, &out)
	return out.String(), err
}

func TestDeviceAddListRevoke(t *testing.T) {
	t.Parallel()

	cfg := deviceTestConfig(t, "https://pc.example.ts.net")

	out, err := device(t, "add", "-config", cfg, "alice", "Alice phone")
	if err != nil {
		t.Fatalf("device add error = %v", err)
	}
	if !regexp.MustCompile(`\n    [A-Z2-9]{5}-[A-Z2-9]{5}\n`).MatchString(out) {
		t.Errorf("device add output has no enrolment code:\n%s", out)
	}
	if !strings.Contains(out, "https://pc.example.ts.net") || !strings.Contains(out, "15 minutes") {
		t.Errorf("device add output lacks the server address or the validity:\n%s", out)
	}

	out, err = device(t, "list", "-config", cfg)
	if err != nil {
		t.Fatalf("device list error = %v", err)
	}
	for _, want := range []string{"Alice phone", "alice", "waiting", "never"} {
		if !strings.Contains(out, want) {
			t.Errorf("device list output lacks %q:\n%s", want, out)
		}
	}

	if _, err := device(t, "revoke", "-config", cfg, "alice phone"); err != nil {
		t.Fatalf("device revoke error = %v", err)
	}
	out, _ = device(t, "list", "-config", cfg)
	if !strings.Contains(out, "revoked") {
		t.Errorf("device list after revoke lacks revoked:\n%s", out)
	}
}

func TestDeviceAddWithoutPublicURLSaysHowToSetIt(t *testing.T) {
	t.Parallel()

	out, err := device(t, "add", "-config", deviceTestConfig(t, ""), "alice", "phone")
	if err != nil {
		t.Fatalf("device add error = %v", err)
	}
	if !strings.Contains(out, "server.public_url") {
		t.Errorf("output does not mention server.public_url:\n%s", out)
	}
}

func TestDeviceCommandErrorsAreClear(t *testing.T) {
	t.Parallel()

	cfg := deviceTestConfig(t, "")
	if _, err := device(t, "add", "-config", cfg, "alice", "phone"); err != nil {
		t.Fatalf("device add error = %v", err)
	}

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no command", nil, "missing device command"},
		{"unknown command", []string{"remove"}, "unknown device command"},
		{"add without a name", []string{"add", "-config", cfg, "alice"}, "takes 2 argument"},
		{"revoke without a name", []string{"revoke", "-config", cfg}, "takes 1 argument"},
		{"unknown flag", []string{"list", "-nope"}, "usage"},
		{"unknown user", []string{"add", "-config", cfg, "carol", "tablet"}, `no user "carol"`},
		{"second device", []string{"add", "-config", cfg, "alice", "tablet"}, "already has a device"},
		{"name in use", []string{"add", "-config", cfg, "bob", "phone"}, "already exists"},
		{"bad name", []string{"add", "-config", cfg, "bob", "-"}, "device name"},
		{"unknown device", []string{"revoke", "-config", cfg, "nothing"}, "no active device"},
		{"no database", []string{"list", "-config", writeConfigWithoutDatabase(t)}, "start the server once first"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := device(t, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("got %v, want an error containing %q", err, tt.want)
			}
		})
	}
}

func writeConfigWithoutDatabase(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	writeFile(t, path, "database:\n  path: "+filepath.Join(dir, "missing.db")+"\n")
	return path
}

func TestDeviceListWithNoDevices(t *testing.T) {
	t.Parallel()

	out, err := device(t, "list", "-config", deviceTestConfig(t, ""))
	if err != nil || !strings.Contains(out, "No devices") {
		t.Errorf("device list = %q, %v, want the empty message", out, err)
	}
}

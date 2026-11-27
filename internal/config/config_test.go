package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/modules"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestLoadAppliesDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := Load(writeConfig(t, ""))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.Host != "127.0.0.1" || cfg.Server.Port != 8080 || cfg.Database.Path != "household.db" || cfg.Seed.Dir != "seed" || cfg.Log.Level != "info" {
		t.Fatalf("defaults = %+v", cfg)
	}
}

func TestLoadReadsSeedDir(t *testing.T) {
	t.Parallel()

	cfg, err := Load(writeConfig(t, "seed:\n  dir: data/csv\n"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Seed.Dir != "data/csv" {
		t.Fatalf("Seed.Dir = %q, want data/csv", cfg.Seed.Dir)
	}
}

func TestLoadRunsEveryModuleByDefault(t *testing.T) {
	t.Parallel()

	cfg, err := Load(writeConfig(t, ""))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.ModuleSet().Names(); !slices.Equal(got, modules.All()) {
		t.Fatalf("ModuleSet().Names() = %v, want %v", got, modules.All())
	}
}

func TestLoadReadsTheModulesToRun(t *testing.T) {
	t.Parallel()

	cfg, err := Load(writeConfig(t, "modules: [goals, cashflow]\n"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := cfg.ModuleSet().Names(), []string{modules.Cashflow, modules.Goals}; !slices.Equal(got, want) {
		t.Fatalf("ModuleSet().Names() = %v, want %v", got, want)
	}
}

func TestLoadRejectsAnUnknownModule(t *testing.T) {
	t.Parallel()

	_, err := Load(writeConfig(t, "modules: [cashflow, goal]\n"))
	if err == nil || !strings.Contains(err.Error(), `"goal"`) {
		t.Fatalf("Load() error = %v, want it to name the unknown module", err)
	}
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
	}{
		{"unknown section", "servr:\n  port: 9000\n"},
		{"unknown key", "server:\n  secure_cookie: true\n"},
		{"removed projection section", "projection:\n  monthly_deposit_cents: 36600\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := Load(writeConfig(t, tt.content)); err == nil {
				t.Fatal("Load() succeeded, want an error naming the unknown key")
			}
		})
	}
}

func TestLoadPublicURL(t *testing.T) {
	t.Parallel()

	cfg, err := Load(writeConfig(t, "server:\n  public_url: https://pc.tailnet.ts.net\n"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.PublicURL != "https://pc.tailnet.ts.net" {
		t.Errorf("PublicURL = %q, want the configured address", cfg.Server.PublicURL)
	}

	for _, bad := range []string{"pc.tailnet.ts.net", "ftp://pc", "https://", "://"} {
		_, err := Load(writeConfig(t, "server:\n  public_url: \""+bad+"\"\n"))
		if err == nil || !strings.Contains(err.Error(), "server.public_url") {
			t.Errorf("Load() with public_url %q error = %v, want one naming server.public_url", bad, err)
		}
	}
}

func TestBackupsAreOffByDefault(t *testing.T) {
	t.Parallel()

	for name, content := range map[string]string{
		"no section":     "",
		"dir only":       "backup:\n  dir: backups\n",
		"keep zero":      "backup:\n  dir: backups\n  keep: 0\n",
		"keep zero only": "backup:\n  keep: 0\n",
	} {
		cfg, err := Load(writeConfig(t, content))
		if err != nil {
			t.Fatalf("%s: Load() error = %v", name, err)
		}
		if cfg.Backup.Enabled() {
			t.Errorf("%s: backups enabled, want off", name)
		}
	}
}

func TestLoadReadsTheBackupSettings(t *testing.T) {
	t.Parallel()

	cfg, err := Load(writeConfig(t, "backup:\n  dir: backups\n  keep: 14\n"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Backup.Enabled() || cfg.Backup.Dir != "backups" || cfg.Backup.Keep != 14 {
		t.Fatalf("backup = %+v", cfg.Backup)
	}
}

func TestLoadRejectsABadBackupSetup(t *testing.T) {
	t.Parallel()

	cases := map[string]struct{ content, want string }{
		"negative keep":      {"backup:\n  dir: b\n  keep: -1\n", "backup.keep"},
		"keep without a dir": {"backup:\n  keep: 7\n", "backup.dir is required"},
		"the seed directory": {"seed:\n  dir: data\nbackup:\n  dir: ./data/\n  keep: 7\n", "seed directory"},
		"the default seed":   {"backup:\n  dir: seed\n  keep: 7\n", "seed directory"},
	}
	for name, tc := range cases {
		_, err := Load(writeConfig(t, tc.content))
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: Load() error = %v, want it to mention %q", name, err, tc.want)
		}
	}
}

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/database"
	"github.com/tiagorocha94/household-webapp/internal/modules"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

// copyDir copies the files of one directory into another (no sub-directories).
func copyDir(t *testing.T, from, to string) {
	t.Helper()
	entries, err := os.ReadDir(from)
	if err != nil {
		t.Fatalf("read %s: %v", from, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(from, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		writeFile(t, filepath.Join(to, entry.Name()), string(data))
	}
}

// readDir returns the content of every file in a directory, by file name.
func readDir(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	files := map[string]string{}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		files[entry.Name()] = string(data)
	}
	return files
}

// The invented household in demo/ is what the smoke tests and "make demo" start
// from, so it must always import, sync and export, and starting again on what
// was exported must change nothing. The server rewrites the files it exports,
// so the test works on a copy.
func TestDemoDataStartsAndIsRepeatable(t *testing.T) {
	t.Parallel()

	db, err := database.Open(filepath.Join(t.TempDir(), "demo-test.db"))
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })

	dir := t.TempDir()
	copyDir(t, filepath.Join("..", "..", "demo"), dir)

	if err := prepareData(db, dir, modules.Set{}, testutil.Logger()); err != nil {
		t.Fatalf("first prepareData() on the demo data error = %v", err)
	}
	first := readDir(t, dir)
	if len(first) == 0 {
		t.Fatal("no files after the first start")
	}

	if err := prepareData(db, dir, modules.Set{}, testutil.Logger()); err != nil {
		t.Fatalf("second prepareData() error = %v", err)
	}
	second := readDir(t, dir)
	for name, want := range first {
		if second[name] != want {
			t.Errorf("%s changed on the second start", name)
		}
	}
}

// Every module must run alone and in small groups: the demo data starts and is
// repeatable for each of these sets, the files of a module that does not run
// are not touched, and the files of one that does are exported.
func TestDemoDataStartsWithAnySetOfModules(t *testing.T) {
	t.Parallel()

	prefixes := map[string]string{
		modules.Cashflow:    "cashflow_",
		modules.Goals:       "goal_",
		modules.Savings:     "savings_",
		modules.Credits:     "credit_",
		modules.Investments: "investment_",
	}
	original := readDir(t, filepath.Join("..", "..", "demo"))

	for _, names := range [][]string{
		{modules.Cashflow},
		{modules.Goals},
		{modules.Savings},
		{modules.Credits},
		{modules.Investments},
		{modules.Cashflow, modules.Goals},
		{modules.Savings, modules.Credits, modules.Investments},
	} {
		t.Run(strings.Join(names, "+"), func(t *testing.T) {
			t.Parallel()

			set, err := modules.NewSet(names)
			if err != nil {
				t.Fatalf("modules.NewSet(%v) error = %v", names, err)
			}
			db, err := database.Open(filepath.Join(t.TempDir(), "modules-test.db"))
			if err != nil {
				t.Fatalf("database.Open() error = %v", err)
			}
			t.Cleanup(func() { db.Close() })
			dir := t.TempDir()
			copyDir(t, filepath.Join("..", "..", "demo"), dir)

			if err := prepareData(db, dir, set, testutil.Logger()); err != nil {
				t.Fatalf("first prepareData() error = %v", err)
			}
			first := readDir(t, dir)
			if err := prepareData(db, dir, set, testutil.Logger()); err != nil {
				t.Fatalf("second prepareData() error = %v", err)
			}
			second := readDir(t, dir)

			for name, want := range first {
				if second[name] != want {
					t.Errorf("%s changed on the second start", name)
				}
			}
			for module, prefix := range prefixes {
				if set.Has(module) {
					continue
				}
				for name, want := range original {
					if strings.HasPrefix(name, prefix) && second[name] != want {
						t.Errorf("%s belongs to the %s module, which does not run, but it changed", name, module)
					}
				}
			}
		})
	}
}

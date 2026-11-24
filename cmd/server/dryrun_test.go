package main

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/database"
	"github.com/tiagorocha94/household-webapp/internal/modules"
	"github.com/tiagorocha94/household-webapp/internal/seed"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

const dryRunGoalsCSV = "account,name,target_cents,start_month,end_month,status\nHousehold,Trip,10000,2020-01,2020-04,active\n"

func writeDryRunSeed(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "users.csv"), "username,password\nowner,"+testutil.PasswordHash+"\n")
	writeFile(t, filepath.Join(dir, "goal_accounts.csv"), "name,monthly_deposit_cents\nHousehold,36600\n")
	writeFile(t, filepath.Join(dir, "goal_account_goals.csv"), dryRunGoalsCSV)
	return dir
}

func TestPreviewStartupReportsChangesAndWritesNothing(t *testing.T) {
	t.Parallel()

	dir := writeDryRunSeed(t)
	dbPath := filepath.Join(t.TempDir(), "household.db")

	var out bytes.Buffer
	if err := previewStartup(&out, dbPath, dir, modules.Set{}, testutil.Logger()); err != nil {
		t.Fatalf("previewStartup() error = %v", err)
	}

	report := out.String()
	for _, want := range []string{
		"Dry run: nothing was written.",
		"Migrations that would run:",
		"0002_create_goal_module.sql",
		"goal_accounts: +1 -0",
		"goal_account_movements: +",
		"goal_account_movements.csv: new",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("report does not contain %q:\n%s", want, report)
		}
	}
	if strings.Contains(report, testutil.PasswordHash) {
		t.Errorf("report shows a password:\n%s", report)
	}

	if _, err := os.Stat(dbPath); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("database file after dry run: err = %v, want it not to exist", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "goal_account_movements.csv")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("goal_account_movements.csv after dry run: err = %v, want it not to exist", err)
	}
	goalsCSV, err := os.ReadFile(filepath.Join(dir, "goal_account_goals.csv"))
	if err != nil || string(goalsCSV) != dryRunGoalsCSV {
		t.Errorf("goal_account_goals.csv = %q, %v, want it untouched", goalsCSV, err)
	}
}

func TestPreviewStartupOnAnUpToDateDatabaseChangesNothing(t *testing.T) {
	t.Parallel()

	dir := writeDryRunSeed(t)
	dbPath := filepath.Join(t.TempDir(), "household.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := prepareData(db, dir, modules.Set{}, testutil.Logger()); err != nil {
		t.Fatalf("prepareData() error = %v", err)
	}

	var out bytes.Buffer
	if err := previewStartup(&out, dbPath, dir, modules.Set{}, testutil.Logger()); err != nil {
		t.Fatalf("previewStartup() error = %v", err)
	}
	if want := "Nothing would change."; !strings.Contains(out.String(), want) {
		t.Fatalf("report = %q, want it to contain %q", out.String(), want)
	}
}

func TestPreviewStartupSeesAChangedCSVFile(t *testing.T) {
	t.Parallel()

	dir := writeDryRunSeed(t)
	dbPath := filepath.Join(t.TempDir(), "household.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := prepareData(db, dir, modules.Set{}, testutil.Logger()); err != nil {
		t.Fatalf("prepareData() error = %v", err)
	}

	writeFile(t, filepath.Join(dir, "goal_accounts.csv"), "name,monthly_deposit_cents\nHousehold,50000\n")
	var out bytes.Buffer
	if err := previewStartup(&out, dbPath, dir, modules.Set{}, testutil.Logger()); err != nil {
		t.Fatalf("previewStartup() error = %v", err)
	}
	report := out.String()
	for _, want := range []string{"goal_accounts: +1 -1", "+ Household | 50000", "- Household | 36600"} {
		if !strings.Contains(report, want) {
			t.Errorf("report does not contain %q:\n%s", want, report)
		}
	}
}

func TestStartupPreviewReportFormatsEverySection(t *testing.T) {
	t.Parallel()

	rows := make([]string, maxExampleRows+2)
	for i := range rows {
		rows[i] = "row"
	}
	preview := &startupPreview{
		migrations: []string{"0002_x.sql"},
		tables:     []seed.TableChange{{Table: "goal_accounts", Added: rows, Removed: []string{"old"}}},
		files: []fileChange{
			{name: "a.csv", isNew: true, after: 3},
			{name: "b.csv", before: 2, after: 2},
			{name: "c.csv", before: 2, after: 5},
		},
	}

	report := preview.report()
	for _, want := range []string{
		"  0002_x.sql\n",
		"  goal_accounts: +12 -1\n",
		"    ... and 2 more\n",
		"    - old\n",
		"  a.csv: new, 3 rows\n",
		"  b.csv: 2 rows, rewritten (order or formatting)\n",
		"  c.csv: 2 -> 5 rows\n",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("report does not contain %q:\n%s", want, report)
		}
	}
}

func TestDataRowsCountsLinesAfterTheHeader(t *testing.T) {
	t.Parallel()

	for content, want := range map[string]int{"": 0, "h\n": 0, "h\na\nb\n": 2} {
		if got := dataRows([]byte(content)); got != want {
			t.Errorf("dataRows(%q) = %d, want %d", content, got, want)
		}
	}
}

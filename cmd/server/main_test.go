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

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestPrepareDataIsRepeatableAndExports(t *testing.T) {
	t.Parallel()

	db, err := database.Open(filepath.Join(t.TempDir(), "startup-test.db"))
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "users.csv"), "username,password\nowner,"+testutil.PasswordHash+"\n")
	writeFile(t, filepath.Join(dir, "goal_accounts.csv"), "name,monthly_deposit_cents\nHousehold,36600\n")
	writeFile(t, filepath.Join(dir, "goal_account_goals.csv"),
		"account,name,target_cents,start_month,end_month,status\nHousehold,Trip,10000,2020-01,2020-04,active\n")

	if err := prepareData(db, dir, modules.Set{}, testutil.Logger()); err != nil {
		t.Fatalf("first prepareData() error = %v", err)
	}

	// Sync allocated the past months and export wrote that back to the CSV
	// files. The goal ended long ago, but it stays active until it is completed
	// by hand.
	movements, err := os.ReadFile(filepath.Join(dir, "goal_account_movements.csv"))
	if err != nil {
		t.Fatalf("read exported movements: %v", err)
	}
	if len(movements) == 0 {
		t.Fatal("exported movements.csv is empty")
	}
	goalsCSV, err := os.ReadFile(filepath.Join(dir, "goal_account_goals.csv"))
	if err != nil {
		t.Fatalf("read exported goals: %v", err)
	}
	if want := "Household,Trip,10000,2020-01,2020-04,active"; !strings.Contains(string(goalsCSV), want) {
		t.Fatalf("exported goals = %q, want a row %q", goalsCSV, want)
	}

	// Running again against the exported files changes nothing.
	if err := prepareData(db, dir, modules.Set{}, testutil.Logger()); err != nil {
		t.Fatalf("second prepareData() error = %v", err)
	}
	again, err := os.ReadFile(filepath.Join(dir, "goal_account_movements.csv"))
	if err != nil {
		t.Fatalf("read movements after second run: %v", err)
	}
	if string(again) != string(movements) {
		t.Fatalf("movements changed on second run:\n%s\n---\n%s", movements, again)
	}
}

func TestPrepareDataStopsBeforeExportOnBadCSV(t *testing.T) {
	t.Parallel()

	db, err := database.Open(filepath.Join(t.TempDir(), "startup-bad-test.db"))
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "users.csv"), "wrong,header\nowner,"+testutil.PasswordHash+"\n")
	writeFile(t, filepath.Join(dir, "goal_accounts.csv"), "name,monthly_deposit_cents\nKeep,36600\n")

	if err := prepareData(db, dir, modules.Set{}, testutil.Logger()); err == nil {
		t.Fatal("prepareData() succeeded, want header error")
	}
	got, err := os.ReadFile(filepath.Join(dir, "goal_accounts.csv"))
	if err != nil || string(got) != "name,monthly_deposit_cents\nKeep,36600\n" {
		t.Fatalf("goal_accounts.csv = %q, %v, want it untouched", got, err)
	}
}

func TestImportRejectsObsoleteMoneyHeaders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		file   string
		header []string
	}{
		{"goal target", "goal_account_goals.csv", []string{"account", "name", "target_amount", "start_month", "end_month", "status"}},
		{"goal movement", "goal_account_movements.csv", []string{"account", "type", "goal", "month", "amount", "description"}},
		{"credit principal", "credit_contracts.csv", []string{"name", "category", "contract_number", "lender", "original_principal", "start_month", "expected_end_month", "notes"}},
		{"credit payment", "credit_payments.csv", []string{"credit", "month", "type", "principal_amount", "interest_amount", "description"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, tt.file), strings.Join(tt.header, ",")+"\n")
			err := newSeedRunner(testutil.Logger(), modules.Set{}).Import(testutil.NewDB(t), dir)
			if err == nil || !strings.Contains(err.Error(), "header") {
				t.Fatalf("Import() error = %v, want header mismatch", err)
			}
		})
	}
}

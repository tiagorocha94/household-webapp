package goals_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

func TestGoalImportIsIdempotentAndMirrorsCSV(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	dir := t.TempDir()

	goalsHeader := []string{"account", "name", "target_cents", "start_month", "end_month", "status"}
	movementsHeader := []string{"account", "type", "goal", "month", "amount_cents", "description"}

	writeTestCSV(t, filepath.Join(dir, "users.csv"), [][]string{
		{"username", "password"}, {"owner", testutil.PasswordHash},
	})
	writeTestCSV(t, filepath.Join(dir, "goal_accounts.csv"), [][]string{
		{"name", "monthly_deposit_cents"}, {"Household", "36600"},
	})
	writeTestCSV(t, filepath.Join(dir, "goal_account_goals.csv"), [][]string{
		goalsHeader,
		{"Household", "A", "10000", "2026-01", "2026-04", "active"},
		{"Household", "B", "5000", "2026-01", "2026-02", "active"},
	})
	writeTestCSV(t, filepath.Join(dir, "goal_account_movements.csv"), [][]string{
		movementsHeader,
		{"Household", "deposit", "", "2026-01", "20000", ""},
		{"Household", "allocation", "A", "2026-01", "2500", ""},
		{"Household", "allocation", "B", "2026-01", "2500", ""},
	})

	if err := newRunner().Import(db, dir); err != nil {
		t.Fatalf("newRunner().Import() error = %v", err)
	}

	const oldStamp = "2000-01-01 00:00:00"
	if _, err := db.Exec("UPDATE goal_account_goals SET updated_at = ?", oldStamp); err != nil {
		t.Fatalf("set updated_at: %v", err)
	}
	snapshot := func() []string {
		return []string{
			queryString(t, db, "SELECT group_concat(id) FROM goal_accounts"),
			queryString(t, db, "SELECT group_concat(id) FROM (SELECT id FROM goal_account_goals ORDER BY id)"),
			queryString(t, db, "SELECT group_concat(id) FROM (SELECT id FROM goal_account_movements ORDER BY id)"),
			queryString(t, db, "SELECT password_hash FROM users WHERE username = 'owner'"),
			queryString(t, db, "SELECT group_concat(CAST(updated_at AS TEXT)) FROM goal_account_goals"),
		}
	}
	before := snapshot()

	if err := newRunner().Import(db, dir); err != nil {
		t.Fatalf("second newRunner().Import() error = %v", err)
	}
	after := snapshot()
	for i := range before {
		if before[i] != after[i] {
			t.Errorf("snapshot[%d] changed on re-import: %q -> %q", i, before[i], after[i])
		}
	}
	if after[4] != oldStamp+","+oldStamp {
		t.Errorf("updated_at = %q, want untouched", after[4])
	}

	// Removing rows from the CSV removes them from the database.
	writeTestCSV(t, filepath.Join(dir, "goal_account_goals.csv"), [][]string{
		goalsHeader,
		{"Household", "A", "10000", "2026-01", "2026-04", "active"},
	})
	writeTestCSV(t, filepath.Join(dir, "goal_account_movements.csv"), [][]string{
		movementsHeader,
		{"Household", "deposit", "", "2026-01", "20000", ""},
		{"Household", "allocation", "A", "2026-01", "2500", ""},
	})
	if err := newRunner().Import(db, dir); err != nil {
		t.Fatalf("pruning newRunner().Import() error = %v", err)
	}
	if got := queryString(t, db, "SELECT COUNT(*) FROM goal_account_goals"); got != "1" {
		t.Errorf("goals after prune = %s, want 1", got)
	}
	if got := queryString(t, db, "SELECT COUNT(*) FROM goal_account_movements"); got != "2" {
		t.Errorf("movements after prune = %s, want 2", got)
	}
	if got := queryString(t, db, "SELECT group_concat(id) FROM goal_accounts"); got != before[0] {
		t.Errorf("account id = %s, want %s", got, before[0])
	}

	// A header-only file clears its table.
	writeTestCSV(t, filepath.Join(dir, "goal_account_movements.csv"), [][]string{movementsHeader})
	if err := newRunner().Import(db, dir); err != nil {
		t.Fatalf("clearing newRunner().Import() error = %v", err)
	}
	if got := queryString(t, db, "SELECT COUNT(*) FROM goal_account_movements"); got != "0" {
		t.Errorf("movements after empty file = %s, want 0", got)
	}
}

func TestGoalCSVUsesRawIntegerCents(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	seedDir := t.TempDir()
	writeTestCSV(t, filepath.Join(seedDir, "goal_accounts.csv"), [][]string{
		{"name", "monthly_deposit_cents"},
		{"Household", "36600"},
	})
	writeTestCSV(t, filepath.Join(seedDir, "goal_account_goals.csv"), [][]string{
		{"account", "name", "target_cents", "start_month", "end_month", "status"},
		{"Household", "Trip", "12345", "2026-01", "2026-12", "active"},
	})
	writeTestCSV(t, filepath.Join(seedDir, "goal_account_movements.csv"), [][]string{
		{"account", "type", "goal", "month", "amount_cents", "description"},
		{"Household", "deposit", "", "2026-01", "20000", "Deposit"},
		{"Household", "allocation", "Trip", "2026-01", "12345", "Allocation"},
	})

	if err := newRunner().Import(db, seedDir); err != nil {
		t.Fatalf("newRunner().Import() error = %v", err)
	}

	var target, movementTotal int64
	if err := db.QueryRow("SELECT target_cents FROM goal_account_goals WHERE name = 'Trip'").Scan(&target); err != nil {
		t.Fatalf("query goal target: %v", err)
	}
	if err := db.QueryRow("SELECT SUM(amount_cents) FROM goal_account_movements").Scan(&movementTotal); err != nil {
		t.Fatalf("query movement total: %v", err)
	}
	if target != 12345 || movementTotal != 32345 {
		t.Fatalf("imported cents = (target %d, movements %d), want (12345, 32345)", target, movementTotal)
	}

	exportDir := t.TempDir()
	if err := newRunner().Export(db, exportDir); err != nil {
		t.Fatalf("newRunner().Export() error = %v", err)
	}
	assertCSVData(t, filepath.Join(exportDir, "goal_account_goals.csv"),
		[]string{"account", "name", "target_cents", "start_month", "end_month", "status"},
		[][]string{{"Household", "Trip", "12345", "2026-01", "2026-12", "active"}},
	)
	assertCSVData(t, filepath.Join(exportDir, "goal_account_movements.csv"),
		[]string{"account", "type", "goal", "month", "amount_cents", "description"},
		[][]string{
			{"Household", "allocation", "Trip", "2026-01", "12345", "Allocation"},
			{"Household", "deposit", "", "2026-01", "20000", "Deposit"},
		},
	)
}

func TestGoalAccountDepositIsMirroredAndExported(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	dir := t.TempDir()
	accountsPath := filepath.Join(dir, "goal_accounts.csv")

	writeTestCSV(t, accountsPath, [][]string{{"name", "monthly_deposit_cents"}, {"Household", "36600"}})
	if err := newRunner().Import(db, dir); err != nil {
		t.Fatalf("newRunner().Import() error = %v", err)
	}
	id := queryString(t, db, "SELECT id FROM goal_accounts WHERE name = 'Household'")

	writeTestCSV(t, accountsPath, [][]string{{"name", "monthly_deposit_cents"}, {"Household", "50000"}})
	if err := newRunner().Import(db, dir); err != nil {
		t.Fatalf("second newRunner().Import() error = %v", err)
	}
	if got := queryString(t, db, "SELECT CAST(monthly_deposit_cents AS TEXT) FROM goal_accounts"); got != "50000" {
		t.Errorf("monthly_deposit_cents = %s, want 50000 after the CSV changed", got)
	}
	if got := queryString(t, db, "SELECT id FROM goal_accounts WHERE name = 'Household'"); got != id {
		t.Errorf("account id = %s, want %s: a changed deposit must not recreate the account", got, id)
	}

	exportDir := t.TempDir()
	if err := newRunner().Export(db, exportDir); err != nil {
		t.Fatalf("newRunner().Export() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(exportDir, "goal_accounts.csv"))
	if err != nil || string(got) != "name,monthly_deposit_cents\nHousehold,50000\n" {
		t.Errorf("exported goal_accounts.csv = %q, %v", got, err)
	}
}

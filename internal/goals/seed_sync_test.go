package goals_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

var boilerAllocations = [][]string{
	{"Household", "allocation", "Boiler", "2020-01", "10000", ""},
	{"Household", "allocation", "Boiler", "2020-02", "10000", ""},
}

func TestSyncFinalAllocationAbsorbsDrift(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	dir := t.TempDir()

	writeTestCSV(t, filepath.Join(dir, "goal_accounts.csv"), [][]string{
		{"name", "monthly_deposit_cents"}, {"Household", "36600"},
	})
	writeTestCSV(t, filepath.Join(dir, "goal_account_goals.csv"), [][]string{
		{"account", "name", "target_cents", "start_month", "end_month", "status"},
		{"Household", "Trip", "10000", "2020-01", "2020-03", "active"},
	})
	// Split(10000, 3) is 3334/3333/3333, but the first month holds 3333.
	writeTestCSV(t, filepath.Join(dir, "goal_account_movements.csv"), [][]string{
		{"account", "type", "goal", "month", "amount_cents", "description"},
		{"Household", "allocation", "Trip", "2020-01", "3333", ""},
	})

	if err := newRunner().Import(db, dir); err != nil {
		t.Fatalf("newRunner().Import() error = %v", err)
	}
	if err := newRunner().Sync(db); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	got := queryString(t, db,
		"SELECT CAST(SUM(amount_cents) AS TEXT) FROM goal_account_movements WHERE type = 'allocation'")
	if got != "10000" {
		t.Errorf("allocated = %s, want 10000", got)
	}
	// The goal ended long ago and is fully allocated, yet it stays active until
	// someone completes it: its expenses may still come.
	if status := queryString(t, db, "SELECT status FROM goal_account_goals"); status != "active" {
		t.Errorf("status = %s, want active", status)
	}
	if got := queryString(t, db, "SELECT COUNT(*) FROM goal_account_movements WHERE type IN ('release', 'cover')"); got != "0" {
		t.Errorf("closing movements = %s, want 0", got)
	}
}

// syncGoal imports one goal with its movements, syncs, and returns the database.
func syncGoal(t *testing.T, status string, movements [][]string) *sql.DB {
	t.Helper()

	db := testutil.NewDB(t)
	dir := t.TempDir()
	writeTestCSV(t, filepath.Join(dir, "goal_accounts.csv"), [][]string{
		{"name", "monthly_deposit_cents"}, {"Household", "0"},
	})
	writeTestCSV(t, filepath.Join(dir, "goal_account_goals.csv"), [][]string{
		{"account", "name", "target_cents", "start_month", "end_month", "status"},
		{"Household", "Boiler", "20000", "2020-01", "2020-02", status},
	})
	writeTestCSV(t, filepath.Join(dir, "goal_account_movements.csv"),
		append([][]string{{"account", "type", "goal", "month", "amount_cents", "description"}}, movements...))
	if err := newRunner().Import(db, dir); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if err := newRunner().Sync(db); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	return db
}

func TestSyncCompletedGoalReleasesLeftover(t *testing.T) {
	t.Parallel()

	movements := append([][]string{{"Household", "expense", "Boiler", "2020-02", "15000", ""}}, boilerAllocations...)
	db := syncGoal(t, "completed", movements)

	if got := queryString(t, db, "SELECT type || ':' || amount_cents FROM goal_account_movements WHERE type IN ('release', 'cover')"); got != "release:5000" {
		t.Errorf("closing movement = %s, want release:5000", got)
	}
	if got := queryString(t, db, "SELECT CAST(closing_balance_cents AS TEXT) FROM goal_account_goals"); got != "5000" {
		t.Errorf("closing balance = %s, want 5000", got)
	}
	if got := queryString(t, db, "SELECT CAST(goal_balance_cents AS TEXT) FROM goal_account_goal_summary_view"); got != "0" {
		t.Errorf("goal balance = %s, want 0", got)
	}
	if got := queryString(t, db, "SELECT CAST(unallocated_cents AS TEXT) FROM goal_account_balance_view"); got != "-15000" {
		t.Errorf("unallocated = %s, want -15000 (allocations 20000 less the 5000 released)", got)
	}
}

func TestSyncCompletedGoalCoversOverspend(t *testing.T) {
	t.Parallel()

	movements := append([][]string{{"Household", "expense", "Boiler", "2020-02", "23500", ""}}, boilerAllocations...)
	db := syncGoal(t, "completed", movements)

	if got := queryString(t, db, "SELECT type || ':' || amount_cents FROM goal_account_movements WHERE type IN ('release', 'cover')"); got != "cover:3500" {
		t.Errorf("closing movement = %s, want cover:3500", got)
	}
	if got := queryString(t, db, "SELECT CAST(closing_balance_cents AS TEXT) FROM goal_account_goals"); got != "-3500" {
		t.Errorf("closing balance = %s, want -3500", got)
	}
	if got := queryString(t, db, "SELECT CAST(goal_balance_cents AS TEXT) FROM goal_account_goal_summary_view"); got != "0" {
		t.Errorf("goal balance = %s, want 0", got)
	}
	if got := queryString(t, db, "SELECT CAST(unallocated_cents AS TEXT) FROM goal_account_balance_view"); got != "-23500" {
		t.Errorf("unallocated = %s, want -23500 (allocations 20000 plus the 3500 cover)", got)
	}
}

func TestSyncSettlesACompletedGoalOnce(t *testing.T) {
	t.Parallel()

	db := syncGoal(t, "completed", boilerAllocations)
	if err := newRunner().Sync(db); err != nil {
		t.Fatalf("second Sync() error = %v", err)
	}
	if got := queryString(t, db, "SELECT COUNT(*) FROM goal_account_movements WHERE type IN ('release', 'cover')"); got != "1" {
		t.Errorf("closing movements = %s, want 1", got)
	}
}

func TestSyncKeepsAnOldRefundAsTheClosingMovement(t *testing.T) {
	t.Parallel()

	// A goal the old sync completed on its own: the refund is already recorded.
	movements := append([][]string{
		{"Household", "expense", "Boiler", "2020-02", "15000", ""},
		{"Household", "refund", "Boiler", "2020-03", "5000", "Goal completed: remaining balance"},
	}, boilerAllocations...)
	db := syncGoal(t, "completed", movements)

	if got := queryString(t, db, "SELECT type || ':' || amount_cents FROM goal_account_movements WHERE type NOT IN ('allocation', 'expense')"); got != "release:5000" {
		t.Errorf("closing movement = %s, want release:5000", got)
	}
	if got := queryString(t, db, "SELECT CAST(closing_balance_cents AS TEXT) FROM goal_account_goals"); got != "5000" {
		t.Errorf("closing balance = %s, want 5000", got)
	}
}

func TestSyncOpenGoalHasNoClosingBalance(t *testing.T) {
	t.Parallel()

	db := syncGoal(t, "active", boilerAllocations)
	if got := queryString(t, db, "SELECT COALESCE(CAST(closing_balance_cents AS TEXT), 'none') FROM goal_account_goals"); got != "none" {
		t.Errorf("closing balance = %s, want none", got)
	}
}

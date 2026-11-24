package service

import (
	"database/sql"
	"testing"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/goals"
	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/month"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

// createTestAccount inserts a goal account with no recurring deposit and
// returns its ID.
func createTestAccount(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	return createAccountWithDeposit(t, db, "Test", 0)
}

// createAccountWithDeposit inserts a goal account that assumes the given
// monthly deposit in its projection and returns its ID.
func createAccountWithDeposit(t *testing.T, db *sql.DB, name string, deposit money.Cents) int64 {
	t.Helper()
	return testutil.MustExec(t, db,
		"INSERT INTO goal_accounts (name, monthly_deposit_cents) VALUES (?, ?)", name, deposit)
}

func createGoal(t *testing.T, db *sql.DB, accountID int64, name string, target money.Cents, startMonth, endMonth, status string) int64 {
	t.Helper()

	res, err := db.Exec(
		"INSERT INTO goal_account_goals (account_id, name, target_cents, start_month, end_month, status) VALUES (?, ?, ?, ?, ?, ?)",
		accountID, name, target, startMonth, endMonth, status,
	)
	if err != nil {
		t.Fatalf("create goal: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("get goal id: %v", err)
	}
	return id
}

func insertMovement(t *testing.T, db *sql.DB, accountID int64, goalID *int64, movType, month string, amount money.Cents) {
	t.Helper()
	testutil.MustExec(t, db,
		"INSERT INTO goal_account_movements (account_id, goal_id, type, amount_cents, month) VALUES (?, ?, ?, ?, ?)",
		accountID, goalID, movType, amount, month,
	)
}

// monthOffset returns the YYYY-MM string for `months` months from now.
func monthOffset(months int) string {
	return month.Offset(time.Now(), months)
}

func findMonth(t *testing.T, projection []goals.ProjectionMonth, month string) goals.ProjectionMonth {
	t.Helper()
	for _, pm := range projection {
		if pm.Month == month {
			return pm
		}
	}
	t.Fatalf("month %s not found in projection", month)
	return goals.ProjectionMonth{}
}

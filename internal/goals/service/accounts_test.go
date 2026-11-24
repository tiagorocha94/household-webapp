package service

import (
	"errors"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/goals"
	"github.com/tiagorocha94/household-webapp/internal/goals/store"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

func TestReadsReportMissingAccountsAndGoals(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	accountID := createAccountWithDeposit(t, db, "Household", 36600)
	goalID := createGoal(t, db, accountID, "Trip", 10000, "2026-01", "2026-02", goals.StatusActive)
	svc := New(store.New(db))

	account, err := svc.GetAccount(accountID)
	if err != nil || account.Name != "Household" || account.MonthlyDepositCents != 36600 {
		t.Errorf("GetAccount() = %+v, %v, want Household with a deposit of 36600", account, err)
	}
	if _, err := svc.GetGoalSummary(goalID); err != nil {
		t.Errorf("GetGoalSummary() error = %v", err)
	}
	list, err := svc.ListAccounts()
	if err != nil || len(list) != 1 {
		t.Errorf("ListAccounts() = %v, %v, want one account", list, err)
	}

	missing := map[string]error{}
	_, missing["GetAccount"] = svc.GetAccount(9999)
	_, missing["GetBalance"] = svc.GetBalance(9999)
	_, missing["ListGoalSummaries"] = svc.ListGoalSummaries(9999)
	_, missing["ListMovements"] = svc.ListMovements(9999)
	_, missing["GetProjection"] = svc.GetProjection(9999)
	_, missing["GetGoalSummary"] = svc.GetGoalSummary(9999)
	_, missing["ListGoalMovements"] = svc.ListGoalMovements(9999)
	for name, err := range missing {
		if !errors.Is(err, apperr.ErrNotFound) {
			t.Errorf("%s(missing) error = %v, want not found", name, err)
		}
	}
}

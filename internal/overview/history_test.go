package overview

import (
	"reflect"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/cashflow"
	"github.com/tiagorocha94/household-webapp/internal/credits"
	"github.com/tiagorocha94/household-webapp/internal/goals"
	"github.com/tiagorocha94/household-webapp/internal/investments"
	"github.com/tiagorocha94/household-webapp/internal/savings"
)

type historyGoals struct{}
type historyCredits struct{}
type historyInvestments struct{}
type historySavings struct{}
type historyCashflow struct{ none bool }

func (historyGoals) ListAccounts() ([]goals.Account, error) {
	return []goals.Account{{ID: 1, Name: "Household"}}, nil
}

func (historyGoals) ListGoalSummaries(int64) ([]goals.GoalSummary, error) {
	return []goals.GoalSummary{
		{GoalID: 10, Name: "Car", Status: goals.StatusActive},
		{GoalID: 11, Name: "Old", Status: "completed"},
	}, nil
}

func (historyGoals) ListMovements(int64) ([]goals.Movement, error) {
	car, old := int64(10), int64(11)
	return []goals.Movement{
		{GoalID: &car, Type: goals.MovementAllocation, AmountCents: 10_000, Month: "2026-01"},
		{GoalID: &car, Type: goals.MovementAllocation, AmountCents: 10_000, Month: "2026-02"},
		{GoalID: &car, Type: goals.MovementExpense, AmountCents: 4_000, Month: "2026-03"},
		{GoalID: &old, Type: goals.MovementAllocation, AmountCents: 99_000, Month: "2026-01"},
		{Type: goals.MovementDeposit, AmountCents: 50_000, Month: "2026-01"},
	}, nil
}

func (historyCredits) ListContracts() ([]credits.ContractSummary, error) {
	return []credits.ContractSummary{
		{Contract: credits.Contract{ID: 1, Name: "Mortgage", OriginalPrincipalCents: 100_000, StartMonth: "2026-01"}},
		{Contract: credits.Contract{ID: 2, Name: "Car loan", OriginalPrincipalCents: 20_000, StartMonth: "2026-03"}},
	}, nil
}

func (historyCredits) ListPayments(id int64) ([]credits.Payment, error) {
	if id == 1 {
		return []credits.Payment{
			{Month: "2026-01", PrincipalCents: 1_000},
			{Month: "2026-02", PrincipalCents: 1_000},
			{Month: "2026-03", PrincipalCents: 1_000},
		}, nil
	}
	return nil, nil
}

func (historyInvestments) ListEntities() ([]investments.EntitySummary, error) {
	return []investments.EntitySummary{{Entity: investments.Entity{ID: 1}}}, nil
}

func (historyInvestments) ListInstrumentsByEntity(int64) ([]investments.InstrumentSummary, error) {
	return []investments.InstrumentSummary{{Instrument: investments.Instrument{ID: 7}}}, nil
}

func (historyInvestments) ListSnapshots(int64) ([]investments.Snapshot, error) {
	return []investments.Snapshot{
		{ID: 1, Month: "2026-02", ValueCents: 30_000},
		{ID: 2, Month: "2026-03", ValueCents: 33_000},
	}, nil
}

func (historySavings) ListAccounts() ([]savings.AccountSummary, error) {
	return []savings.AccountSummary{{Account: savings.Account{ID: 1, Name: "Emergency"}}}, nil
}

func (historySavings) ListMovements(int64) ([]savings.Movement, error) {
	return []savings.Movement{
		{Month: "2026-01", Type: savings.TypeDeposit, AmountCents: 20_000},
		{Month: "2026-02", Type: savings.TypeDeposit, AmountCents: 20_000},
	}, nil
}

func (historySavings) ListSnapshots(int64) ([]savings.Snapshot, error) {
	return []savings.Snapshot{{ID: 1, Month: "2026-02", ValueCents: 40_500}}, nil
}

func (historyCashflow) ListMonths() ([]cashflow.MonthSummary, error) {
	// Newest first, with February missing: it has no cashflow.
	return []cashflow.MonthSummary{{Month: "2026-03"}, {Month: "2026-01"}}, nil
}

func (c historyCashflow) CashBalances(months []string) ([]cashflow.CashBalance, error) {
	if c.none {
		return nil, nil
	}
	var out []cashflow.CashBalance
	for _, m := range months {
		out = append(out, cashflow.CashBalance{User: "alice", Month: m, BalanceCents: 5_000}, cashflow.CashBalance{User: "bob", Month: m, BalanceCents: -1_000})
	}
	return out, nil
}

func TestNetWorthHistoryByMonth(t *testing.T) {
	t.Parallel()

	h := NewHistory(historyGoals{}, historyCredits{}, historyInvestments{}, historySavings{}, historyCashflow{})
	got, err := h.NetWorth()
	if err != nil {
		t.Fatalf("NetWorth: %v", err)
	}

	want := []NetWorthMonth{
		// January: savings are the deposits (no snapshot yet), no investments,
		// 10000 in the active goal, the mortgage after one payment.
		{Month: "2026-01", SavingsCents: 20_000, GoalsCents: 10_000, CashCents: 4_000, AssetsCents: 34_000, CreditCents: 99_000, NetCents: 34_000 - 99_000},
		// March: the February snapshot still holds, the goal spent 4000, the
		// car loan has started and the mortgage is after three payments.
		{Month: "2026-03", SavingsCents: 40_500, InvestmentsCents: 33_000, GoalsCents: 16_000, CashCents: 4_000, AssetsCents: 93_500, CreditCents: 97_000 + 20_000, NetCents: 93_500 - 117_000},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NetWorth() = %+v, want %+v", got, want)
	}
}

func TestNetWorthHistoryWithoutCashBalancesLeavesCashOut(t *testing.T) {
	t.Parallel()

	h := NewHistory(historyGoals{}, historyCredits{}, historyInvestments{}, historySavings{}, historyCashflow{none: true})
	got, err := h.NetWorth()
	if err != nil {
		t.Fatalf("NetWorth: %v", err)
	}
	if len(got) != 2 || got[0].CashCents != 0 || got[1].CashCents != 0 {
		t.Errorf("NetWorth() = %+v, want no cash in either month", got)
	}
}

func TestNetWorthHistoryNeedsCashflowForItsMonths(t *testing.T) {
	t.Parallel()

	got, err := NewHistory(historyGoals{}, historyCredits{}, historyInvestments{}, historySavings{}, nil).NetWorth()
	if err != nil {
		t.Fatalf("NetWorth: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("NetWorth() = %#v, want an empty list", got)
	}
}

func TestNetWorthHistoryLeavesOutTheModulesThatDoNotRun(t *testing.T) {
	t.Parallel()

	got, err := NewHistory(nil, nil, nil, nil, historyCashflow{}).NetWorth()
	if err != nil {
		t.Fatalf("NetWorth: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("NetWorth() = %+v, want the two months with cashflow", got)
	}
	for _, m := range got {
		if m.SavingsCents != 0 || m.InvestmentsCents != 0 || m.GoalsCents != 0 || m.CreditCents != 0 || m.AssetsCents != m.CashCents {
			t.Errorf("month %s = %+v, want only cash", m.Month, m)
		}
	}
}

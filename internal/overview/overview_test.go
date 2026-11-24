package overview

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/cashflow"
	"github.com/tiagorocha94/household-webapp/internal/credits"
	"github.com/tiagorocha94/household-webapp/internal/goals"
	"github.com/tiagorocha94/household-webapp/internal/investments"
	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/savings"
)

type fakeGoals struct {
	accounts    []goals.Account
	summaries   map[int64][]goals.GoalSummary
	projections map[int64][]goals.ProjectionMonth
	deposited   map[string]money.Cents
}

type fakeCredits struct {
	contracts []credits.ContractSummary
	payments  map[int64][]credits.Payment
}

type fakeInvestments struct {
	entities    []investments.EntitySummary
	contributed map[string]money.Cents
}

type fakeSavings struct {
	accounts  []savings.AccountSummary
	deposited map[string]money.Cents
}

type fakeCashflow struct {
	details   map[string]*cashflow.MonthDetail
	movements map[string][]cashflow.Movement
	byRole    map[string]map[string]money.Cents
	cash      []cashflow.CashBalance
}

func (f fakeCashflow) CashBalances([]string) ([]cashflow.CashBalance, error) { return f.cash, nil }

func (f fakeGoals) ListAccounts() ([]goals.Account, error) { return f.accounts, nil }

func (f fakeGoals) ListGoalSummaries(id int64) ([]goals.GoalSummary, error) {
	return f.summaries[id], nil
}

func (f fakeGoals) DepositsByMonth() (map[string]money.Cents, error) { return f.deposited, nil }

func (f fakeGoals) GetProjection(id int64) ([]goals.ProjectionMonth, error) {
	return f.projections[id], nil
}

func (f fakeCredits) ListContracts() ([]credits.ContractSummary, error) { return f.contracts, nil }

func (f fakeCredits) ListPayments(id int64) ([]credits.Payment, error) { return f.payments[id], nil }

func (f fakeInvestments) ListEntities() ([]investments.EntitySummary, error) {
	return f.entities, nil
}

func (f fakeSavings) ListAccounts() ([]savings.AccountSummary, error) { return f.accounts, nil }

func (f fakeSavings) DepositsByMonth() (map[string]money.Cents, error) { return f.deposited, nil }

func (f fakeInvestments) ContributionsByMonth() (map[string]money.Cents, error) {
	return f.contributed, nil
}

func (f fakeCashflow) ExpensesByRole(role string) (map[string]money.Cents, error) {
	return f.byRole[role], nil
}

func (f fakeCashflow) GetMonth(m string) (*cashflow.MonthDetail, error) {
	d, ok := f.details[m]
	if !ok {
		return nil, fmt.Errorf("get month %s: %w", m, apperr.ErrNotFound)
	}
	return d, nil
}

func (f fakeCashflow) ListMonthMovements(m string) ([]cashflow.Movement, error) {
	return f.movements[m], nil
}

// newService builds a Service whose clock is 9 November 2026.
func newService(g fakeGoals, c fakeCredits, i fakeInvestments, cf fakeCashflow) *Service {
	s := New(g, c, i, fakeSavings{}, cf)
	s.now = func() time.Time { return time.Date(2026, time.November, 9, 12, 0, 0, 0, time.Local) }
	return s
}

func goalSummary(id int64, name, end string, target, saved, allocated int64, total, elapsed int) goals.GoalSummary {
	return goals.GoalSummary{
		GoalID: id, Name: name, Status: goals.StatusActive, EndMonth: end,
		TargetCents: money.Cents(target), GoalBalanceCents: money.Cents(saved),
		TotalAllocatedCents: money.Cents(allocated), TotalMonths: total, ElapsedMonths: elapsed,
		ProgressPct: int(saved * 100 / target),
	}
}

func TestGetWithoutDataIsEmptyButNotNull(t *testing.T) {
	t.Parallel()

	got, err := newService(fakeGoals{}, fakeCredits{}, fakeInvestments{}, fakeCashflow{}).Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Month != "2026-11" || got.Cashflow.Month != "2026-11" {
		t.Errorf("months = %q and %q, want 2026-11", got.Month, got.Cashflow.Month)
	}
	if got.Attention == nil || got.ActiveGoals == nil || got.Contracts == nil || got.Holdings == nil || got.Cashflow.Groups == nil {
		t.Errorf("a list is nil, want empty: %+v", got)
	}
	if got.Cashflow.IncomeCents != 0 || len(got.Attention) != 0 {
		t.Errorf("figures = %+v, want zero", got.Cashflow)
	}
}

func TestGoalsFiguresListAndShortfall(t *testing.T) {
	t.Parallel()

	g := fakeGoals{
		accounts: []goals.Account{{ID: 1, Name: "Household"}},
		summaries: map[int64][]goals.GoalSummary{1: {
			goalSummary(10, "New car", "2027-12", 1500000, 620000, 620000, 24, 10),
			goalSummary(11, "Laptop", "2027-01", 90000, 45000, 45000, 6, 3),
			{GoalID: 12, Name: "Old", Status: "completed", EndMonth: "2026-06", TargetCents: 100},
		}},
		projections: map[int64][]goals.ProjectionMonth{1: {
			{Month: "2026-10", UnallocatedCents: -500},
			{Month: "2026-12", IsProjection: true, UnallocatedCents: -42000},
			{Month: "2027-01", IsProjection: true, UnallocatedCents: -90000},
		}},
	}
	got, err := newService(g, fakeCredits{}, fakeInvestments{}, fakeCashflow{}).Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Goals.SavedCents != 665000 || got.Goals.TargetCents != 1590000 {
		t.Errorf("goals = %+v, want saved 665000 of 1590000 (completed goal ignored)", got.Goals)
	}
	if len(got.ActiveGoals) != 2 || got.ActiveGoals[0].Name != "Laptop" || got.ActiveGoals[1].Name != "New car" {
		t.Fatalf("active goals = %+v, want Laptop then New car (soonest end first)", got.ActiveGoals)
	}
	want := Attention{Kind: AttentionGoalsShortfall, Name: "Household", Month: "2026-12", AmountCents: 42000}
	if len(got.Attention) != 1 || got.Attention[0] != want {
		t.Errorf("attention = %+v, want only %+v (the first future month, not the past one)", got.Attention, want)
	}
}

func TestIsBehindComparesAllocationsWithAnEvenSchedule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		allocated int64
		elapsed   int
		want      bool
	}{
		{"not started", 0, 0, false},
		{"on plan", 25000, 5, false},
		{"ahead", 30000, 5, false},
		{"behind", 24999, 5, true},
		{"after the end", 50000, 12, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := goalSummary(1, "Goal", "2026-12", 50000, 0, tt.allocated, 10, tt.elapsed)
			g.ProgressPct = 0
			if got := isBehind(g); got != tt.want {
				t.Errorf("isBehind(allocated %d, elapsed %d) = %v, want %v", tt.allocated, tt.elapsed, got, tt.want)
			}
		})
	}
}

func TestCreditsUseTheLastRegularPayment(t *testing.T) {
	t.Parallel()

	c := fakeCredits{
		contracts: []credits.ContractSummary{
			{Contract: credits.Contract{ID: 1, Name: "Mortgage", ExpectedEndMonth: "2049-03"}, Status: "active", OutstandingPrincipalCents: 18640000},
			{Contract: credits.Contract{ID: 2, Name: "Old loan"}, Status: "paid_off", OutstandingPrincipalCents: 0},
			{Contract: credits.Contract{ID: 3, Name: "New loan"}, Status: "active", OutstandingPrincipalCents: 500000},
		},
		payments: map[int64][]credits.Payment{
			1: {
				{ID: 1, Month: "2026-09", Type: credits.PaymentRegular, PrincipalCents: 40000, InterestCents: 34000},
				{ID: 2, Month: "2026-10", Type: credits.PaymentRegular, PrincipalCents: 40500, InterestCents: 33700},
				{ID: 3, Month: "2026-11", Type: credits.PaymentExtraordinary, PrincipalCents: 100000},
			},
		},
	}
	got, err := newService(fakeGoals{}, c, fakeInvestments{}, fakeCashflow{}).Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Credit.ActiveContracts != 2 || got.Credit.OutstandingCents != 19140000 {
		t.Errorf("credit = %+v, want 2 active contracts and 19140000 outstanding", got.Credit)
	}
	if got.Credit.MonthlyInstalmentCents != 74200 {
		t.Errorf("monthly instalments = %d, want 74200 (the extraordinary payment is not an instalment)", got.Credit.MonthlyInstalmentCents)
	}
	if len(got.Contracts) != 2 {
		t.Fatalf("contracts = %+v, want the two active ones", got.Contracts)
	}
	mortgage := got.Contracts[0]
	if mortgage.NextInstalmentMonth != "2026-11" || mortgage.InstalmentCents != 74200 || mortgage.EndMonth != "2049-03" {
		t.Errorf("mortgage = %+v, want next instalment 2026-11 of 74200 ending 2049-03", mortgage)
	}
	if fresh := got.Contracts[1]; fresh.NextInstalmentMonth != "" || fresh.InstalmentCents != 0 {
		t.Errorf("contract without payments = %+v, want no instalment", fresh)
	}
}

func TestInvestmentsAddUpTheEntities(t *testing.T) {
	t.Parallel()

	i := fakeInvestments{entities: []investments.EntitySummary{
		{Entity: investments.Entity{ID: 1, Name: "Broker"}, CurrentValueCents: 4120000},
		{Entity: investments.Entity{ID: 2, Name: "Pension"}, CurrentValueCents: 1770000},
	}}
	got, err := newService(fakeGoals{}, fakeCredits{}, i, fakeCashflow{}).Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Investments.CurrentValueCents != 5890000 || len(got.Holdings) != 2 || got.Holdings[1].Name != "Pension" {
		t.Errorf("investments = %+v, holdings = %+v", got.Investments, got.Holdings)
	}
}

func TestCashflowFiguresBudgetsAndPreviousMonthAtTheSamePoint(t *testing.T) {
	t.Parallel()

	cf := fakeCashflow{
		details: map[string]*cashflow.MonthDetail{"2026-11": {
			MonthSummary: cashflow.MonthSummary{Month: "2026-11", IncomeCents: 385000, ExpenseCents: 231000, NetCents: 154000, SavingsRatePct: 4000},
			SavedCents:   25000,
			ByGroup: []cashflow.GroupTotal{
				{Name: "Essential", BudgetCents: 170000, SpentCents: 148000},
				{Name: "Fun", BudgetCents: 18000, SpentCents: 20500},
				{Name: "Empty", BudgetCents: 0, SpentCents: 0},
			},
		}},
		movements: map[string][]cashflow.Movement{"2026-10": {
			{Date: "2026-10-03", Direction: cashflow.DirectionExpense, AmountCents: 10000},
			{Date: "2026-10-09", Direction: cashflow.DirectionExpense, AmountCents: 5000},
			{Date: "2026-10-10", Direction: cashflow.DirectionExpense, AmountCents: 70000},
			{Date: "2026-10-05", Direction: cashflow.DirectionIncome, AmountCents: 300000},
		}},
	}
	got, err := newService(fakeGoals{}, fakeCredits{}, fakeInvestments{}, cf).Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	c := got.Cashflow
	if c.IncomeCents != 385000 || c.ExpenseCents != 231000 || c.NetCents != 154000 || c.SavingsRatePct != 4000 || c.SavedCents != 25000 {
		t.Errorf("cashflow = %+v", c)
	}
	if c.PreviousExpenseCents != 15000 {
		t.Errorf("previous expenses = %d, want 15000 (up to 9 October, expenses only)", c.PreviousExpenseCents)
	}
	if len(c.Groups) != 3 || c.Groups[0].UsedPct != 87 || c.Groups[1].UsedPct != 113 || c.Groups[2].UsedPct != 0 {
		t.Errorf("groups = %+v, want used 87, 113 and 0", c.Groups)
	}
	want := Attention{Kind: AttentionBudgetOver, Name: "Fun", AmountCents: 2500, Pct: 113}
	if len(got.Attention) != 1 || got.Attention[0] != want {
		t.Errorf("attention = %+v, want only %+v", got.Attention, want)
	}
}

func TestSavingsAddUpTheAccountsAndOnlyTheTargetsThatExist(t *testing.T) {
	t.Parallel()

	s := newService(fakeGoals{}, fakeCredits{}, fakeInvestments{}, fakeCashflow{})
	s.savings = fakeSavings{accounts: []savings.AccountSummary{
		{
			Account:       savings.Account{ID: 1, Name: "Emergency", Username: "alex", TargetCents: 1000000},
			BalanceCents:  740000,
			InterestCents: 40000,
			ProgressPct:   74,
		},
		{Account: savings.Account{ID: 2, Name: "Holiday pot"}, BalanceCents: 31500},
	}}
	got, err := s.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	want := SavingsFigures{
		BalanceCents: 771500, TargetCents: 1000000, Accounts: 2,
		InterestCents: 40000,
		Items: []SavingsRow{
			{ID: 1, Name: "Emergency", Owner: "alex", BalanceCents: 740000, TargetCents: 1000000, ProgressPct: 74, InterestCents: 40000},
			{ID: 2, Name: "Holiday pot", BalanceCents: 31500},
		},
	}
	if !reflect.DeepEqual(got.Savings, want) {
		t.Errorf("savings = %+v, want %+v", got.Savings, want)
	}
}

func TestUnassignedMoneyIsWarnedAbout(t *testing.T) {
	t.Parallel()

	// "Today" is 2026-11-09, so the months checked are 2026-09, 2026-10 and 2026-11.
	s := New(fakeGoals{deposited: map[string]money.Cents{"2026-11": 25000}}, fakeCredits{},
		fakeInvestments{contributed: map[string]money.Cents{"2026-11": 30000, "2026-10": 10000}},
		fakeSavings{deposited: map[string]money.Cents{"2026-11": 5000, "2026-10": 20000}},
		fakeCashflow{
			byRole: map[string]map[string]money.Cents{
				cashflow.RoleSavings:     {"2026-11": 15000, "2026-10": 20000, "2026-08": 99999},
				cashflow.RoleInvestments: {"2026-11": 30000, "2026-10": 25000},
				cashflow.RoleGoals:       {"2026-11": 40000},
			},
		})
	s.now = func() time.Time { return time.Date(2026, time.November, 9, 12, 0, 0, 0, time.Local) }

	got, err := s.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	want := []Attention{
		// Savings: November has 100.00 more than was deposited. October matches and
		// August is outside the three months.
		{Kind: AttentionUnassignedSavings, Month: "2026-11", AmountCents: 10000},
		// Investments: October has 150.00 more than was contributed. November matches.
		{Kind: AttentionUnassignedInvestments, Month: "2026-10", AmountCents: 15000},
		// Goals: November has 150.00 more than was deposited.
		{Kind: AttentionUnassignedGoals, Month: "2026-11", AmountCents: 15000},
	}
	if !reflect.DeepEqual(got.Attention, want) {
		t.Errorf("Attention = %+v, want %+v", got.Attention, want)
	}
}

func TestMoreRecordedThanSpentIsNotWarnedAbout(t *testing.T) {
	t.Parallel()

	s := New(fakeGoals{}, fakeCredits{}, fakeInvestments{},
		fakeSavings{deposited: map[string]money.Cents{"2026-11": 50000}},
		fakeCashflow{byRole: map[string]map[string]money.Cents{cashflow.RoleSavings: {"2026-11": 10000}}})
	s.now = func() time.Time { return time.Date(2026, time.November, 9, 12, 0, 0, 0, time.Local) }

	got, err := s.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(got.Attention) != 0 {
		t.Errorf("Attention = %+v, want none", got.Attention)
	}
}

func TestCreditWithoutPaymentAfterExpectedDayIsWarnedAbout(t *testing.T) {
	t.Parallel()

	contract := func(id int64, name string, day int) credits.ContractSummary {
		return credits.ContractSummary{
			Contract: credits.Contract{ID: id, Name: name, StartMonth: "2026-01", ExpectedDay: day},
			Status:   statusActive,
		}
	}
	// "Today" is 2026-11-09.
	s := New(fakeGoals{}, fakeCredits{
		contracts: []credits.ContractSummary{
			contract(1, "Car", 5),       // day passed, nothing recorded: warned
			contract(2, "Home", 5),      // day passed, paid: fine
			contract(3, "Personal", 15), // day not reached yet: fine
			contract(4, "Other", 0),     // no expected day: never chased
		},
		payments: map[int64][]credits.Payment{
			2: {{ID: 1, ContractID: 2, Month: "2026-11", Type: credits.PaymentRegular}},
		},
	}, fakeInvestments{}, fakeSavings{}, fakeCashflow{})
	s.now = func() time.Time { return time.Date(2026, time.November, 9, 12, 0, 0, 0, time.Local) }

	got, err := s.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	want := []Attention{{Kind: AttentionCreditUnpaid, Name: "Car", Month: "2026-11"}}
	if !reflect.DeepEqual(got.Attention, want) {
		t.Errorf("Attention = %+v, want %+v", got.Attention, want)
	}
}

func TestCreditsExpensesWithoutAPaymentAreWarnedAboutButAmountsAreNotCompared(t *testing.T) {
	t.Parallel()

	s := New(fakeGoals{}, fakeCredits{
		contracts: []credits.ContractSummary{{
			Contract: credits.Contract{ID: 1, Name: "Home", StartMonth: "2026-01"},
			Status:   statusActive,
		}},
		payments: map[int64][]credits.Payment{
			// October was paid: 632.00 against 750.00 in cashflow is fine.
			1: {{ID: 1, ContractID: 1, Month: "2026-10", Type: credits.PaymentRegular, PrincipalCents: 25700, InterestCents: 37500}},
		},
	}, fakeInvestments{}, fakeSavings{}, fakeCashflow{
		byRole: map[string]map[string]money.Cents{
			cashflow.RoleCredits: {"2026-11": 75000, "2026-10": 75000},
		},
	})
	s.now = func() time.Time { return time.Date(2026, time.November, 9, 12, 0, 0, 0, time.Local) }

	got, err := s.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	want := []Attention{{Kind: AttentionUnassignedCredits, Month: "2026-11", AmountCents: 75000}}
	if !reflect.DeepEqual(got.Attention, want) {
		t.Errorf("Attention = %+v, want %+v", got.Attention, want)
	}
}

func TestNetWorthIsAssetsMinusCreditWithGoalsCountedOnce(t *testing.T) {
	t.Parallel()

	g := fakeGoals{
		accounts:  []goals.Account{{ID: 1, Name: "Household"}},
		summaries: map[int64][]goals.GoalSummary{1: {goalSummary(10, "New car", "2027-12", 1500000, 620000, 620000, 24, 10)}},
	}
	c := fakeCredits{contracts: []credits.ContractSummary{
		{Contract: credits.Contract{ID: 1, Name: "Mortgage"}, Status: "active", OutstandingPrincipalCents: 18640000},
		{Contract: credits.Contract{ID: 2, Name: "Old loan"}, Status: "paid_off"},
	}}
	i := fakeInvestments{entities: []investments.EntitySummary{
		{Entity: investments.Entity{ID: 1, Name: "Broker"}, CurrentValueCents: 4120000},
	}}
	s := newService(g, c, i, fakeCashflow{})
	s.savings = fakeSavings{accounts: []savings.AccountSummary{
		{Account: savings.Account{ID: 1, Name: "Emergency"}, BalanceCents: 740000},
	}}
	got, err := s.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	want := NetWorthFigures{AssetsCents: 740000 + 4120000 + 620000, CreditCents: 18640000, NetCents: 5480000 - 18640000}
	if !reflect.DeepEqual(got.NetWorth, want) {
		t.Errorf("net worth = %+v, want %+v", got.NetWorth, want)
	}
}

func TestNetWorthAddsEachPersonsCash(t *testing.T) {
	t.Parallel()

	cf := fakeCashflow{cash: []cashflow.CashBalance{
		{User: "alice", Month: "2026-11", BalanceCents: 300000},
		{User: "bob", Month: "2026-11", BalanceCents: -20000},
	}}
	s := newService(fakeGoals{}, fakeCredits{contracts: []credits.ContractSummary{
		{Contract: credits.Contract{ID: 1, Name: "Car"}, Status: "active", OutstandingPrincipalCents: 100000},
	}}, fakeInvestments{}, cf)
	got, err := s.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	want := NetWorthFigures{
		AssetsCents: 280000,
		CashCents:   280000,
		CashByUser:  []UserCash{{User: "alice", BalanceCents: 300000}, {User: "bob", BalanceCents: -20000}},
		CreditCents: 100000,
		NetCents:    180000,
	}
	if !reflect.DeepEqual(got.NetWorth, want) {
		t.Errorf("net worth = %+v, want %+v", got.NetWorth, want)
	}
}

func TestGetWithOnlyCashflowLeavesTheOtherModulesOut(t *testing.T) {
	t.Parallel()

	cf := fakeCashflow{
		details: map[string]*cashflow.MonthDetail{"2026-11": {
			MonthSummary: cashflow.MonthSummary{Month: "2026-11", IncomeCents: 100000, ExpenseCents: 40000, NetCents: 60000},
		}},
		// Money put aside in a savings category: nothing to compare it with
		// when the savings module does not run, so there is no warning.
		byRole: map[string]map[string]money.Cents{cashflow.RoleSavings: {"2026-11": 25000}},
		cash:   []cashflow.CashBalance{{User: "alice", Month: "2026-11", BalanceCents: 7000}},
	}
	s := New(nil, nil, nil, nil, cf)
	s.now = func() time.Time { return time.Date(2026, time.November, 9, 12, 0, 0, 0, time.Local) }
	got, err := s.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Goals != nil || got.Credit != nil || got.Investments != nil || got.Savings != nil {
		t.Errorf("sections of modules that do not run are present: %+v", got)
	}
	if got.Cashflow == nil || got.Cashflow.IncomeCents != 100000 {
		t.Errorf("cashflow = %+v, want the month's figures", got.Cashflow)
	}
	if len(got.Attention) != 0 {
		t.Errorf("attention = %+v, want none", got.Attention)
	}
	if got.NetWorth.CashCents != 7000 || got.NetWorth.NetCents != 7000 {
		t.Errorf("net worth = %+v, want just the cash", got.NetWorth)
	}
}

func TestGetWithoutCashflowStillShowsTheOtherModules(t *testing.T) {
	t.Parallel()

	g := fakeGoals{
		accounts:  []goals.Account{{ID: 1, Name: "Joint"}},
		summaries: map[int64][]goals.GoalSummary{1: {goalSummary(1, "Holiday", "2027-06", 100000, 40000, 40000, 12, 6)}},
	}
	s := New(g, nil, nil, nil, nil)
	got, err := s.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Cashflow != nil || got.Credit != nil || got.Savings != nil || got.Investments != nil {
		t.Errorf("sections of modules that do not run are present: %+v", got)
	}
	if got.Goals == nil || got.Goals.SavedCents != 40000 || len(got.ActiveGoals) != 1 {
		t.Errorf("goals = %+v, active = %+v", got.Goals, got.ActiveGoals)
	}
	if got.NetWorth.AssetsCents != 40000 || got.NetWorth.CashByUser != nil {
		t.Errorf("net worth = %+v, want the goal money and no cash", got.NetWorth)
	}
}

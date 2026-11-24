// Package overview composes the figures of the Home page from the other
// modules. It owns no tables and stores nothing: every figure is read from the
// goals, credits, investments, savings and cashflow services on each request.
// It is the one place that knows all of them, like the phone app, and each
// module is optional: a module that does not run is passed as nil, and its
// figures, rows and warnings are left out.
package overview

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/cashflow"
	"github.com/tiagorocha94/household-webapp/internal/credits"
	"github.com/tiagorocha94/household-webapp/internal/goals"
	"github.com/tiagorocha94/household-webapp/internal/investments"
	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/month"
	"github.com/tiagorocha94/household-webapp/internal/savings"
)

// statusActive is the status of a credit contract that still has principal
// outstanding.
const statusActive = "active"

// unassignedMonths is how many months, counting the current one, are checked for
// money put aside in cashflow that was not typed into savings, investments or goals.
const unassignedMonths = 3

// Kinds of Attention.
const (
	// AttentionGoalsShortfall means the planned allocations of a future month
	// exceed the money the goals account will have by then.
	AttentionGoalsShortfall = "goals_shortfall"
	// AttentionBudgetOver means a budget group spent more than its share of
	// this month's income.
	AttentionBudgetOver = "budget_over"
	// AttentionCreditUnpaid means an active credit with an expected day has no
	// regular payment recorded for this month and that day has passed.
	AttentionCreditUnpaid = "credit_unpaid"
	// AttentionUnassignedSavings means a month's expenses in savings categories
	// are more than the deposits typed in the savings module.
	AttentionUnassignedSavings = "unassigned_savings"
	// AttentionUnassignedInvestments means a month's expenses in investment
	// categories are more than the contributions typed in the investments module.
	AttentionUnassignedInvestments = "unassigned_investments"
	// AttentionUnassignedGoals means a month's expenses in goals categories are
	// more than the deposits typed in the goals module.
	AttentionUnassignedGoals = "unassigned_goals"
	// AttentionUnassignedCredits means a month has expenses in credits
	// categories but no credit payment recorded. Amounts are not compared: a
	// cashflow instalment often includes insurance and fees that a payment does
	// not.
	AttentionUnassignedCredits = "unassigned_credits"
)

// Goals is what the overview reads from the goals module.
type Goals interface {
	ListAccounts() ([]goals.Account, error)
	ListGoalSummaries(accountID int64) ([]goals.GoalSummary, error)
	GetProjection(accountID int64) ([]goals.ProjectionMonth, error)
	// DepositsByMonth returns the deposits of every goal account by month.
	DepositsByMonth() (map[string]money.Cents, error)
}

// Credits is what the overview reads from the credits module.
type Credits interface {
	ListContracts() ([]credits.ContractSummary, error)
	ListPayments(contractID int64) ([]credits.Payment, error)
}

// Investments is what the overview reads from the investments module.
type Investments interface {
	ListEntities() ([]investments.EntitySummary, error)
	// ContributionsByMonth returns the contributions of every instrument by month.
	ContributionsByMonth() (map[string]money.Cents, error)
}

// Savings is what the overview reads from the savings module.
type Savings interface {
	ListAccounts() ([]savings.AccountSummary, error)
	// DepositsByMonth returns the deposits of every account by month.
	DepositsByMonth() (map[string]money.Cents, error)
}

// Cashflow is what the overview reads from the cashflow module.
type Cashflow interface {
	GetMonth(m string) (*cashflow.MonthDetail, error)
	ListMonthMovements(m string) ([]cashflow.Movement, error)
	// ExpensesByRole returns the expenses of the categories with a role by month.
	ExpensesByRole(role string) (map[string]money.Cents, error)
	// CashBalances returns each person's cash at the end of the given months.
	CashBalances(months []string) ([]cashflow.CashBalance, error)
}

// Overview is everything the Home page shows.
type Overview struct {
	// Month is the current month as YYYY-MM.
	Month     string
	Attention []Attention
	// Goals, Credit, Investments, Savings and Cashflow are nil when their
	// module does not run.
	Goals       *GoalsFigures
	Credit      *CreditFigures
	Investments *InvestmentFigures
	Savings     *SavingsFigures
	NetWorth    NetWorthFigures
	Cashflow    *CashflowFigures
	// ActiveGoals lists every active goal, the soonest end first.
	ActiveGoals []GoalRow
	// Contracts lists every active credit contract.
	Contracts []ContractRow
	// Holdings lists the value held at each investment entity.
	Holdings []HoldingRow
}

// Attention is one thing that needs looking at. Month is empty and Pct is zero
// when they do not apply to the Kind.
type Attention struct {
	Kind string
	// Name is the goal account, the budget group or the credit; empty for
	// unassigned money.
	Name  string
	Month string
	// AmountCents is the shortfall, how much the group is over its budget, or the
	// money put aside in cashflow that no savings deposit or investment
	// contribution accounts for.
	AmountCents money.Cents
	// Pct is the share of the budget spent, as a whole percent.
	Pct int
}

// GoalsFigures totals the active goals.
type GoalsFigures struct {
	SavedCents  money.Cents
	TargetCents money.Cents
}

// CreditFigures totals the active credit contracts.
type CreditFigures struct {
	OutstandingCents money.Cents
	// MonthlyInstalmentCents adds up the last regular payment of each contract.
	MonthlyInstalmentCents money.Cents
	ActiveContracts        int
}

// InvestmentFigures totals the investments.
type InvestmentFigures struct {
	CurrentValueCents money.Cents
}

// NetWorthFigures is what the household has minus what it owes, from the
// figures the modules already hold. The money in the goals account is counted
// once, as saved in the active goals.
type NetWorthFigures struct {
	// AssetsCents adds the savings balance, the investments' current value, the
	// money saved in the active goals and the cash of everyone who has a typed
	// bank balance.
	AssetsCents money.Cents
	// CashCents is the part of AssetsCents held as cash, the sum of CashByUser.
	CashCents money.Cents
	// CashByUser is each person's cash this month, in the order of their
	// names. It is empty when nobody has a typed bank balance.
	CashByUser []UserCash
	// CreditCents is the outstanding principal of the active credit contracts.
	CreditCents money.Cents
	// NetCents is AssetsCents minus CreditCents and can be negative.
	NetCents money.Cents
}

// UserCash is one person's cash.
type UserCash struct {
	User         string
	BalanceCents money.Cents
}

// SavingsFigures totals the savings accounts.
type SavingsFigures struct {
	// BalanceCents is the balance of every account.
	BalanceCents money.Cents
	// TargetCents adds up the targets of the accounts that have one; 0 when no
	// account has a target.
	TargetCents money.Cents
	// Accounts is the number of savings accounts.
	Accounts int
	// InterestCents is the interest earned across the accounts.
	InterestCents money.Cents
	// Items lists the accounts in the order the savings module returns them.
	Items []SavingsRow
}

// SavingsRow is one savings account on the overview.
type SavingsRow struct {
	ID    int64
	Name  string
	Owner string
	// BalanceCents is the account's balance.
	BalanceCents money.Cents
	// TargetCents is 0 when the account has no target.
	TargetCents   money.Cents
	ProgressPct   int
	InterestCents money.Cents
}

// CashflowFigures summarises the current month.
type CashflowFigures struct {
	Month        string
	IncomeCents  money.Cents
	ExpenseCents money.Cents
	// SavedCents is the part of the expenses in savings categories.
	SavedCents     money.Cents
	NetCents       money.Cents
	SavingsRatePct int
	// PreviousExpenseCents is what was spent in the previous month up to the
	// same day of the month, so the two can be compared fairly part way
	// through the month.
	PreviousExpenseCents money.Cents
	Groups               []GroupRow
}

// GroupRow is one budget group's figures for the current month.
type GroupRow struct {
	Name        string
	BudgetCents money.Cents
	SpentCents  money.Cents
	// UsedPct is the share of the budget spent, as a whole percent. It is 0
	// when the budget is 0 and can be above 100.
	UsedPct int
}

// GoalRow is one active goal.
type GoalRow struct {
	ID          int64
	Name        string
	EndMonth    string
	TargetCents money.Cents
	SavedCents  money.Cents
	ProgressPct int
	// Behind is true when less was allocated than an even schedule of the
	// target would have by the current month.
	Behind bool
}

// ContractRow is one active credit contract.
type ContractRow struct {
	ID               int64
	Name             string
	OutstandingCents money.Cents
	// InstalmentCents is the last regular payment; 0 when there is none yet.
	InstalmentCents money.Cents
	// NextInstalmentMonth is the month after the last regular payment, an
	// estimate because payments are recorded by month. Empty when there is no
	// regular payment yet.
	NextInstalmentMonth string
	EndMonth            string
}

// HoldingRow is the current value held at one investment entity.
type HoldingRow struct {
	ID         int64
	Name       string
	ValueCents money.Cents
}

// Service builds the overview.
type Service struct {
	goals       Goals
	credits     Credits
	investments Investments
	savings     Savings
	cashflow    Cashflow
	now         func() time.Time
}

// New creates a Service reading from the given module services. A module that
// does not run is passed as an untyped nil, never as a nil pointer in the
// interface.
func New(g Goals, c Credits, i Investments, sv Savings, cf Cashflow) *Service {
	return &Service{goals: g, credits: c, investments: i, savings: sv, cashflow: cf, now: time.Now}
}

// Get builds the overview as of now, in the local time zone.
func (s *Service) Get() (*Overview, error) {
	now := s.now()
	out := &Overview{
		Month:       month.Current(now),
		Attention:   []Attention{},
		ActiveGoals: []GoalRow{},
		Contracts:   []ContractRow{},
		Holdings:    []HoldingRow{},
	}
	if err := s.addGoals(out); err != nil {
		return nil, err
	}
	if err := s.addCredits(out, now); err != nil {
		return nil, err
	}
	if err := s.addInvestments(out); err != nil {
		return nil, err
	}
	if err := s.addSavings(out); err != nil {
		return nil, err
	}
	if err := s.addNetWorth(out); err != nil {
		return nil, err
	}
	if err := s.addCashflow(out, now); err != nil {
		return nil, err
	}
	if err := s.addUnassigned(out); err != nil {
		return nil, err
	}
	return out, nil
}

// creditPaymentsByMonth returns what the payments of every credit add up to in
// each month, keyed by YYYY-MM.
func (s *Service) creditPaymentsByMonth() (map[string]money.Cents, error) {
	contracts, err := s.credits.ListContracts()
	if err != nil {
		return nil, fmt.Errorf("list credits: %w", err)
	}
	totals := make(map[string]money.Cents)
	for _, c := range contracts {
		payments, err := s.credits.ListPayments(c.ID)
		if err != nil {
			return nil, fmt.Errorf("list payments of %s: %w", c.Name, err)
		}
		for _, p := range payments {
			totals[p.Month] += p.PrincipalCents + p.InterestCents
		}
	}
	return totals, nil
}

// addUnassigned warns about the money that cashflow shows as put into savings,
// investments or goals but that no deposit or contribution accounts for, and
// about months with credits expenses and no credit payment. A role nobody gave
// to a category has no expenses, so it is never warned about. A month where the
// typed amounts are equal or higher is fine.
func (s *Service) addUnassigned(out *Overview) error {
	if s.cashflow == nil {
		return nil
	}
	type check struct {
		kind     string
		role     string
		recorded func() (map[string]money.Cents, error)
		// presence compares only whether anything was recorded, not how much.
		presence bool
	}
	var checks []check
	if s.savings != nil {
		checks = append(checks, check{AttentionUnassignedSavings, cashflow.RoleSavings, s.savings.DepositsByMonth, false})
	}
	if s.investments != nil {
		checks = append(checks, check{AttentionUnassignedInvestments, cashflow.RoleInvestments, s.investments.ContributionsByMonth, false})
	}
	if s.goals != nil {
		checks = append(checks, check{AttentionUnassignedGoals, cashflow.RoleGoals, s.goals.DepositsByMonth, false})
	}
	if s.credits != nil {
		checks = append(checks, check{AttentionUnassignedCredits, cashflow.RoleCredits, s.creditPaymentsByMonth, true})
	}
	for _, c := range checks {
		expenses, err := s.cashflow.ExpensesByRole(c.role)
		if err != nil {
			return fmt.Errorf("read %s expenses by month: %w", c.role, err)
		}
		recorded, err := c.recorded()
		if err != nil {
			return fmt.Errorf("read %s deposits by month: %w", c.role, err)
		}
		for back := unassignedMonths - 1; back >= 0; back-- {
			m, err := month.Add(out.Month, -back)
			if err != nil {
				return fmt.Errorf("find the month %d back from %s: %w", back, out.Month, err)
			}
			gap := expenses[m] - recorded[m]
			if c.presence && recorded[m] > 0 {
				gap = 0
			}
			if gap > 0 {
				out.Attention = append(out.Attention, Attention{Kind: c.kind, Month: m, AmountCents: gap})
			}
		}
	}
	return nil
}

// addNetWorth totals what the household has and owes. It runs after the module
// figures are filled in.
func (s *Service) addNetWorth(out *Overview) error {
	if s.cashflow != nil {
		balances, err := s.cashflow.CashBalances([]string{out.Month})
		if err != nil {
			return fmt.Errorf("cash balances: %w", err)
		}
		for _, b := range balances {
			out.NetWorth.CashByUser = append(out.NetWorth.CashByUser, UserCash{User: b.User, BalanceCents: b.BalanceCents})
			out.NetWorth.CashCents += b.BalanceCents
		}
	}
	assets := out.NetWorth.CashCents
	if out.Savings != nil {
		assets += out.Savings.BalanceCents
	}
	if out.Investments != nil {
		assets += out.Investments.CurrentValueCents
	}
	if out.Goals != nil {
		assets += out.Goals.SavedCents
	}
	out.NetWorth.AssetsCents = assets
	if out.Credit != nil {
		out.NetWorth.CreditCents = out.Credit.OutstandingCents
	}
	out.NetWorth.NetCents = out.NetWorth.AssetsCents - out.NetWorth.CreditCents
	return nil
}

func (s *Service) addGoals(out *Overview) error {
	if s.goals == nil {
		return nil
	}
	out.Goals = &GoalsFigures{}
	accounts, err := s.goals.ListAccounts()
	if err != nil {
		return fmt.Errorf("list goal accounts: %w", err)
	}
	for _, account := range accounts {
		summaries, err := s.goals.ListGoalSummaries(account.ID)
		if err != nil {
			return fmt.Errorf("list goals of account %d: %w", account.ID, err)
		}
		for _, g := range summaries {
			if g.Status != goals.StatusActive {
				continue
			}
			out.Goals.SavedCents += g.GoalBalanceCents
			out.Goals.TargetCents += g.TargetCents
			out.ActiveGoals = append(out.ActiveGoals, GoalRow{
				ID: g.GoalID, Name: g.Name, EndMonth: g.EndMonth,
				TargetCents: g.TargetCents, SavedCents: g.GoalBalanceCents,
				ProgressPct: g.ProgressPct, Behind: isBehind(g),
			})
		}

		projection, err := s.goals.GetProjection(account.ID)
		if err != nil {
			return fmt.Errorf("get projection of account %d: %w", account.ID, err)
		}
		for _, pm := range projection {
			if pm.IsProjection && pm.UnallocatedCents < 0 {
				out.Attention = append(out.Attention, Attention{
					Kind: AttentionGoalsShortfall, Name: account.Name,
					Month: pm.Month, AmountCents: -pm.UnallocatedCents,
				})
				break
			}
		}
	}
	sort.SliceStable(out.ActiveGoals, func(i, j int) bool {
		a, b := out.ActiveGoals[i], out.ActiveGoals[j]
		if a.EndMonth != b.EndMonth {
			return a.EndMonth < b.EndMonth
		}
		return a.Name < b.Name
	})
	return nil
}

// isBehind compares what was allocated to a goal with an even split of its
// target over its months, up to the current one. The phone uses the same rule.
func isBehind(g goals.GoalSummary) bool {
	months := int64(max(g.TotalMonths, 1))
	elapsed := int64(min(max(g.ElapsedMonths, 0), int(months)))
	expected := (int64(g.TargetCents)*elapsed + months/2) / months
	return int64(g.TotalAllocatedCents) < expected
}

func (s *Service) addCredits(out *Overview, now time.Time) error {
	if s.credits == nil {
		return nil
	}
	out.Credit = &CreditFigures{}
	contracts, err := s.credits.ListContracts()
	if err != nil {
		return fmt.Errorf("list credit contracts: %w", err)
	}
	for _, c := range contracts {
		if c.Status != statusActive {
			continue
		}
		out.Credit.ActiveContracts++
		out.Credit.OutstandingCents += c.OutstandingPrincipalCents
		row := ContractRow{
			ID: c.ID, Name: c.Name,
			OutstandingCents: c.OutstandingPrincipalCents, EndMonth: c.ExpectedEndMonth,
		}

		payments, err := s.credits.ListPayments(c.ID)
		if err != nil {
			return fmt.Errorf("list payments of contract %d: %w", c.ID, err)
		}
		if last, ok := lastRegular(payments); ok {
			next, err := month.Add(last.Month, 1)
			if err != nil {
				return fmt.Errorf("find the month after payment %d: %w", last.ID, err)
			}
			row.InstalmentCents = last.TotalCents()
			row.NextInstalmentMonth = next
			out.Credit.MonthlyInstalmentCents += row.InstalmentCents
		}
		if c.ExpectedDay > 0 && now.Day() > c.ExpectedDay && out.Month >= c.StartMonth && !hasRegularIn(payments, out.Month) {
			out.Attention = append(out.Attention, Attention{Kind: AttentionCreditUnpaid, Name: c.Name, Month: out.Month})
		}
		out.Contracts = append(out.Contracts, row)
	}
	return nil
}

// hasRegularIn reports whether a regular payment is recorded for the month.
func hasRegularIn(payments []credits.Payment, m string) bool {
	for _, p := range payments {
		if p.Type == credits.PaymentRegular && p.Month == m {
			return true
		}
	}
	return false
}

// lastRegular returns the regular payment of the latest month.
func lastRegular(payments []credits.Payment) (credits.Payment, bool) {
	var last credits.Payment
	found := false
	for _, p := range payments {
		if p.Type != credits.PaymentRegular {
			continue
		}
		if !found || p.Month > last.Month {
			last, found = p, true
		}
	}
	return last, found
}

func (s *Service) addInvestments(out *Overview) error {
	if s.investments == nil {
		return nil
	}
	out.Investments = &InvestmentFigures{}
	entities, err := s.investments.ListEntities()
	if err != nil {
		return fmt.Errorf("list investment entities: %w", err)
	}
	for _, e := range entities {
		out.Investments.CurrentValueCents += e.CurrentValueCents
		out.Holdings = append(out.Holdings, HoldingRow{ID: e.ID, Name: e.Name, ValueCents: e.CurrentValueCents})
	}
	return nil
}

func (s *Service) addSavings(out *Overview) error {
	if s.savings == nil {
		return nil
	}
	out.Savings = &SavingsFigures{}
	accounts, err := s.savings.ListAccounts()
	if err != nil {
		return fmt.Errorf("list savings accounts: %w", err)
	}
	for _, a := range accounts {
		out.Savings.Accounts++
		out.Savings.BalanceCents += a.BalanceCents
		if a.TargetCents > 0 {
			out.Savings.TargetCents += a.TargetCents
		}
		out.Savings.InterestCents += a.InterestCents
		target := a.TargetCents
		if target < 0 {
			target = 0
		}
		out.Savings.Items = append(out.Savings.Items, SavingsRow{
			ID: a.ID, Name: a.Name, Owner: a.Username,
			BalanceCents: a.BalanceCents, TargetCents: target,
			ProgressPct: a.ProgressPct, InterestCents: a.InterestCents,
		})
	}
	return nil
}

func (s *Service) addCashflow(out *Overview, now time.Time) error {
	if s.cashflow == nil {
		return nil
	}
	out.Cashflow = &CashflowFigures{Month: out.Month, Groups: []GroupRow{}}

	detail, err := s.cashflow.GetMonth(out.Month)
	switch {
	case errors.Is(err, apperr.ErrNotFound):
		// No movements yet this month: the figures stay at zero.
	case err != nil:
		return fmt.Errorf("get cashflow month %s: %w", out.Month, err)
	default:
		out.Cashflow.IncomeCents = detail.IncomeCents
		out.Cashflow.ExpenseCents = detail.ExpenseCents
		out.Cashflow.SavedCents = detail.SavedCents
		out.Cashflow.NetCents = detail.NetCents
		out.Cashflow.SavingsRatePct = detail.SavingsRatePct
		for _, g := range detail.ByGroup {
			row := GroupRow{Name: g.Name, BudgetCents: g.BudgetCents, SpentCents: g.SpentCents}
			if g.BudgetCents > 0 {
				row.UsedPct = int(g.SpentCents * 100 / g.BudgetCents)
			}
			out.Cashflow.Groups = append(out.Cashflow.Groups, row)
			if g.BudgetCents > 0 && g.SpentCents > g.BudgetCents {
				out.Attention = append(out.Attention, Attention{
					Kind: AttentionBudgetOver, Name: g.Name,
					AmountCents: g.SpentCents - g.BudgetCents, Pct: row.UsedPct,
				})
			}
		}
	}

	previous, err := month.Add(out.Month, -1)
	if err != nil {
		return fmt.Errorf("find the previous month: %w", err)
	}
	movements, err := s.cashflow.ListMonthMovements(previous)
	if err != nil {
		return fmt.Errorf("list movements of %s: %w", previous, err)
	}
	// Dates are YYYY-MM-DD, so they compare as text.
	cutoff := fmt.Sprintf("%s-%02d", previous, now.Day())
	for _, m := range movements {
		if m.Direction == cashflow.DirectionExpense && m.Date <= cutoff {
			out.Cashflow.PreviousExpenseCents += m.AmountCents
		}
	}
	return nil
}

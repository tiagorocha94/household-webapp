package overview

import (
	"fmt"
	"sort"

	"github.com/tiagorocha94/household-webapp/internal/cashflow"
	"github.com/tiagorocha94/household-webapp/internal/credits"
	"github.com/tiagorocha94/household-webapp/internal/goals"
	"github.com/tiagorocha94/household-webapp/internal/investments"
	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/savings"
)

// HistoryGoals is what the net worth history reads from the goals module.
type HistoryGoals interface {
	ListAccounts() ([]goals.Account, error)
	ListGoalSummaries(accountID int64) ([]goals.GoalSummary, error)
	ListMovements(accountID int64) ([]goals.Movement, error)
}

// HistoryCredits is what the net worth history reads from the credits module.
type HistoryCredits interface {
	ListContracts() ([]credits.ContractSummary, error)
	ListPayments(contractID int64) ([]credits.Payment, error)
}

// HistoryInvestments is what the net worth history reads from the investments
// module.
type HistoryInvestments interface {
	ListEntities() ([]investments.EntitySummary, error)
	ListInstrumentsByEntity(entityID int64) ([]investments.InstrumentSummary, error)
	ListSnapshots(instrumentID int64) ([]investments.Snapshot, error)
}

// HistorySavings is what the net worth history reads from the savings module.
type HistorySavings interface {
	ListAccounts() ([]savings.AccountSummary, error)
	ListMovements(accountID int64) ([]savings.Movement, error)
	ListSnapshots(accountID int64) ([]savings.Snapshot, error)
}

// HistoryCashflow is what the net worth history reads from the cashflow module.
type HistoryCashflow interface {
	ListMonths() ([]cashflow.MonthSummary, error)
	CashBalances(months []string) ([]cashflow.CashBalance, error)
}

// NetWorthMonth is what the household had and owed at the end of one month.
type NetWorthMonth struct {
	// Month is the month as YYYY-MM.
	Month            string
	SavingsCents     money.Cents
	InvestmentsCents money.Cents
	GoalsCents       money.Cents
	CashCents        money.Cents
	// AssetsCents adds the savings, investments, goals and cash.
	AssetsCents money.Cents
	// CreditCents is the principal still owed on the credit contracts that had
	// started.
	CreditCents money.Cents
	// NetCents is AssetsCents minus CreditCents and can be negative.
	NetCents money.Cents
}

// History works out net worth month by month. Nothing is stored: every month
// comes from the typed data of the modules as it stands now.
type History struct {
	goals       HistoryGoals
	credits     HistoryCredits
	investments HistoryInvestments
	savings     HistorySavings
	cashflow    HistoryCashflow
}

// NewHistory creates a History reading from the given modules.
func NewHistory(g HistoryGoals, c HistoryCredits, i HistoryInvestments, s HistorySavings, cf HistoryCashflow) *History {
	return &History{goals: g, credits: c, investments: i, savings: s, cashflow: cf}
}

// NetWorth returns one entry per month that has cashflow, oldest first. Each
// figure is the one at the end of that month: a savings account or an
// instrument without a snapshot that early keeps the last earlier one, and
// cash comes from the typed bank balances (a person with none is left out).
func (h *History) NetWorth() ([]NetWorthMonth, error) {
	summaries, err := h.cashflow.ListMonths()
	if err != nil {
		return nil, fmt.Errorf("list cashflow months: %w", err)
	}
	months := make([]string, 0, len(summaries))
	for _, s := range summaries {
		months = append(months, s.Month)
	}
	sort.Strings(months)
	if len(months) == 0 {
		return []NetWorthMonth{}, nil
	}

	saved, err := h.savingsSeries()
	if err != nil {
		return nil, err
	}
	invested, err := h.investmentsSeries()
	if err != nil {
		return nil, err
	}
	goalSaved, err := h.goalsSeries()
	if err != nil {
		return nil, err
	}
	owed, err := h.creditsSeries()
	if err != nil {
		return nil, err
	}
	balances, err := h.cashflow.CashBalances(months)
	if err != nil {
		return nil, fmt.Errorf("cash balances: %w", err)
	}
	cash := make(map[string]money.Cents)
	for _, b := range balances {
		cash[b.Month] += b.BalanceCents
	}

	out := make([]NetWorthMonth, 0, len(months))
	for _, m := range months {
		n := NetWorthMonth{
			Month: m, SavingsCents: saved(m), InvestmentsCents: invested(m),
			GoalsCents: goalSaved(m), CashCents: cash[m], CreditCents: owed(m),
		}
		n.AssetsCents = n.SavingsCents + n.InvestmentsCents + n.GoalsCents + n.CashCents
		n.NetCents = n.AssetsCents - n.CreditCents
		out = append(out, n)
	}
	return out, nil
}

// savingsSeries returns the total savings balance at the end of a month: each
// account as it was with only the movements and snapshots up to that month.
func (h *History) savingsSeries() (func(string) money.Cents, error) {
	accounts, err := h.savings.ListAccounts()
	if err != nil {
		return nil, fmt.Errorf("list savings accounts: %w", err)
	}
	type data struct {
		account   savings.Account
		movements []savings.Movement
		snapshots []savings.Snapshot
	}
	all := make([]data, 0, len(accounts))
	for _, a := range accounts {
		movements, err := h.savings.ListMovements(a.ID)
		if err != nil {
			return nil, fmt.Errorf("list movements of savings account %d: %w", a.ID, err)
		}
		snapshots, err := h.savings.ListSnapshots(a.ID)
		if err != nil {
			return nil, fmt.Errorf("list snapshots of savings account %d: %w", a.ID, err)
		}
		all = append(all, data{a.Account, movements, snapshots})
	}
	return func(m string) money.Cents {
		var total money.Cents
		for _, d := range all {
			var mv []savings.Movement
			for _, x := range d.movements {
				if x.Month <= m {
					mv = append(mv, x)
				}
			}
			var sn []savings.Snapshot
			for _, x := range d.snapshots {
				if x.Month <= m {
					sn = append(sn, x)
				}
			}
			total += savings.Summarize(d.account, mv, sn).BalanceCents
		}
		return total
	}, nil
}

// investmentsSeries returns the total value of the instruments at the end of a
// month: the latest snapshot of each up to that month, or nothing before the
// first one.
func (h *History) investmentsSeries() (func(string) money.Cents, error) {
	entities, err := h.investments.ListEntities()
	if err != nil {
		return nil, fmt.Errorf("list investment entities: %w", err)
	}
	var snapshots [][]investments.Snapshot
	for _, e := range entities {
		instruments, err := h.investments.ListInstrumentsByEntity(e.ID)
		if err != nil {
			return nil, fmt.Errorf("list instruments of entity %d: %w", e.ID, err)
		}
		for _, in := range instruments {
			list, err := h.investments.ListSnapshots(in.ID)
			if err != nil {
				return nil, fmt.Errorf("list snapshots of instrument %d: %w", in.ID, err)
			}
			snapshots = append(snapshots, list)
		}
	}
	return func(m string) money.Cents {
		var total money.Cents
		for _, list := range snapshots {
			var latest *investments.Snapshot
			for i := range list {
				s := &list[i]
				if s.Month > m {
					continue
				}
				if latest == nil || s.Month > latest.Month || (s.Month == latest.Month && s.ID > latest.ID) {
					latest = s
				}
			}
			if latest != nil {
				total += latest.ValueCents
			}
		}
		return total
	}, nil
}

// goalsSeries returns the money saved in the active goals at the end of a
// month: allocations less expenses and releases, plus covers, up to that month, as the goal
// summary adds them up.
func (h *History) goalsSeries() (func(string) money.Cents, error) {
	accounts, err := h.goals.ListAccounts()
	if err != nil {
		return nil, fmt.Errorf("list goal accounts: %w", err)
	}
	var movements []goals.Movement
	active := make(map[int64]bool)
	for _, a := range accounts {
		summaries, err := h.goals.ListGoalSummaries(a.ID)
		if err != nil {
			return nil, fmt.Errorf("list goals of account %d: %w", a.ID, err)
		}
		for _, g := range summaries {
			if g.Status == goals.StatusActive {
				active[g.GoalID] = true
			}
		}
		list, err := h.goals.ListMovements(a.ID)
		if err != nil {
			return nil, fmt.Errorf("list movements of goal account %d: %w", a.ID, err)
		}
		movements = append(movements, list...)
	}
	return func(m string) money.Cents {
		var total money.Cents
		for _, x := range movements {
			if x.GoalID == nil || !active[*x.GoalID] || x.Month > m {
				continue
			}
			switch x.Type {
			case goals.MovementAllocation, goals.MovementCover:
				total += x.AmountCents
			case goals.MovementExpense, goals.MovementRelease:
				total -= x.AmountCents
			}
		}
		return total
	}, nil
}

// creditsSeries returns what was still owed at the end of a month: the
// original principal of each contract that had started, less the principal paid
// up to that month, never below zero.
func (h *History) creditsSeries() (func(string) money.Cents, error) {
	contracts, err := h.credits.ListContracts()
	if err != nil {
		return nil, fmt.Errorf("list credit contracts: %w", err)
	}
	payments := make([][]credits.Payment, len(contracts))
	for i, c := range contracts {
		if payments[i], err = h.credits.ListPayments(c.ID); err != nil {
			return nil, fmt.Errorf("list payments of contract %d: %w", c.ID, err)
		}
	}
	return func(m string) money.Cents {
		var total money.Cents
		for i, c := range contracts {
			if c.StartMonth > m {
				continue
			}
			owed := c.OriginalPrincipalCents
			for _, p := range payments[i] {
				if p.Month <= m {
					owed -= p.PrincipalCents
				}
			}
			total += max(owed, 0)
		}
		return total
	}, nil
}

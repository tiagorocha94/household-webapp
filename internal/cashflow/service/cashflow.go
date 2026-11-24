package service

import (
	"errors"
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/cashflow"
	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/month"
)

// ListCategories returns the fixed set of categories.
func (s *Service) ListCategories() ([]cashflow.Category, error) {
	return s.store.ListCategories()
}

// ExpensesByRole returns what the expenses of the categories with a role add up
// to in each month, keyed by YYYY-MM.
func (s *Service) ExpensesByRole(role string) (map[string]money.Cents, error) {
	return s.store.ExpensesByRole(role)
}

// CashBalances returns each person's cash at the end of each of the given
// months, from their typed bank balances and the cashflow in between. A person
// with no typed balance is counted from zero at their first movement.
func (s *Service) CashBalances(months []string) ([]cashflow.CashBalance, error) {
	anchors, err := s.store.ListCashAnchors()
	if err != nil {
		return nil, err
	}
	nets, err := s.store.NetsByUserMonth()
	if err != nil {
		return nil, err
	}
	return cashflow.CashBalances(anchors, nets, months), nil
}

// OwnCash returns what the named person had in the bank at the end of the
// given month, or nil when they have neither a typed balance nor a movement. It
// fails with a bad-request error for a malformed month.
func (s *Service) OwnCash(username, m string) (*money.Cents, error) {
	if err := validateMonth(m); err != nil {
		return nil, err
	}
	balances, err := s.CashBalances([]string{m})
	if err != nil {
		return nil, err
	}
	for _, b := range balances {
		if b.User == username {
			cents := b.BalanceCents
			return &cents, nil
		}
	}
	return nil, nil
}

// ListMonths returns one summary per month that has movements, newest first.
func (s *Service) ListMonths() ([]cashflow.MonthSummary, error) {
	return s.store.ListMonthSummaries()
}

// GetMonth returns one month with its totals by category and by person. It
// fails with a bad-request error for a malformed month and a not-found error
// for a month without movements.
func (s *Service) GetMonth(m string) (*cashflow.MonthDetail, error) {
	if err := validateMonth(m); err != nil {
		return nil, err
	}
	summary, err := s.store.GetMonthSummary(m)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return nil, fmt.Errorf("get cashflow month %s: %w", m, apperr.ErrNotFound)
	}
	byCategory, err := s.store.ListMonthCategoryTotals(m)
	if err != nil {
		return nil, err
	}
	byUser, err := s.store.ListMonthUserTotals(m)
	if err != nil {
		return nil, err
	}
	groups, err := s.store.ListBudgetGroups()
	if err != nil {
		return nil, err
	}
	categoryGroups, err := s.store.ListCategoryGroups()
	if err != nil {
		return nil, err
	}
	saved, err := s.markRoles(byCategory)
	if err != nil {
		return nil, err
	}
	history, err := s.peopleHistory(m, byUser)
	if err != nil {
		return nil, err
	}
	return &cashflow.MonthDetail{
		MonthSummary: *summary, ByCategory: byCategory, ByUser: byUser, SavedCents: saved,
		PeopleHistory: history,
		ByGroup:       budgetByGroup(summary.IncomeCents, groups, categoryGroups, byCategory),
	}, nil
}

// peopleHistory returns each person's income and money put aside in the six
// months ending at m, and fills in what each person in byUser put aside and
// their cash in m.
func (s *Service) peopleHistory(m string, byUser []cashflow.UserTotal) ([]cashflow.UserMonthTotal, error) {
	first, err := month.Add(m, -5)
	if err != nil {
		return nil, err
	}
	history, err := s.store.ListUserMonthTotals(first, m)
	if err != nil {
		return nil, err
	}
	cash, err := s.CashBalances([]string{m})
	if err != nil {
		return nil, err
	}
	for i := range byUser {
		for _, h := range history {
			if h.Month == m && h.Username == byUser[i].Username {
				byUser[i].PutAsideCents = h.PutAsideCents
			}
		}
		for _, c := range cash {
			if c.User == byUser[i].Username {
				balance := c.BalanceCents
				byUser[i].CashCents = &balance
			}
		}
	}
	return history, nil
}

// GetBudget returns a month's budget by group for the phone. Unlike GetMonth,
// a month without movements is not an error: it has no income and no spending,
// with the groups listed. It fails with a bad-request error for a malformed
// month.
func (s *Service) GetBudget(m string) (*cashflow.Budget, error) {
	if err := validateMonth(m); err != nil {
		return nil, err
	}
	summaries, err := s.store.ListMonthSummaries()
	if err != nil {
		return nil, err
	}
	months := make([]string, len(summaries))
	for i, summary := range summaries {
		months[i] = summary.Month
	}

	detail, err := s.GetMonth(m)
	if errors.Is(err, apperr.ErrNotFound) {
		groups, err := s.store.ListBudgetGroups()
		if err != nil {
			return nil, err
		}
		return &cashflow.Budget{Month: m, ByGroup: budgetByGroup(0, groups, nil, nil), Months: months}, nil
	}
	if err != nil {
		return nil, err
	}
	return &cashflow.Budget{
		Month: m, IncomeCents: detail.IncomeCents, ExpenseCents: detail.ExpenseCents,
		NetCents: detail.NetCents, ByGroup: detail.ByGroup, Months: months,
	}, nil
}

// markRoles sets Role on the totals of expense categories that have one and
// returns the amount the savings categories add up to.
func (s *Service) markRoles(totals []cashflow.CategoryTotal) (money.Cents, error) {
	categories, err := s.store.ListCategories()
	if err != nil {
		return 0, err
	}
	roles := make(map[int64]string, len(categories))
	for _, c := range categories {
		roles[c.ID] = c.Role
	}
	var saved money.Cents
	for i := range totals {
		if totals[i].Direction != cashflow.DirectionExpense {
			continue
		}
		totals[i].Role = roles[totals[i].CategoryID]
		if totals[i].Role == cashflow.RoleSavings {
			saved += totals[i].TotalCents
		}
	}
	return saved, nil
}

// ListMonthMovements returns every movement of a month, newest date first. A
// malformed month is a bad request; a month without movements is an empty
// list, not an error.
func (s *Service) ListMonthMovements(m string) ([]cashflow.Movement, error) {
	if err := validateMonth(m); err != nil {
		return nil, err
	}
	return s.store.ListMovementsByMonth(m)
}

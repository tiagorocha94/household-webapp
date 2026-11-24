package service

import (
	"github.com/tiagorocha94/household-webapp/internal/cashflow"
	"github.com/tiagorocha94/household-webapp/internal/money"
)

// budgetByGroup returns the budget of every group for a month: each group's
// share of the income and what its categories spent. It returns nil when there
// are no groups.
func budgetByGroup(
	income money.Cents,
	groups []cashflow.BudgetGroup,
	categoryGroups map[int64]int64,
	byCategory []cashflow.CategoryTotal,
) []cashflow.GroupTotal {
	if len(groups) == 0 {
		return nil
	}
	shares := make([]int, len(groups))
	for i, g := range groups {
		shares[i] = g.SharePct
	}
	budgets := splitBudget(income, shares)

	out := make([]cashflow.GroupTotal, len(groups))
	index := make(map[int64]int, len(groups))
	for i, g := range groups {
		index[g.ID] = i
		out[i] = cashflow.GroupTotal{
			Name: g.Name, SharePct: g.SharePct, BudgetCents: budgets[i],
			Categories: []cashflow.CategoryTotal{},
		}
	}
	for _, c := range byCategory {
		if c.Direction != "expense" {
			continue
		}
		groupID, ok := categoryGroups[c.CategoryID]
		if !ok {
			continue
		}
		i, ok := index[groupID]
		if !ok {
			continue
		}
		out[i].SpentCents += c.TotalCents
		out[i].Categories = append(out[i].Categories, c)
	}
	for i := range out {
		out[i].LeftCents = out[i].BudgetCents - out[i].SpentCents
	}
	return out
}

// splitBudget divides income by whole-percent shares into amounts that add up
// to income exactly: each share is rounded down to the cent and the cents that
// are left go to the largest share (the first one on a tie).
func splitBudget(income money.Cents, shares []int) []money.Cents {
	out := make([]money.Cents, len(shares))
	var assigned money.Cents
	largest := 0
	for i, pct := range shares {
		out[i] = income * money.Cents(pct) / 100
		assigned += out[i]
		if pct > shares[largest] {
			largest = i
		}
	}
	if len(out) > 0 {
		out[largest] += income - assigned
	}
	return out
}

package cashflow

import (
	"sort"

	"github.com/tiagorocha94/household-webapp/internal/money"
)

// CashAnchor is what one person's bank account held at the end of a month, as
// typed in. It can be negative.
type CashAnchor struct {
	User         string
	Month        string
	BalanceCents money.Cents
}

// CashBalance is one person's cash at the end of a month.
type CashBalance struct {
	User         string
	Month        string
	BalanceCents money.Cents
	// Typed is true when the balance was typed in for that very month, false
	// when it was worked out from another month's balance.
	Typed bool
}

// CashBalances works out each person's cash at the end of each of the given
// months. The balance is the latest typed one at or before the month plus the
// net cashflow after it, up to and including the month; a month before every
// typed balance counts backwards from the earliest one, taking off the net
// cashflow in between. A person with no typed balance starts from zero before
// their first movement, so their cash is the net cashflow up to and including
// the month; a person with neither a balance nor a movement has none. nets
// holds each person's net cashflow (income minus expenses) by YYYY-MM month. The
// result is ordered by person and then month.
func CashBalances(anchors []CashAnchor, nets map[string]map[string]money.Cents, months []string) []CashBalance {
	byUser := make(map[string][]CashAnchor)
	for _, a := range anchors {
		byUser[a.User] = append(byUser[a.User], a)
	}
	for user, months := range nets {
		if _, typed := byUser[user]; !typed && len(months) > 0 {
			byUser[user] = nil // no typed balance: counted from zero
		}
	}
	users := make([]string, 0, len(byUser))
	for user, list := range byUser {
		sort.Slice(list, func(i, j int) bool { return list[i].Month < list[j].Month })
		users = append(users, user)
	}
	sort.Strings(users)

	var out []CashBalance
	for _, user := range users {
		list := byUser[user]
		for _, m := range months {
			out = append(out, cashAt(user, list, nets[user], m))
		}
	}
	return out
}

// cashAt is the balance of one person at the end of month m. list is sorted by
// month and is empty when the person never typed a balance.
func cashAt(user string, list []CashAnchor, nets map[string]money.Cents, m string) CashBalance {
	if len(list) == 0 {
		var balance money.Cents
		for month, net := range nets {
			if month <= m {
				balance += net
			}
		}
		return CashBalance{User: user, Month: m, BalanceCents: balance}
	}
	// The latest typed balance at or before m, if there is one.
	before := -1
	for i, a := range list {
		if a.Month <= m {
			before = i
		}
	}
	if before >= 0 {
		a := list[before]
		balance := a.BalanceCents
		for month, net := range nets {
			if month > a.Month && month <= m {
				balance += net
			}
		}
		return CashBalance{User: user, Month: m, BalanceCents: balance, Typed: a.Month == m}
	}
	// Every typed balance is later: count back from the earliest one.
	a := list[0]
	balance := a.BalanceCents
	for month, net := range nets {
		if month > m && month <= a.Month {
			balance -= net
		}
	}
	return CashBalance{User: user, Month: m, BalanceCents: balance}
}

package cashflow

import (
	"reflect"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/money"
)

func TestCashBalancesCountForwardAndBackFromTheTypedBalance(t *testing.T) {
	t.Parallel()

	anchors := []CashAnchor{{User: "alice", Month: "2026-03", BalanceCents: 100000}}
	nets := map[string]map[string]money.Cents{"alice": {
		"2026-01": 10000, "2026-02": -20000, "2026-03": 5000, "2026-04": 30000, "2026-06": -7000,
	}}
	months := []string{"2026-01", "2026-02", "2026-03", "2026-04", "2026-05", "2026-06"}

	got := CashBalances(anchors, nets, months)
	want := []CashBalance{
		// Counting back from March: February is March minus March's net, and
		// January is February minus February's net.
		{User: "alice", Month: "2026-01", BalanceCents: 100000 - 5000 + 20000},
		{User: "alice", Month: "2026-02", BalanceCents: 100000 - 5000},
		{User: "alice", Month: "2026-03", BalanceCents: 100000, Typed: true},
		{User: "alice", Month: "2026-04", BalanceCents: 130000},
		{User: "alice", Month: "2026-05", BalanceCents: 130000},
		{User: "alice", Month: "2026-06", BalanceCents: 123000},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CashBalances() = %+v, want %+v", got, want)
	}
}

func TestCashBalancesUseTheLatestTypedBalanceAndKeepPeopleApart(t *testing.T) {
	t.Parallel()

	anchors := []CashAnchor{
		{User: "bob", Month: "2026-05", BalanceCents: -2000},
		{User: "alice", Month: "2026-01", BalanceCents: 50000},
		{User: "alice", Month: "2026-04", BalanceCents: 80000},
	}
	nets := map[string]map[string]money.Cents{
		"alice": {"2026-02": 1000, "2026-05": 500},
		"bob":   {"2026-05": 900, "2026-06": 100},
	}

	got := CashBalances(anchors, nets, []string{"2026-03", "2026-05"})
	want := []CashBalance{
		{User: "alice", Month: "2026-03", BalanceCents: 51000},
		{User: "alice", Month: "2026-05", BalanceCents: 80500},
		// Bob's only balance is May's, so March counts back from it.
		{User: "bob", Month: "2026-03", BalanceCents: -2000 - 900},
		{User: "bob", Month: "2026-05", BalanceCents: -2000, Typed: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CashBalances() = %+v, want %+v", got, want)
	}
}

func TestCashBalancesStartFromZeroForAPersonWithoutATypedBalance(t *testing.T) {
	t.Parallel()

	nets := map[string]map[string]money.Cents{
		"alice": {"2026-02": 100, "2026-04": -30},
		"bob":   {},
	}
	got := CashBalances(nil, nets, []string{"2026-01", "2026-02", "2026-03", "2026-04"})
	want := []CashBalance{
		// Before the first movement there is nothing yet.
		{User: "alice", Month: "2026-01", BalanceCents: 0},
		{User: "alice", Month: "2026-02", BalanceCents: 100},
		{User: "alice", Month: "2026-03", BalanceCents: 100},
		{User: "alice", Month: "2026-04", BalanceCents: 70},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CashBalances() = %+v, want %+v (bob has no movement, so no cash)", got, want)
	}
}

func TestCashBalancesCountFromZeroOnlyForPeopleWithoutATypedBalance(t *testing.T) {
	t.Parallel()

	anchors := []CashAnchor{{User: "alice", Month: "2026-02", BalanceCents: 1000}}
	nets := map[string]map[string]money.Cents{
		"alice": {"2026-01": 50, "2026-03": 5},
		"bob":   {"2026-01": 50, "2026-03": 5},
	}
	got := CashBalances(anchors, nets, []string{"2026-03"})
	want := []CashBalance{
		{User: "alice", Month: "2026-03", BalanceCents: 1005},
		{User: "bob", Month: "2026-03", BalanceCents: 55},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CashBalances() = %+v, want %+v", got, want)
	}
}

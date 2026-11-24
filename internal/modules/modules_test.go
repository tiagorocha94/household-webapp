package modules

import (
	"slices"
	"strings"
	"testing"
)

func TestTheZeroSetRunsEveryModule(t *testing.T) {
	t.Parallel()

	var s Set
	if got := s.Names(); !slices.Equal(got, All()) {
		t.Errorf("Names() = %v, want %v", got, All())
	}
	for _, name := range All() {
		if !s.Has(name) {
			t.Errorf("Has(%q) = false, want true", name)
		}
	}
}

func TestNewSetWithNoNamesRunsEveryModule(t *testing.T) {
	t.Parallel()

	s, err := NewSet(nil)
	if err != nil {
		t.Fatalf("NewSet(nil) error = %v", err)
	}
	if got := s.Names(); !slices.Equal(got, All()) {
		t.Errorf("Names() = %v, want %v", got, All())
	}
}

func TestNewSetKeepsOnlyTheChosenModulesInTheUsualOrder(t *testing.T) {
	t.Parallel()

	s, err := NewSet([]string{Investments, Cashflow, Cashflow})
	if err != nil {
		t.Fatalf("NewSet() error = %v", err)
	}
	if got, want := s.Names(), []string{Cashflow, Investments}; !slices.Equal(got, want) {
		t.Errorf("Names() = %v, want %v", got, want)
	}
	if !s.Has(Cashflow) || s.Has(Goals) {
		t.Errorf("Has(cashflow) = %v, Has(goals) = %v, want true and false", s.Has(Cashflow), s.Has(Goals))
	}
}

func TestNewSetRejectsAnUnknownModule(t *testing.T) {
	t.Parallel()

	_, err := NewSet([]string{Goals, "goal"})
	if err == nil || !strings.Contains(err.Error(), `"goal"`) {
		t.Fatalf("NewSet() error = %v, want it to name the unknown module", err)
	}
}

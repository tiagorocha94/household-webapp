// Package modules names the optional feature modules and which of them a
// server runs. Authentication, users and devices are always on and are not
// modules.
package modules

import (
	"fmt"
	"slices"
	"strings"
)

// The feature modules.
const (
	Cashflow    = "cashflow"
	Goals       = "goals"
	Savings     = "savings"
	Credits     = "credits"
	Investments = "investments"
)

// Set is the modules a server runs. The zero value runs all of them, so code
// that never chooses gets everything.
type Set struct {
	names []string
}

// All returns every module, in the order they are listed everywhere.
func All() []string {
	return []string{Cashflow, Goals, Savings, Credits, Investments}
}

// NewSet returns the set of the given modules; none given means all. An
// unknown name is an error, so a typo cannot silently turn a module off.
func NewSet(names []string) (Set, error) {
	if len(names) == 0 {
		return Set{}, nil
	}
	chosen := make([]string, 0, len(names))
	for _, name := range names {
		if !slices.Contains(All(), name) {
			return Set{}, fmt.Errorf("unknown module %q (modules are %s)", name, strings.Join(All(), ", "))
		}
		if !slices.Contains(chosen, name) {
			chosen = append(chosen, name)
		}
	}
	return Set{names: chosen}, nil
}

// Has reports whether the module runs.
func (s Set) Has(name string) bool {
	return s.names == nil || slices.Contains(s.names, name)
}

// Names returns the modules that run, in the order of All.
func (s Set) Names() []string {
	return slices.DeleteFunc(All(), func(name string) bool { return !s.Has(name) })
}

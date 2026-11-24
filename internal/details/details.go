// Package details is the contract between cashflow and the modules that keep
// where the money of a movement went: a savings deposit, investment
// contributions or a credit payment, linked to the movement by its sync ID
// (ADR 021). Cashflow only knows the Registry; each module that runs adds an
// Applier that writes into its own tables, so cashflow never touches them and
// a module that does not run is simply absent. An Applier runs inside the
// transaction that writes the movement, so the two arrive together or not at
// all, and the module's CSV files are rewritten after the change (see
// csvstate).
package details

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/tiagorocha94/household-webapp/internal/csvstate"
	"github.com/tiagorocha94/household-webapp/internal/modules"
	"github.com/tiagorocha94/household-webapp/internal/money"
)

// Input is one movement and the detail that goes with it.
type Input struct {
	// UserID is the person whose device sent the movement.
	UserID int64
	SyncID string
	// Category is the movement's category name; its role says which kind of
	// detail it can carry.
	Category string
	// Month is the month of the movement's date, as YYYY-MM.
	Month       string
	AmountCents money.Cents
	// Detail is nil to leave the detail as it is, and an empty Detail to remove
	// it. Either way, a detail that exists follows the movement's month and
	// amount.
	Detail *Detail
}

// Detail is where the money of a movement went. At most one part is set, the
// one that matches the category's role.
type Detail struct {
	Savings     *Savings
	Investments *Investments
	Credit      *Credit
}

// Savings is a deposit into a savings account.
type Savings struct {
	AccountID int64
	// ValueCents is the account's value seen now. Nil records nothing.
	ValueCents *money.Cents
}

// Credit is a payment on a credit. Credits have no owner, so anyone can add
// one. The principal and the interest add up to the movement's amount.
type Credit struct {
	CreditID       int64
	PrincipalCents money.Cents
	InterestCents  money.Cents
	// Extraordinary marks a payment on top of the regular one of the month.
	Extraordinary bool
}

// Investments is money put into the instruments of one investment account. The
// lines may add up to less than the movement while the split is incomplete,
// never to more. No lines keeps nothing, as there is no instrument to hold it.
type Investments struct {
	EntityID int64
	Lines    []InvestmentLine
}

// InvestmentLine is the part of the movement that went into one instrument.
type InvestmentLine struct {
	InstrumentID int64
	AmountCents  money.Cents
	// ValueCents is the instrument's value seen now. Nil records nothing.
	ValueCents *money.Cents
}

// An Applier keeps the detail of one module. Its Module is also the category
// role that module serves. Every method runs in the movement's transaction.
type Applier interface {
	// Module is the module's name, as in package modules.
	Module() string
	// Noun says what the module takes, for a refusal: "a savings detail".
	Noun() string
	// Check says why the detail cannot be written, or "" when it can. The
	// module's part of d is set. It reads only.
	Check(tx *sql.Tx, in Input, d Detail) (string, error)
	// CheckFit refuses a change of the movement that no longer fits what the
	// module stored for it, when the request carries no detail.
	CheckFit(tx *sql.Tx, in Input) (string, error)
	// Apply makes what the module stores match the module's part of d.
	Apply(tx *sql.Tx, in Input, d Detail) error
	// Follow moves what the module stored to the movement's month and amount.
	Follow(tx *sql.Tx, in Input) error
	// Remove deletes what the module stored for the movement.
	Remove(tx *sql.Tx, syncID string) error
}

// Registry holds the appliers of the modules that run, in the order they act.
// A nil Registry has none, so no detail can be saved.
type Registry struct {
	appliers []Applier
}

// kinds names the modules whose part of the detail is set.
func (d *Detail) kinds() []string {
	var out []string
	if d.Savings != nil {
		out = append(out, modules.Savings)
	}
	if d.Investments != nil {
		out = append(out, modules.Investments)
	}
	if d.Credit != nil {
		out = append(out, modules.Credits)
	}
	return out
}

// has reports whether the part of the module is set.
func (d *Detail) has(module string) bool {
	return slices.Contains(d.kinds(), module)
}

// NewRegistry returns a Registry of the given appliers.
func NewRegistry(appliers ...Applier) *Registry {
	return &Registry{appliers: appliers}
}

// Check says why the detail cannot be written, or "" when it can. It reads
// only and changes nothing. A category that does not exist passes: the
// movement's own write refuses it.
func (r *Registry) Check(tx *sql.Tx, in Input) (string, error) {
	role, found, err := categoryRole(tx, in.Category)
	if err != nil || !found {
		return "", err
	}
	d := in.Detail
	if d == nil {
		if a := r.find(role); a != nil {
			return a.CheckFit(tx, in)
		}
		return "", nil
	}
	kinds := d.kinds()
	switch {
	case len(kinds) > 1:
		return "a detail holds one kind of detail, not several", nil
	case len(kinds) == 0:
		return "", nil
	}
	a := r.find(kinds[0])
	if a == nil {
		return fmt.Sprintf("the %s module does not run on this server, so it cannot take a detail", kinds[0]), nil
	}
	if role != kinds[0] {
		return fmt.Sprintf("category %q has no %s role, so it cannot carry %s", in.Category, kinds[0], a.Noun()), nil
	}
	return a.Check(tx, in, *d)
}

// Apply writes the detail. Call it after Check passed and after the movement
// itself was written.
func (r *Registry) Apply(tx *sql.Tx, in Input) error {
	role, _, err := categoryRole(tx, in.Category)
	if err != nil {
		return err
	}
	d := in.Detail
	// A module's detail is replaced when the request carries it, removed when
	// the request carries another detail or the category lost the role, and
	// otherwise follows the movement.
	for _, a := range r.list() {
		switch m := a.Module(); {
		case d != nil && d.has(m):
			err = a.Apply(tx, in, *d)
		case d != nil || role != m:
			err = a.Remove(tx, in.SyncID)
		default:
			err = a.Follow(tx, in)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Remove deletes the detail of a movement that was deleted. Snapshots the
// person typed stay.
func (r *Registry) Remove(tx *sql.Tx, syncID string) error {
	for _, a := range r.list() {
		if err := a.Remove(tx, syncID); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) list() []Applier {
	if r == nil {
		return nil
	}
	return r.appliers
}

func (r *Registry) find(module string) Applier {
	for _, a := range r.list() {
		if a.Module() == module {
			return a
		}
	}
	return nil
}

// categoryRole returns the role of a category, "" when it has none, and
// whether the category exists.
func categoryRole(tx *sql.Tx, name string) (role string, found bool, err error) {
	var r sql.NullString
	err = tx.QueryRow("SELECT role FROM cashflow_categories WHERE name = ?", name).Scan(&r)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("find the role of cashflow category %q: %w", name, err)
	}
	return r.String, true, nil
}

// RaiseIfChanged marks the module's CSV files as behind when the statement
// changed a row. Appliers use it after an UPDATE or DELETE.
func RaiseIfChanged(tx *sql.Tx, res sql.Result, module string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("count the changed %s rows: %w", module, err)
	}
	if n > 0 {
		return csvstate.Raise(tx, module)
	}
	return nil
}

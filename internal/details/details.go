// Package details writes where the money of a cashflow movement went: a savings
// deposit, investment contributions or a credit payment, linked to the movement
// by its sync ID (ADR 021). It runs inside the transaction that writes the
// movement, so the two arrive together or not at all. The rows live in the
// tables of their own modules, and each module's CSV files are rewritten after
// the change (see csvstate).
package details

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/csvstate"
	"github.com/tiagorocha94/household-webapp/internal/money"
)

// Names of the modules whose files a detail changes, as csvstate knows them.
const (
	SavingsModule     = "savings"
	InvestmentsModule = "investments"
	CreditsModule     = "credits"
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

// kinds counts the parts that are set.
func (d *Detail) kinds() int {
	n := 0
	for _, set := range []bool{d.Savings != nil, d.Investments != nil, d.Credit != nil} {
		if set {
			n++
		}
	}
	return n
}

// Check says why the detail cannot be written, or "" when it can. It reads
// only and changes nothing. A category that does not exist passes: the
// movement's own write refuses it.
func Check(tx *sql.Tx, in Input) (string, error) {
	role, found, err := categoryRole(tx, in.Category)
	if err != nil || !found {
		return "", err
	}
	d := in.Detail
	switch {
	case d != nil && d.kinds() > 1:
		return "a detail holds one kind of detail, not several", nil
	case d != nil && d.Savings != nil:
		if role != "savings" {
			return fmt.Sprintf("category %q has no savings role, so it cannot carry a savings detail", in.Category), nil
		}
		return checkSavings(tx, in.UserID, *d.Savings)
	case d != nil && d.Investments != nil:
		if role != "investments" {
			return fmt.Sprintf("category %q has no investments role, so it cannot carry an investments detail", in.Category), nil
		}
		return checkInvestments(tx, in, *d.Investments)
	case d != nil && d.Credit != nil:
		if role != "credits" {
			return fmt.Sprintf("category %q has no credits role, so it cannot carry a credit payment", in.Category), nil
		}
		return checkCredit(tx, in, *d.Credit)
	case d == nil && role == "investments":
		return checkInvestmentsFit(tx, in)
	case d == nil && role == "credits":
		return checkCreditFit(tx, in)
	}
	return "", nil
}

// Apply writes the detail. Call it after Check passed and after the movement
// itself was written.
func Apply(tx *sql.Tx, in Input) error {
	role, _, err := categoryRole(tx, in.Category)
	if err != nil {
		return err
	}
	d := in.Detail
	// A kind of detail is replaced when the request carries it, removed when
	// the request carries another detail or the category lost the role, and
	// otherwise follows the movement.
	switch {
	case d != nil && d.Savings != nil:
		err = applySavings(tx, in, *d.Savings)
	case d != nil || role != "savings":
		err = removeSavings(tx, in.SyncID)
	default:
		err = followSavings(tx, in)
	}
	if err != nil {
		return err
	}
	switch {
	case d != nil && d.Investments != nil:
		err = applyInvestments(tx, in, *d.Investments)
	case d != nil || role != "investments":
		err = removeInvestments(tx, in.SyncID)
	default:
		err = followInvestments(tx, in)
	}
	if err != nil {
		return err
	}
	switch {
	case d != nil && d.Credit != nil:
		return applyCredit(tx, in, *d.Credit)
	case d != nil || role != "credits":
		return removeCredit(tx, in.SyncID)
	default:
		return followCredit(tx, in)
	}
}

// Remove deletes the detail of a movement that was deleted. Snapshots the
// person typed stay.
func Remove(tx *sql.Tx, syncID string) error {
	if err := removeSavings(tx, syncID); err != nil {
		return err
	}
	if err := removeInvestments(tx, syncID); err != nil {
		return err
	}
	return removeCredit(tx, syncID)
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

func checkSavings(tx *sql.Tx, userID int64, s Savings) (string, error) {
	var owner sql.NullInt64
	err := tx.QueryRow("SELECT user_id FROM savings_accounts WHERE id = ?", s.AccountID).Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Sprintf("savings account %d does not exist", s.AccountID), nil
	}
	if err != nil {
		return "", fmt.Errorf("find savings account %d: %w", s.AccountID, err)
	}
	if owner.Valid && owner.Int64 != userID {
		return fmt.Sprintf("savings account %d belongs to another person", s.AccountID), nil
	}
	if s.ValueCents != nil && *s.ValueCents < 0 {
		return "the value of a savings account cannot be negative", nil
	}
	return "", nil
}

// applySavings makes the movement's deposit match, adding it when there is none,
// and records the value seen now when one was given.
func applySavings(tx *sql.Tx, in Input, s Savings) error {
	changed := false

	var id, accountID int64
	var month string
	var cents money.Cents
	err := tx.QueryRow(
		`SELECT id, account_id, month, amount_cents FROM savings_movements
		 WHERE movement_sync_id = ? ORDER BY id LIMIT 1`,
		in.SyncID,
	).Scan(&id, &accountID, &month, &cents)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := tx.Exec(
			`INSERT INTO savings_movements (account_id, month, type, amount_cents, movement_sync_id)
			 VALUES (?, ?, 'deposit', ?, ?)`,
			s.AccountID, in.Month, in.AmountCents, in.SyncID,
		); err != nil {
			return fmt.Errorf("insert the savings deposit of movement %s: %w", in.SyncID, err)
		}
		changed = true
	case err != nil:
		return fmt.Errorf("find the savings deposit of movement %s: %w", in.SyncID, err)
	case accountID != s.AccountID || month != in.Month || cents != in.AmountCents:
		if _, err := tx.Exec(
			`UPDATE savings_movements
			 SET account_id = ?, month = ?, amount_cents = ?, updated_at = CURRENT_TIMESTAMP
			 WHERE id = ?`,
			s.AccountID, in.Month, in.AmountCents, id,
		); err != nil {
			return fmt.Errorf("update the savings deposit of movement %s: %w", in.SyncID, err)
		}
		changed = true
	}

	if s.ValueCents != nil {
		var same int
		if err := tx.QueryRow(
			`SELECT COUNT(*) FROM savings_snapshots WHERE account_id = ? AND month = ? AND value_cents = ?`,
			s.AccountID, in.Month, *s.ValueCents,
		).Scan(&same); err != nil {
			return fmt.Errorf("find the savings snapshot of account %d: %w", s.AccountID, err)
		}
		if same == 0 {
			if _, err := tx.Exec(
				`INSERT INTO savings_snapshots (account_id, month, value_cents) VALUES (?, ?, ?)`,
				s.AccountID, in.Month, *s.ValueCents,
			); err != nil {
				return fmt.Errorf("insert the savings snapshot of account %d: %w", s.AccountID, err)
			}
			changed = true
		}
	}

	if changed {
		return csvstate.Raise(tx, SavingsModule)
	}
	return nil
}

// followSavings moves an existing deposit to the movement's month and amount.
func followSavings(tx *sql.Tx, in Input) error {
	res, err := tx.Exec(
		`UPDATE savings_movements
		 SET month = ?, amount_cents = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE movement_sync_id = ? AND (month IS NOT ? OR amount_cents IS NOT ?)`,
		in.Month, in.AmountCents, in.SyncID, in.Month, in.AmountCents,
	)
	if err != nil {
		return fmt.Errorf("update the savings deposit of movement %s: %w", in.SyncID, err)
	}
	return raiseIfChanged(tx, res, SavingsModule)
}

func removeSavings(tx *sql.Tx, syncID string) error {
	res, err := tx.Exec("DELETE FROM savings_movements WHERE movement_sync_id = ?", syncID)
	if err != nil {
		return fmt.Errorf("delete the savings deposit of movement %s: %w", syncID, err)
	}
	return raiseIfChanged(tx, res, SavingsModule)
}

func raiseIfChanged(tx *sql.Tx, res sql.Result, module string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("count the changed %s rows: %w", module, err)
	}
	if n > 0 {
		return csvstate.Raise(tx, module)
	}
	return nil
}

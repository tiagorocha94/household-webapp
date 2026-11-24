// Package detail keeps the credits part of a cashflow movement's detail, in this
// module's own tables. It is the credits Applier of package details.
package detail

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/csvstate"
	"github.com/tiagorocha94/household-webapp/internal/details"
	"github.com/tiagorocha94/household-webapp/internal/modules"
	"github.com/tiagorocha94/household-webapp/internal/money"
)

// Applier is the credits module's part of the detail of a movement.
type Applier struct{}

// creditPayment is a stored payment of a movement.
type creditPayment struct {
	id, creditID        int64
	month, kind         string
	principal, interest money.Cents
}

// Module returns the module's name.
func (Applier) Module() string { return modules.Credits }

// Noun says what the module takes.
func (Applier) Noun() string { return "a credit payment" }

// Check says why the detail cannot be written, or "" when it can.
func (Applier) Check(tx *sql.Tx, in details.Input, d details.Detail) (string, error) {
	return checkCredit(tx, in, *d.Credit)
}

// CheckFit refuses a change of the movement that no longer fits what is stored.
func (Applier) CheckFit(tx *sql.Tx, in details.Input) (string, error) { return checkCreditFit(tx, in) }

// Apply makes what is stored match the detail.
func (Applier) Apply(tx *sql.Tx, in details.Input, d details.Detail) error {
	return applyCredit(tx, in, *d.Credit)
}

// Follow moves what is stored to the movement's month and amount.
func (Applier) Follow(tx *sql.Tx, in details.Input) error { return followCredit(tx, in) }

// Remove deletes what is stored for the movement.
func (Applier) Remove(tx *sql.Tx, syncID string) error { return removeCredit(tx, syncID) }

func paymentType(c details.Credit) string {
	if c.Extraordinary {
		return "extraordinary"
	}
	return "regular"
}

func checkCredit(tx *sql.Tx, in details.Input, c details.Credit) (string, error) {
	if c.PrincipalCents < 0 || c.InterestCents < 0 {
		return "the principal and the interest of a payment cannot be negative", nil
	}
	if c.PrincipalCents+c.InterestCents != in.AmountCents {
		return fmt.Sprintf("the principal and the interest add up to %d cents, want the movement's %d",
			c.PrincipalCents+c.InterestCents, in.AmountCents), nil
	}
	var start string
	err := tx.QueryRow("SELECT start_month FROM credit_contracts WHERE id = ?", c.CreditID).Scan(&start)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Sprintf("credit %d does not exist", c.CreditID), nil
	}
	if err != nil {
		return "", fmt.Errorf("find credit %d: %w", c.CreditID, err)
	}
	if in.Month < start {
		return fmt.Sprintf("the payment month %s is before the credit starts in %s", in.Month, start), nil
	}
	return regularTaken(tx, in.SyncID, c.CreditID, in.Month, paymentType(c))
}

// checkCreditFit refuses a change of the movement that no longer matches the
// stored payment, or that moves a regular payment onto a month that already
// has one. The device sends the detail again with the change.
func checkCreditFit(tx *sql.Tx, in details.Input) (string, error) {
	p, found, err := paymentOf(tx, in.SyncID)
	if err != nil || !found {
		return "", err
	}
	if p.principal+p.interest != in.AmountCents {
		return fmt.Sprintf("the movement is now %d cents but the credit payment is %d: send the detail again",
			in.AmountCents, p.principal+p.interest), nil
	}
	return regularTaken(tx, in.SyncID, p.creditID, in.Month, p.kind)
}

// regularTaken says why a regular payment cannot be in the month: there is one
// already, from another movement or from the files.
func regularTaken(tx *sql.Tx, syncID string, creditID int64, month, kind string) (string, error) {
	if kind != "regular" {
		return "", nil
	}
	var taken int
	if err := tx.QueryRow(
		`SELECT COUNT(*) FROM credit_payments
		 WHERE contract_id = ? AND month = ? AND type = 'regular'
		   AND COALESCE(movement_sync_id, '') != ?`,
		creditID, month, syncID,
	).Scan(&taken); err != nil {
		return "", fmt.Errorf("find the regular payment of credit %d in %s: %w", creditID, month, err)
	}
	if taken > 0 {
		return fmt.Sprintf("credit %d already has a regular payment in %s", creditID, month), nil
	}
	return "", nil
}

func paymentOf(tx *sql.Tx, syncID string) (creditPayment, bool, error) {
	var p creditPayment
	err := tx.QueryRow(
		`SELECT id, contract_id, month, type, principal_cents, interest_cents
		 FROM credit_payments WHERE movement_sync_id = ? ORDER BY id LIMIT 1`,
		syncID,
	).Scan(&p.id, &p.creditID, &p.month, &p.kind, &p.principal, &p.interest)
	if errors.Is(err, sql.ErrNoRows) {
		return p, false, nil
	}
	if err != nil {
		return p, false, fmt.Errorf("find the credit payment of movement %s: %w", syncID, err)
	}
	return p, true, nil
}

// applyCredit makes the movement's payment match, adding it when there is none.
func applyCredit(tx *sql.Tx, in details.Input, c details.Credit) error {
	p, found, err := paymentOf(tx, in.SyncID)
	if err != nil {
		return err
	}
	switch {
	case !found:
		if _, err := tx.Exec(
			`INSERT INTO credit_payments (contract_id, month, type, principal_cents, interest_cents, movement_sync_id)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			c.CreditID, in.Month, paymentType(c), c.PrincipalCents, c.InterestCents, in.SyncID,
		); err != nil {
			return fmt.Errorf("insert the credit payment of movement %s: %w", in.SyncID, err)
		}
	case p.creditID != c.CreditID || p.month != in.Month || p.kind != paymentType(c) ||
		p.principal != c.PrincipalCents || p.interest != c.InterestCents:
		if _, err := tx.Exec(
			`UPDATE credit_payments
			 SET contract_id = ?, month = ?, type = ?, principal_cents = ?, interest_cents = ?, updated_at = CURRENT_TIMESTAMP
			 WHERE id = ?`,
			c.CreditID, in.Month, paymentType(c), c.PrincipalCents, c.InterestCents, p.id,
		); err != nil {
			return fmt.Errorf("update the credit payment of movement %s: %w", in.SyncID, err)
		}
	default:
		return nil
	}
	return csvstate.Raise(tx, modules.Credits)
}

// followCredit moves an existing payment to the movement's month. Its amount
// already matches, as checkCreditFit has made sure.
func followCredit(tx *sql.Tx, in details.Input) error {
	res, err := tx.Exec(
		`UPDATE credit_payments SET month = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE movement_sync_id = ? AND month != ?`,
		in.Month, in.SyncID, in.Month,
	)
	if err != nil {
		return fmt.Errorf("move the credit payment of movement %s: %w", in.SyncID, err)
	}
	return details.RaiseIfChanged(tx, res, modules.Credits)
}

func removeCredit(tx *sql.Tx, syncID string) error {
	res, err := tx.Exec("DELETE FROM credit_payments WHERE movement_sync_id = ?", syncID)
	if err != nil {
		return fmt.Errorf("delete the credit payment of movement %s: %w", syncID, err)
	}
	return details.RaiseIfChanged(tx, res, modules.Credits)
}

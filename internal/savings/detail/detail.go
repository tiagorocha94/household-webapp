// Package detail keeps the savings part of a cashflow movement's detail, in this
// module's own tables. It is the savings Applier of package details.
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

// Applier is the savings module's part of the detail of a movement.
type Applier struct{}

// Module returns the module's name.
func (Applier) Module() string { return modules.Savings }

// Noun says what the module takes.
func (Applier) Noun() string { return "a savings detail" }

// Check says why the detail cannot be written, or "" when it can.
func (Applier) Check(tx *sql.Tx, in details.Input, d details.Detail) (string, error) {
	return checkSavings(tx, in.UserID, *d.Savings)
}

// CheckFit refuses a change of the movement that no longer fits what is stored.
func (Applier) CheckFit(*sql.Tx, details.Input) (string, error) { return "", nil }

// Apply makes what is stored match the detail.
func (Applier) Apply(tx *sql.Tx, in details.Input, d details.Detail) error {
	return applySavings(tx, in, *d.Savings)
}

// Follow moves what is stored to the movement's month and amount.
func (Applier) Follow(tx *sql.Tx, in details.Input) error { return followSavings(tx, in) }

// Remove deletes what is stored for the movement.
func (Applier) Remove(tx *sql.Tx, syncID string) error { return removeSavings(tx, syncID) }

func checkSavings(tx *sql.Tx, userID int64, s details.Savings) (string, error) {
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
func applySavings(tx *sql.Tx, in details.Input, s details.Savings) error {
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
		return csvstate.Raise(tx, modules.Savings)
	}
	return nil
}

// followSavings moves an existing deposit to the movement's month and amount.
func followSavings(tx *sql.Tx, in details.Input) error {
	res, err := tx.Exec(
		`UPDATE savings_movements
		 SET month = ?, amount_cents = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE movement_sync_id = ? AND (month IS NOT ? OR amount_cents IS NOT ?)`,
		in.Month, in.AmountCents, in.SyncID, in.Month, in.AmountCents,
	)
	if err != nil {
		return fmt.Errorf("update the savings deposit of movement %s: %w", in.SyncID, err)
	}
	return details.RaiseIfChanged(tx, res, modules.Savings)
}

func removeSavings(tx *sql.Tx, syncID string) error {
	res, err := tx.Exec("DELETE FROM savings_movements WHERE movement_sync_id = ?", syncID)
	if err != nil {
		return fmt.Errorf("delete the savings deposit of movement %s: %w", syncID, err)
	}
	return details.RaiseIfChanged(tx, res, modules.Savings)
}

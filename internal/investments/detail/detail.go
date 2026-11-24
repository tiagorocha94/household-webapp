// Package detail keeps the investments part of a cashflow movement's detail, in this
// module's own tables. It is the investments Applier of package details.
package detail

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/csvstate"
	"github.com/tiagorocha94/household-webapp/internal/details"
	"github.com/tiagorocha94/household-webapp/internal/modules"
	"github.com/tiagorocha94/household-webapp/internal/money"
	"sort"
)

// Applier is the investments module's part of the detail of a movement.
type Applier struct{}

// Module returns the module's name.
func (Applier) Module() string { return modules.Investments }

// Noun says what the module takes.
func (Applier) Noun() string { return "an investments detail" }

// Check says why the detail cannot be written, or "" when it can.
func (Applier) Check(tx *sql.Tx, in details.Input, d details.Detail) (string, error) {
	return checkInvestments(tx, in, *d.Investments)
}

// CheckFit refuses a change of the movement that no longer fits what is stored.
func (Applier) CheckFit(tx *sql.Tx, in details.Input) (string, error) {
	return checkInvestmentsFit(tx, in)
}

// Apply makes what is stored match the detail.
func (Applier) Apply(tx *sql.Tx, in details.Input, d details.Detail) error {
	return applyInvestments(tx, in, *d.Investments)
}

// Follow moves what is stored to the movement's month and amount.
func (Applier) Follow(tx *sql.Tx, in details.Input) error { return followInvestments(tx, in) }

// Remove deletes what is stored for the movement.
func (Applier) Remove(tx *sql.Tx, syncID string) error { return removeInvestments(tx, syncID) }

type contribution struct {
	instrumentID int64
	month        string
	cents        money.Cents
}

func checkInvestments(tx *sql.Tx, in details.Input, inv details.Investments) (string, error) {
	var owner sql.NullInt64
	err := tx.QueryRow("SELECT user_id FROM investment_entities WHERE id = ?", inv.EntityID).Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Sprintf("investment account %d does not exist", inv.EntityID), nil
	}
	if err != nil {
		return "", fmt.Errorf("find investment account %d: %w", inv.EntityID, err)
	}
	if owner.Valid && owner.Int64 != in.UserID {
		return fmt.Sprintf("investment account %d belongs to another person", inv.EntityID), nil
	}

	var total money.Cents
	seen := make(map[int64]bool, len(inv.Lines))
	for _, l := range inv.Lines {
		if seen[l.InstrumentID] {
			return fmt.Sprintf("investment instrument %d appears more than once", l.InstrumentID), nil
		}
		seen[l.InstrumentID] = true
		if l.AmountCents <= 0 {
			return fmt.Sprintf("the amount for investment instrument %d must be above zero", l.InstrumentID), nil
		}
		if l.ValueCents != nil && *l.ValueCents < 0 {
			return "the value of an investment instrument cannot be negative", nil
		}
		var entityID int64
		err := tx.QueryRow("SELECT entity_id FROM investment_instruments WHERE id = ?", l.InstrumentID).Scan(&entityID)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Sprintf("investment instrument %d does not exist", l.InstrumentID), nil
		}
		if err != nil {
			return "", fmt.Errorf("find investment instrument %d: %w", l.InstrumentID, err)
		}
		if entityID != inv.EntityID {
			return fmt.Sprintf("investment instrument %d is not in investment account %d", l.InstrumentID, inv.EntityID), nil
		}
		total += l.AmountCents
	}
	if total > in.AmountCents {
		return fmt.Sprintf("the instruments add up to %d cents, more than the movement's %d", total, in.AmountCents), nil
	}
	return "", nil
}

// checkInvestmentsFit refuses a change of the movement that leaves the stored
// split larger than the movement, since the split would then be wrong. The
// device sends the detail again with the change.
func checkInvestmentsFit(tx *sql.Tx, in details.Input) (string, error) {
	var total money.Cents
	if err := tx.QueryRow(
		"SELECT COALESCE(SUM(amount_cents), 0) FROM investment_contributions WHERE movement_sync_id = ?",
		in.SyncID,
	).Scan(&total); err != nil {
		return "", fmt.Errorf("sum the contributions of movement %s: %w", in.SyncID, err)
	}
	if total > in.AmountCents {
		return fmt.Sprintf("the instruments add up to %d cents, more than the movement's %d: send the detail again", total, in.AmountCents), nil
	}
	return "", nil
}

// applyInvestments makes the movement's contributions match the lines and
// records the values seen now.
func applyInvestments(tx *sql.Tx, in details.Input, inv details.Investments) error {
	want := make([]contribution, 0, len(inv.Lines))
	for _, l := range inv.Lines {
		want = append(want, contribution{l.InstrumentID, in.Month, l.AmountCents})
	}
	sort.Slice(want, func(i, j int) bool { return want[i].instrumentID < want[j].instrumentID })

	have, err := contributionsOf(tx, in.SyncID)
	if err != nil {
		return err
	}
	changed := false
	if !sameContributions(have, want) {
		if err := removeContributions(tx, in.SyncID); err != nil {
			return err
		}
		for _, c := range want {
			if _, err := tx.Exec(
				`INSERT INTO investment_contributions (instrument_id, month, amount_cents, movement_sync_id)
				 VALUES (?, ?, ?, ?)`,
				c.instrumentID, c.month, c.cents, in.SyncID,
			); err != nil {
				return fmt.Errorf("insert the contribution to instrument %d: %w", c.instrumentID, err)
			}
		}
		changed = true
	}

	for _, l := range inv.Lines {
		if l.ValueCents == nil {
			continue
		}
		var same int
		if err := tx.QueryRow(
			`SELECT COUNT(*) FROM investment_snapshots WHERE instrument_id = ? AND month = ? AND value_cents = ?`,
			l.InstrumentID, in.Month, *l.ValueCents,
		).Scan(&same); err != nil {
			return fmt.Errorf("find the snapshot of instrument %d: %w", l.InstrumentID, err)
		}
		if same == 0 {
			if _, err := tx.Exec(
				`INSERT INTO investment_snapshots (instrument_id, month, value_cents) VALUES (?, ?, ?)`,
				l.InstrumentID, in.Month, *l.ValueCents,
			); err != nil {
				return fmt.Errorf("insert the snapshot of instrument %d: %w", l.InstrumentID, err)
			}
			changed = true
		}
	}

	if changed {
		return csvstate.Raise(tx, modules.Investments)
	}
	return nil
}

func contributionsOf(tx *sql.Tx, syncID string) ([]contribution, error) {
	rows, err := tx.Query(
		`SELECT instrument_id, month, amount_cents FROM investment_contributions
		 WHERE movement_sync_id = ? ORDER BY instrument_id, id`,
		syncID,
	)
	if err != nil {
		return nil, fmt.Errorf("read the contributions of movement %s: %w", syncID, err)
	}
	defer rows.Close()
	var out []contribution
	for rows.Next() {
		var c contribution
		if err := rows.Scan(&c.instrumentID, &c.month, &c.cents); err != nil {
			return nil, fmt.Errorf("scan a contribution of movement %s: %w", syncID, err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func sameContributions(a, b []contribution) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func removeContributions(tx *sql.Tx, syncID string) error {
	if _, err := tx.Exec("DELETE FROM investment_contributions WHERE movement_sync_id = ?", syncID); err != nil {
		return fmt.Errorf("delete the contributions of movement %s: %w", syncID, err)
	}
	return nil
}

// followInvestments moves the contributions of a movement to its month. Their
// amounts stay, and checkInvestmentsFit has made sure they still fit.
func followInvestments(tx *sql.Tx, in details.Input) error {
	res, err := tx.Exec(
		`UPDATE investment_contributions SET month = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE movement_sync_id = ? AND month != ?`,
		in.Month, in.SyncID, in.Month,
	)
	if err != nil {
		return fmt.Errorf("move the contributions of movement %s: %w", in.SyncID, err)
	}
	return details.RaiseIfChanged(tx, res, modules.Investments)
}

func removeInvestments(tx *sql.Tx, syncID string) error {
	res, err := tx.Exec("DELETE FROM investment_contributions WHERE movement_sync_id = ?", syncID)
	if err != nil {
		return fmt.Errorf("delete the contributions of movement %s: %w", syncID, err)
	}
	return details.RaiseIfChanged(tx, res, modules.Investments)
}

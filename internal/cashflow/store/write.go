package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/cashflow"
)

// Writer changes movements inside one transaction. It is only handed out by
// Store.Write.
type Writer struct {
	tx      *sql.Tx
	changed bool
}

// Write runs fn in a transaction. If fn returns nil and made any change, the
// CSV revision is raised in the same transaction (see
// docs/modules/cashflow/csv.md), so the database never holds a change that the
// start-up check cannot see. If fn returns an error nothing is kept, so fn
// reports a problem with one movement by returning it only when the whole
// batch must be undone.
func (s *Store) Write(fn func(w *Writer) error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin cashflow write: %w", err)
	}
	defer tx.Rollback()

	w := &Writer{tx: tx}
	if err := fn(w); err != nil {
		return err
	}
	if w.changed {
		if _, err := tx.Exec(
			`INSERT INTO cashflow_csv_state (id, revision) VALUES (1, 1)
			 ON CONFLICT(id) DO UPDATE SET revision = revision + 1`,
		); err != nil {
			return fmt.Errorf("raise the cashflow csv revision: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit cashflow write: %w", err)
	}
	return nil
}

// Tx is the transaction of the write, for the detail of a movement that is
// written in the same transaction (see internal/details). A change made through
// it must raise its own CSV revision.
func (w *Writer) Tx() *sql.Tx { return w.tx }

// UpsertMovement adds the movement, or changes the one that has the same sync
// ID, and reports whether it was added. A movement whose values already match
// is left alone and counts as no change. It fails with ErrUnknownCategory for a
// category that does not exist and with ErrSyncIDTaken when the sync ID belongs
// to another person. The values are expected to be valid (see
// cashflow.MovementWrite.Validate); the database checks them again.
func (w *Writer) UpsertMovement(m cashflow.MovementWrite) (created bool, err error) {
	var categoryID int64
	err = w.tx.QueryRow("SELECT id FROM cashflow_categories WHERE name = ?", m.Category).Scan(&categoryID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("category %q: %w", m.Category, cashflow.ErrUnknownCategory)
	}
	if err != nil {
		return false, fmt.Errorf("find cashflow category %q: %w", m.Category, err)
	}

	ownerID, found, err := w.owner(m.SyncID)
	if err != nil {
		return false, err
	}
	if !found {
		if _, err := w.tx.Exec(
			`INSERT INTO cashflow_movements (sync_id, user_id, category_id, date, amount_cents, description)
			 VALUES (?, ?, ?, ?, ?, NULLIF(?, ''))`,
			m.SyncID, m.UserID, categoryID, m.Date, m.AmountCents, m.Description,
		); err != nil {
			return false, fmt.Errorf("insert cashflow movement %s: %w", m.SyncID, err)
		}
		w.changed = true
		return true, nil
	}
	if ownerID != m.UserID {
		return false, fmt.Errorf("movement %s: %w", m.SyncID, cashflow.ErrSyncIDTaken)
	}

	res, err := w.tx.Exec(
		`UPDATE cashflow_movements
		 SET category_id = ?, date = ?, amount_cents = ?, description = NULLIF(?, ''),
		     updated_at = CURRENT_TIMESTAMP
		 WHERE sync_id = ?
		   AND (category_id IS NOT ? OR date IS NOT ? OR amount_cents IS NOT ?
		        OR COALESCE(description, '') IS NOT ?)`,
		categoryID, m.Date, m.AmountCents, m.Description,
		m.SyncID,
		categoryID, m.Date, m.AmountCents, m.Description,
	)
	if err != nil {
		return false, fmt.Errorf("update cashflow movement %s: %w", m.SyncID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("count updated cashflow movements: %w", err)
	}
	if n > 0 {
		w.changed = true
	}
	return false, nil
}

// DeleteMovement removes the person's movement with this sync ID and reports
// whether there was one. Deleting a movement that does not exist is not an
// error, so a repeated delete is harmless. It fails with ErrSyncIDTaken when the
// sync ID belongs to another person.
func (w *Writer) DeleteMovement(userID int64, syncID string) (deleted bool, err error) {
	ownerID, found, err := w.owner(syncID)
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}
	if ownerID != userID {
		return false, fmt.Errorf("movement %s: %w", syncID, cashflow.ErrSyncIDTaken)
	}
	if _, err := w.tx.Exec("DELETE FROM cashflow_movements WHERE sync_id = ?", syncID); err != nil {
		return false, fmt.Errorf("delete cashflow movement %s: %w", syncID, err)
	}
	w.changed = true
	return true, nil
}

// owner returns the person a sync ID belongs to, and whether it exists.
func (w *Writer) owner(syncID string) (userID int64, found bool, err error) {
	err = w.tx.QueryRow("SELECT user_id FROM cashflow_movements WHERE sync_id = ?", syncID).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("find cashflow movement %s: %w", syncID, err)
	}
	return userID, true, nil
}

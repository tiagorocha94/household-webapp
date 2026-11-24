package cashflow

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/tiagorocha94/household-webapp/internal/csvdata"
)

// CSVMirror keeps the movements file up to date after devices change the
// database, so the file stays a faithful copy and a restart loses nothing. It
// is the only place that writes the file while the server runs, which makes it
// the one thing to remove when the file is retired.
type CSVMirror struct {
	db   *sql.DB
	path string

	// mu keeps two flushes from replacing the file at the same time, so an older
	// snapshot can never overwrite a newer one.
	mu sync.Mutex
}

// NewCSVMirror returns a mirror that writes cashflow_movements.csv in dir.
func NewCSVMirror(db *sql.DB, dir string) *CSVMirror {
	return &CSVMirror{db: db, path: filepath.Join(dir, movementsFile)}
}

// Flush writes every movement to the file and records which revision it holds.
// It does nothing when the file is already up to date. Changes that arrive
// while it runs are not lost: they raise the revision past the one recorded, so
// the file still counts as behind and the next flush writes them. If writing
// fails the file stays marked as behind, and so does the start-up check.
func (m *CSVMirror) Flush() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// One read transaction gives rows and revision from the same moment.
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("begin reading the movements: %w", err)
	}
	defer tx.Rollback()

	var revision, written int64
	err = tx.QueryRow(
		`SELECT COALESCE((SELECT revision FROM cashflow_csv_state WHERE id = 1), 0),
		        COALESCE((SELECT csv_revision FROM cashflow_csv_state WHERE id = 1), 0)`,
	).Scan(&revision, &written)
	if err != nil {
		return fmt.Errorf("read the csv revision: %w", err)
	}
	if revision == written {
		return nil
	}
	rows, err := exportMovements(tx)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("finish reading the movements: %w", err)
	}

	if err := csvdata.Write(m.path, rows); err != nil {
		return fmt.Errorf("write %s: %w", movementsFile, err)
	}
	// Only move the record forward: a change made since the read has a higher
	// revision and must stay pending.
	if _, err := m.db.Exec(
		`UPDATE cashflow_csv_state SET csv_revision = ? WHERE id = 1 AND csv_revision < ?`,
		revision, revision,
	); err != nil {
		return fmt.Errorf("record the revision written to %s: %w", movementsFile, err)
	}
	return nil
}

// Package csvstate keeps track of whether a module's CSV files are up to date
// with the database, for the modules that devices write to (ADR 021). A write
// raises the module's revision in its own transaction; a mirror rewrites the
// files and records the revision they hold. The cashflow module has its own
// version of this for its one file.
package csvstate

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/tiagorocha94/household-webapp/internal/csvdata"
	"github.com/tiagorocha94/household-webapp/internal/database"
)

// Exporter is a module that can write its files, such as a seed module.
type Exporter interface {
	Name() string
	Export(db *sql.DB) ([]csvdata.Table, error)
}

// Mirror rewrites a module's CSV files after devices change the database, so
// the files stay a faithful copy and a restart loses nothing.
type Mirror struct {
	db     *sql.DB
	dir    string
	module Exporter

	// mu keeps two flushes from replacing the files at the same time, so an
	// older snapshot can never overwrite a newer one.
	mu sync.Mutex
}

// Raise records that the module's database has a change its files do not have.
// Call it in the transaction that makes the change.
func Raise(q database.Querier, module string) error {
	if _, err := q.Exec(
		`INSERT INTO module_csv_state (module, revision) VALUES (?, 1)
		 ON CONFLICT(module) DO UPDATE SET revision = revision + 1`,
		module,
	); err != nil {
		return fmt.Errorf("raise the %s csv revision: %w", module, err)
	}
	return nil
}

// Behind reports whether a device changed the module's data after its files
// were last written, which makes the files older than the database. A module
// that has never been changed that way is not behind.
func Behind(q database.Querier, module string) (bool, error) {
	var behind bool
	err := q.QueryRow(
		`SELECT COALESCE((SELECT revision > csv_revision FROM module_csv_state WHERE module = ?), 0)`,
		module,
	).Scan(&behind)
	if err != nil {
		return false, fmt.Errorf("read whether the %s files are behind: %w", module, err)
	}
	return behind, nil
}

// MarkWritten records that the module's files now hold everything in the
// database. Call it after an export at start-up, before any device can change
// anything.
func MarkWritten(q database.Querier, module string) error {
	if _, err := q.Exec(
		`INSERT INTO module_csv_state (module, revision, csv_revision) VALUES (?, 0, 0)
		 ON CONFLICT(module) DO UPDATE SET csv_revision = revision`,
		module,
	); err != nil {
		return fmt.Errorf("record the %s files as up to date: %w", module, err)
	}
	return nil
}

// NewMirror returns a mirror that writes the module's files in dir.
func NewMirror(db *sql.DB, dir string, module Exporter) *Mirror {
	return &Mirror{db: db, dir: dir, module: module}
}

// Flush writes every file of the module and records which revision they hold.
// It does nothing when the files are already up to date. A change that arrives
// while it runs raises the revision past the one recorded, so the files still
// count as behind and the next flush writes them. If writing fails the files
// stay marked as behind, and so does the start-up check.
func (m *Mirror) Flush() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := m.module.Name()
	var revision, written int64
	err := m.db.QueryRow(
		`SELECT COALESCE((SELECT revision FROM module_csv_state WHERE module = ?), 0),
		        COALESCE((SELECT csv_revision FROM module_csv_state WHERE module = ?), 0)`,
		name, name,
	).Scan(&revision, &written)
	if err != nil {
		return fmt.Errorf("read the %s csv revision: %w", name, err)
	}
	if revision == written {
		return nil
	}

	// The revision was read before the data, so the files hold at least that
	// revision, possibly a later one.
	tables, err := m.module.Export(m.db)
	if err != nil {
		return fmt.Errorf("export %s: %w", name, err)
	}
	for _, t := range tables {
		if err := csvdata.Write(filepath.Join(m.dir, t.Name), t.Rows); err != nil {
			return fmt.Errorf("write %s: %w", t.Name, err)
		}
	}
	// Only move the record forward: a change made since the read has a higher
	// revision and must stay pending.
	if _, err := m.db.Exec(
		`UPDATE module_csv_state SET csv_revision = ? WHERE module = ? AND csv_revision < ?`,
		revision, name, revision,
	); err != nil {
		return fmt.Errorf("record the revision written for %s: %w", name, err)
	}
	return nil
}

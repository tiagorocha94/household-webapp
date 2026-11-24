// Package seed orchestrates the startup data flow: it reads each module's CSV
// files, hands them to the module to import, runs the modules' sync, and
// writes their exports back. The CSV files are the durable store. Migrations
// handle schema only. What each file means lives in the modules (see Module).
package seed

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/csvdata"
)

// Module is what a feature module provides to the seed flow. The modules
// satisfy it without importing seed.
type Module interface {
	// Name identifies the module in logs and errors.
	Name() string
	// Files lists the CSV files the module imports, with their headers.
	Files() []csvdata.File
	// Import mirrors the rows of the files that exist (keyed by file name)
	// into the database inside tx. The orchestrator commits or rolls back.
	Import(tx *sql.Tx, files map[string][][]string, log *slog.Logger) error
	// Sync does the module's automated maintenance as of now.
	Sync(db *sql.DB, now time.Time, log *slog.Logger) error
	// Export returns the module's current data as CSV tables, in write order.
	// A module that is not exported returns none.
	Export(db *sql.DB) ([]csvdata.Table, error)
}

// AfterExporter is optionally implemented by a module that needs to know its
// exported files were written. The runner calls AfterExport once all the
// module's files are written, never before, and an error aborts the export.
type AfterExporter interface {
	AfterExport(db *sql.DB) error
}

// Runner runs the seed flow over a list of modules, in the order given.
type Runner struct {
	log     *slog.Logger
	modules []Module
	now     func() time.Time
}

// NewRunner returns a Runner for modules that logs to log. The modules run one
// after another in the order given. They do not depend on each other's data,
// so the order only decides the order of logs and exported files.
func NewRunner(log *slog.Logger, modules ...Module) *Runner {
	return &Runner{log: log, modules: modules, now: time.Now}
}

// Import mirrors the CSV files in dir into the database. Files that are
// missing are skipped. Files that exist are authoritative: their rows are
// inserted or updated and rows absent from them are removed (users are never
// removed). Unchanged rows are left untouched, so importing the same files
// twice is a no-op. Each module is applied in a single transaction, and every
// file is read and checked before anything is written.
func (r *Runner) Import(db *sql.DB, dir string) error {
	r.log.Info("importing csv files", "dir", dir)

	type pendingModule struct {
		module Module
		files  map[string][][]string
	}
	var pending []pendingModule

	for _, m := range r.modules {
		files := make(map[string][][]string)
		for _, spec := range m.Files() {
			rows, err := csvdata.Read(filepath.Join(dir, spec.Name), spec.Header)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return fmt.Errorf("read %s: %w", spec.Name, err)
			}
			files[spec.Name] = rows
			r.log.Debug("read csv file", "file", spec.Name, "rows", len(rows))
		}
		if len(files) > 0 {
			pending = append(pending, pendingModule{m, files})
		}
	}

	for _, p := range pending {
		if err := r.applyModule(db, p.module, p.files); err != nil {
			return fmt.Errorf("import %s: %w", p.module.Name(), err)
		}
	}
	return nil
}

// Sync runs every module's automated maintenance as of the current time.
func (r *Runner) Sync(db *sql.DB) error {
	r.log.Info("running sync")

	now := r.now()
	for _, m := range r.modules {
		if err := m.Sync(db, now, r.log); err != nil {
			return fmt.Errorf("sync %s: %w", m.Name(), err)
		}
	}
	return nil
}

// Export writes current database data to CSV files in the given directory.
// Existing files are overwritten.
func (r *Runner) Export(db *sql.DB, dir string) error {
	r.log.Info("exporting csv files", "dir", dir)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	for _, m := range r.modules {
		tables, err := m.Export(db)
		if err != nil {
			return fmt.Errorf("export %s: %w", m.Name(), err)
		}
		for _, t := range tables {
			if err := csvdata.Write(filepath.Join(dir, t.Name), t.Rows); err != nil {
				return fmt.Errorf("write %s: %w", t.Name, err)
			}
			r.log.Debug("exported csv file", "file", t.Name, "rows", len(t.Rows)-1)
		}
		if after, ok := m.(AfterExporter); ok {
			if err := after.AfterExport(db); err != nil {
				return fmt.Errorf("after export %s: %w", m.Name(), err)
			}
		}
	}
	return nil
}

func (r *Runner) applyModule(db *sql.DB, m Module, files map[string][][]string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := m.Import(tx, files, r.log); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

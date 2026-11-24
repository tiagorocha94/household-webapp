package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tiagorocha94/household-webapp/internal/database"
	"github.com/tiagorocha94/household-webapp/internal/database/migrations"
	"github.com/tiagorocha94/household-webapp/internal/modules"
	"github.com/tiagorocha94/household-webapp/internal/seed"
)

// maxExampleRows limits how many added and removed rows the report lists for
// each table.
const maxExampleRows = 10

// fileChange describes a CSV file that startup would write.
type fileChange struct {
	name   string
	isNew  bool
	before int // data rows now on disk
	after  int // data rows startup would write
}

// startupPreview is what starting the server would change.
type startupPreview struct {
	migrations []string
	tables     []seed.TableChange
	files      []fileChange
}

// previewStartup writes to w what starting the server would change: pending
// migrations, the rows the import and sync would add or remove, and the CSV
// files the export would rewrite. Nothing is changed. The work runs on a copy
// of the database in a temporary directory, and the CSV files are only read.
func previewStartup(w io.Writer, dbPath, seedDir string, enabled modules.Set, logger *slog.Logger) error {
	tmp, err := os.MkdirTemp("", "household-dry-run-")
	if err != nil {
		return fmt.Errorf("create dry-run directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmp); err != nil {
			logger.Warn("remove dry-run files", "dir", tmp, "error", err)
		}
	}()

	preview, err := runPreview(filepath.Join(tmp, "preview.db"), filepath.Join(tmp, "export"), dbPath, seedDir, enabled, logger)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, preview.report())
	return err
}

func runPreview(copyPath, exportDir, dbPath, seedDir string, enabled modules.Set, logger *slog.Logger) (*startupPreview, error) {
	if err := copyDatabase(dbPath, copyPath); err != nil {
		return nil, err
	}
	db, err := database.Open(copyPath)
	if err != nil {
		return nil, fmt.Errorf("open database copy: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Warn("close database copy", "error", err)
		}
	}()

	applied, err := appliedMigrations(db)
	if err != nil {
		return nil, err
	}
	if err := migrations.Run(db, logger); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	current, err := appliedMigrations(db)
	if err != nil {
		return nil, err
	}
	pending := slices.DeleteFunc(current, func(name string) bool { return slices.Contains(applied, name) })

	before, err := seed.Capture(db)
	if err != nil {
		return nil, fmt.Errorf("capture data before import: %w", err)
	}
	runner := newSeedRunner(logger, enabled)
	if err := runner.Import(db, seedDir); err != nil {
		return nil, fmt.Errorf("import seed data: %w", err)
	}
	if err := runner.Sync(db); err != nil {
		return nil, fmt.Errorf("sync data: %w", err)
	}
	after, err := seed.Capture(db)
	if err != nil {
		return nil, fmt.Errorf("capture data after import: %w", err)
	}
	if err := runner.Export(db, exportDir); err != nil {
		return nil, fmt.Errorf("export seed data: %w", err)
	}
	files, err := changedFiles(seedDir, exportDir)
	if err != nil {
		return nil, err
	}
	return &startupPreview{migrations: pending, tables: before.Diff(after), files: files}, nil
}

// copyDatabase copies the database at src to dst. A database that does not
// exist yet is not copied: the preview then starts from an empty one, as the
// first start would.
func copyDatabase(src, dst string) error {
	if _, err := os.Stat(src); errors.Is(err, fs.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("check database %s: %w", src, err)
	}
	db, err := database.Open(src)
	if err != nil {
		return err
	}
	_, copyErr := db.Exec("VACUUM INTO ?", dst)
	closeErr := db.Close()
	if copyErr != nil {
		return fmt.Errorf("copy database: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close database %s: %w", src, closeErr)
	}
	return nil
}

// appliedMigrations lists the migrations the database has applied, none when
// it has never been migrated.
func appliedMigrations(db *sql.DB) ([]string, error) {
	var tables int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'schema_migrations'").Scan(&tables); err != nil {
		return nil, fmt.Errorf("check migrations table: %w", err)
	}
	if tables == 0 {
		return nil, nil
	}
	rows, err := db.Query("SELECT name FROM schema_migrations ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("list applied migrations: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan migration name: %w", err)
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// changedFiles compares the files the export wrote to exportDir with the ones
// in seedDir and returns those that would be created or rewritten.
func changedFiles(seedDir, exportDir string) ([]fileChange, error) {
	entries, err := os.ReadDir(exportDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list exported files: %w", err)
	}

	var changes []fileChange
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		next, err := os.ReadFile(filepath.Join(exportDir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read exported %s: %w", entry.Name(), err)
		}
		current, err := os.ReadFile(filepath.Join(seedDir, entry.Name()))
		isNew := errors.Is(err, fs.ErrNotExist)
		if err != nil && !isNew {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		if !isNew && bytes.Equal(current, next) {
			continue
		}
		changes = append(changes, fileChange{name: entry.Name(), isNew: isNew, before: dataRows(current), after: dataRows(next)})
	}
	return changes, nil
}

// dataRows counts the lines of a CSV file after its header.
func dataRows(content []byte) int {
	return max(bytes.Count(content, []byte("\n"))-1, 0)
}

// report renders the preview as text for a person to read.
func (p *startupPreview) report() string {
	var b strings.Builder
	b.WriteString("Dry run: nothing was written.\n")

	if len(p.migrations) == 0 && len(p.tables) == 0 && len(p.files) == 0 {
		b.WriteString("\nNothing would change.\n")
		return b.String()
	}

	if len(p.migrations) > 0 {
		b.WriteString("\nMigrations that would run:\n")
		for _, name := range p.migrations {
			fmt.Fprintf(&b, "  %s\n", name)
		}
	}
	if len(p.tables) > 0 {
		b.WriteString("\nDatabase changes from the CSV files and sync (+ added, - removed):\n")
		for _, change := range p.tables {
			fmt.Fprintf(&b, "  %s: +%d -%d\n", change.Table, len(change.Added), len(change.Removed))
			writeExamples(&b, "+", change.Added)
			writeExamples(&b, "-", change.Removed)
		}
	}
	if len(p.files) > 0 {
		b.WriteString("\nCSV files that would be written:\n")
		for _, f := range p.files {
			switch {
			case f.isNew:
				fmt.Fprintf(&b, "  %s: new, %d rows\n", f.name, f.after)
			case f.before == f.after:
				fmt.Fprintf(&b, "  %s: %d rows, rewritten (order or formatting)\n", f.name, f.after)
			default:
				fmt.Fprintf(&b, "  %s: %d -> %d rows\n", f.name, f.before, f.after)
			}
		}
	}
	return b.String()
}

func writeExamples(b *strings.Builder, sign string, rows []string) {
	for i, row := range rows {
		if i == maxExampleRows {
			fmt.Fprintf(b, "    ... and %d more\n", len(rows)-maxExampleRows)
			return
		}
		fmt.Fprintf(b, "    %s %s\n", sign, row)
	}
}

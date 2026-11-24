// Package csvdata holds the CSV reading, writing and database-mirroring helpers
// that the modules share when importing and exporting their files.
package csvdata

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/tiagorocha94/household-webapp/internal/money"
)

// File names a CSV file a module owns and the header it must start with.
type File struct {
	Name   string
	Header []string
}

// Table is one CSV file's worth of rows to write, header row included.
type Table struct {
	Name string
	Rows [][]string
}

// Read reads a CSV file and returns its data rows. The first record must
// equal expectedHeader. A header-only file returns no rows.
func Read(path string, expectedHeader []string) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	// Read-only: a failed close loses nothing.
	defer func() { _ = f.Close() }()

	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse CSV %s: %w", path, err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("CSV %s is missing its header", path)
	}
	if !slices.Equal(records[0], expectedHeader) {
		return nil, fmt.Errorf("CSV %s header = %q, want %q", path, records[0], expectedHeader)
	}
	if len(records) == 1 {
		return nil, nil
	}
	return records[1:], nil
}

// ParseNonNegativeCents parses raw integer cents. Signs, decimal separators,
// empty values and overflow are rejected.
func ParseNonNegativeCents(value, field string) (money.Cents, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("%s must be a non-negative integer", field)
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid %s %q: expected a non-negative integer", field, value)
		}
	}
	amount, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", field, value, err)
	}
	return money.Cents(amount), nil
}

// FormatCents formats cents as raw integer cents, the CSV representation.
func FormatCents(amount money.Cents) string {
	return strconv.FormatInt(int64(amount), 10)
}

// Write writes rows to path through a temporary file in the same directory
// and renames it into place, so a crash never leaves a truncated CSV. Close
// errors are checked explicitly.
func Write(path string, rows [][]string) error {
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary file for %s: %w", path, err)
	}
	tmp := f.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmp)
		}
	}()

	w := csv.NewWriter(f)
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			_ = f.Close()
			return fmt.Errorf("write row to %s: %w", path, err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		_ = f.Close()
		return fmt.Errorf("flush %s: %w", path, err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("sync %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}
	if err := os.Chmod(tmp, 0o644); err != nil {
		return fmt.Errorf("set permissions on %s: %w", path, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	committed = true
	return nil
}

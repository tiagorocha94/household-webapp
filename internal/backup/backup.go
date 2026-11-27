// Package backup writes a daily copy of the SQLite database.
//
// The copy is made with VACUUM INTO, which gives a consistent snapshot while
// the server keeps running. There is one file a day, named
// household-YYYY-MM-DD.db, and the oldest files beyond the number to keep are
// removed.
package backup

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"time"
)

const (
	dirMode  = 0o700
	fileMode = 0o600

	filePrefix = "household-"
	fileSuffix = ".db"
	dayLayout  = "2006-01-02"
	tempSuffix = ".tmp"
)

// fileName matches the files this package writes, and nothing else, so pruning
// can never remove a file that is not a backup.
var fileName = regexp.MustCompile(`^household-\d{4}-\d{2}-\d{2}\.db$`)

// Result says what a backup run did.
type Result struct {
	// File is the backup that was written.
	File string
	// Removed is how many older backups were deleted.
	Removed int
}

// Run copies the database into dir as the backup of the day of now, replacing
// an earlier backup of the same day, and keeps only the newest keep backups. The
// file is written under a temporary name and renamed, so a failed run never
// leaves a partial backup, and it is readable only by its owner.
func Run(ctx context.Context, db *sql.DB, dir string, keep int, now time.Time) (Result, error) {
	if keep < 1 {
		return Result{}, fmt.Errorf("keep %d must be at least 1", keep)
	}
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return Result{}, fmt.Errorf("create backup directory: %w", err)
	}

	final := filepath.Join(dir, filePrefix+now.Format(dayLayout)+fileSuffix)
	if err := write(ctx, db, final); err != nil {
		return Result{}, err
	}

	removed, err := prune(dir, keep)
	if err != nil {
		return Result{File: final, Removed: removed}, err
	}
	return Result{File: final, Removed: removed}, nil
}

// write makes the snapshot at final through a temporary file.
func write(ctx context.Context, db *sql.DB, final string) error {
	temp := final + tempSuffix
	// VACUUM INTO refuses an existing file, which here is only the leftover of a
	// run that was cut short.
	if err := os.Remove(temp); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove leftover backup file: %w", err)
	}
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", temp); err != nil {
		_ = os.Remove(temp)
		return fmt.Errorf("copy database: %w", err)
	}
	if err := os.Chmod(temp, fileMode); err != nil {
		_ = os.Remove(temp)
		return fmt.Errorf("restrict backup file: %w", err)
	}
	if err := os.Rename(temp, final); err != nil {
		_ = os.Remove(temp)
		return fmt.Errorf("move backup file into place: %w", err)
	}
	return nil
}

// prune removes the backups beyond the newest keep and returns how many it
// removed. The date in the name sorts the same as the date itself.
func prune(dir string, keep int) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("list backups: %w", err)
	}
	var files []string
	for _, e := range entries {
		if e.Type().IsRegular() && fileName.MatchString(e.Name()) {
			files = append(files, e.Name())
		}
	}
	slices.Sort(files)

	removed := 0
	for len(files)-removed > keep {
		if err := os.Remove(filepath.Join(dir, files[removed])); err != nil {
			return removed, fmt.Errorf("remove old backup: %w", err)
		}
		removed++
	}
	return removed, nil
}

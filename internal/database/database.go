// Package database handles SQLite connection setup and schema migration.
package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Querier is the subset of *sql.DB and *sql.Tx that modules use, so the same
// code runs inside or outside a transaction.
type Querier interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// Open creates a new SQLite connection to the database at the given path.
// It enables WAL mode and foreign keys for better concurrency and data integrity.
func Open(path string) (*sql.DB, error) {
	dsn := path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database at %s: %w", path, err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}

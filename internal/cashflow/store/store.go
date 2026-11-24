package store

import "database/sql"

// Store provides cashflow persistence.
type Store struct {
	db *sql.DB
}

type rowScanner interface {
	Scan(dest ...any) error
}

// New creates a Store backed by the given database connection.
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

package store

import (
	"database/sql"
)

// Store provides database operations for goal accounts, goals,
// and movements.
type Store struct {
	db *sql.DB
}

// New creates a new Store backed by the given database connection.
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

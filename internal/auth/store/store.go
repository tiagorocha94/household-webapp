package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/auth"
)

// Store provides database operations for user records.
type Store struct {
	db *sql.DB
}

// New creates a new Store backed by the given database connection.
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// GetByUsername retrieves a user by username. Returns nil if not found.
func (s *Store) GetByUsername(username string) (*auth.User, error) {
	row := s.db.QueryRow("SELECT id, username, password_hash FROM users WHERE username = ?", username)

	u := &auth.User{}
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query user %s: %w", username, err)
	}

	return u, nil
}

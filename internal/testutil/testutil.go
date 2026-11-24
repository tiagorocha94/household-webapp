// Package testutil provides database helpers shared by tests in several packages.
package testutil

import (
	"database/sql"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/database"
	"github.com/tiagorocha94/household-webapp/internal/database/migrations"
)

// PasswordHash is a bcrypt hash (of "secret", at the lowest cost) for tests that
// write a users.csv: the import accepts only hashes.
const PasswordHash = "$2b$04$8jlwRqkvlaWA2esicu4uHeEIf3YcMJD6DiXY3SMelllyChXBD9hZW"

// Logger returns a logger that discards everything, for tests that must pass
// one but do not look at the output.
func Logger() *slog.Logger { return slog.New(slog.DiscardHandler) }

// NewDB opens a temporary SQLite database with all migrations applied. It is
// closed when the test ends.
func NewDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})
	if err := migrations.Run(db, Logger()); err != nil {
		t.Fatalf("migrations.Run() error = %v", err)
	}
	return db
}

// MustExec runs a statement and returns the last inserted ID.
func MustExec(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	res, err := db.Exec(query, args...)
	if err != nil {
		t.Fatalf("exec failed: %v\nquery: %s", err, query)
	}
	id, _ := res.LastInsertId()
	return id
}

// InsertUser inserts a user with a placeholder password hash.
func InsertUser(t *testing.T, db *sql.DB, username string) int64 {
	t.Helper()
	return MustExec(t, db,
		"INSERT INTO users (username, password_hash) VALUES (?, 'x')", username)
}

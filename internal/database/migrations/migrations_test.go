package migrations

import (
	"database/sql"
	"log/slog"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/database"
)

func openUnmigrated(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "migrations.db"))
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	return db
}

func count(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return n
}

func TestMigrationFilesAreNumberedAndEmbedded(t *testing.T) {
	t.Parallel()

	entries, err := migrationFiles.ReadDir("sql")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no migrations are embedded")
	}
	name := regexp.MustCompile(`^\d{4}_[a-z0-9_]+\.sql$`)
	previous := ""
	for _, e := range entries {
		if !name.MatchString(e.Name()) {
			t.Errorf("migration %q does not match NNNN_name.sql", e.Name())
		}
		if e.Name() <= previous {
			t.Errorf("migration %q does not sort after %q", e.Name(), previous)
		}
		previous = e.Name()
	}
}

func TestRunAppliesEveryMigrationOnceAndIsRepeatable(t *testing.T) {
	t.Parallel()

	db := openUnmigrated(t)
	entries, err := migrationFiles.ReadDir("sql")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}

	for run := 1; run <= 2; run++ {
		if err := Run(db, slog.New(slog.DiscardHandler)); err != nil {
			t.Fatalf("Run() #%d error = %v", run, err)
		}
		if got := count(t, db, "SELECT COUNT(*) FROM schema_migrations"); got != len(entries) {
			t.Fatalf("after run %d: %d migrations recorded, want %d", run, got, len(entries))
		}
	}

	if got := count(t, db, "SELECT COUNT(*) FROM users"); got != 0 {
		t.Errorf("users = %d, want none: migrations create no users", got)
	}
	for _, table := range []string{"users", "goal_accounts", "goal_account_goals", "goal_account_movements", "credit_contracts", "credit_payments"} {
		if got := count(t, db, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table); got != 1 {
			t.Errorf("table %s exists %d times, want 1", table, got)
		}
	}
}

func TestFailedMigrationRollsBackAndIsNotRecorded(t *testing.T) {
	t.Parallel()

	db := openUnmigrated(t)
	if err := ensureMigrationsTable(db); err != nil {
		t.Fatalf("ensureMigrationsTable() error = %v", err)
	}

	err := applyMigration(db, "9999_bad.sql", "CREATE TABLE half_done (x INTEGER); INSERT INTO missing_table VALUES (1);")
	if err == nil {
		t.Fatal("applyMigration() succeeded, want an error")
	}
	if got := count(t, db, "SELECT COUNT(*) FROM sqlite_master WHERE name = 'half_done'"); got != 0 {
		t.Errorf("half_done tables = %d, want 0: the failed migration must roll back", got)
	}
	if got := count(t, db, "SELECT COUNT(*) FROM schema_migrations WHERE name = '9999_bad.sql'"); got != 0 {
		t.Errorf("recorded migrations = %d, want 0 for a failed one", got)
	}
}

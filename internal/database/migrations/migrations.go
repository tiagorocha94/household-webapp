// Package migrations handles automatic database schema migration on startup.
// SQL migration files are embedded in the binary and applied sequentially.
// Each migration is tracked in a schema_migrations table to avoid re-application.
package migrations

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"path"
	"sort"
	"strings"
)

//go:embed sql/*.sql
var migrationFiles embed.FS

// Run applies all pending migrations to the database. Migrations are read from
// the embedded sql/ directory and applied in lexicographic filename order.
// A schema_migrations table is used to track which migrations have been applied,
// and each one that is applied is reported to log.
func Run(db *sql.DB, log *slog.Logger) error {
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}

	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	entries, err := migrationFiles.ReadDir("sql")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		if applied[name] {
			continue
		}

		content, err := migrationFiles.ReadFile(path.Join("sql", name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		if err := applyMigration(db, name, string(content)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}

		log.Info("applied migration", "name", name)
	}

	return nil
}

func ensureMigrationsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name TEXT PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}
	return nil
}

func getAppliedMigrations(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("SELECT name FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan migration name: %w", err)
		}
		applied[name] = true
	}

	return applied, rows.Err()
}

func applyMigration(db *sql.DB, name, content string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	if _, err := tx.Exec(content); err != nil {
		tx.Rollback()
		return fmt.Errorf("execute migration SQL: %w", err)
	}

	if _, err := tx.Exec("INSERT INTO schema_migrations (name) VALUES (?)", name); err != nil {
		tx.Rollback()
		return fmt.Errorf("record migration: %w", err)
	}

	return tx.Commit()
}

package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/tiagorocha94/household-webapp/internal/csvdata"
)

// devicesHeader is the header of devices.csv: the enrolled phones, with the hash
// of their token and never the token itself.
var devicesHeader = []string{"username", "name", "token_hash", "enrolled_at"}

// SeedModule imports users.csv and devices.csv. Users are only added or have
// their password changed, never removed. The password column is a bcrypt hash,
// and users.csv is never rewritten. devices.csv is written on every start
// so that a rebuilt database keeps its enrolled phones. It satisfies the seed
// package's Module interface.
type SeedModule struct{}

// Name returns the module's name for logs and errors.
func (SeedModule) Name() string { return "users" }

// Files lists the CSV files the module imports.
func (SeedModule) Files() []csvdata.File {
	return []csvdata.File{
		{Name: "users.csv", Header: []string{"username", "password"}},
		{Name: "devices.csv", Header: devicesHeader},
	}
}

// Import adds the users in users.csv, updates changed passwords and restores the
// devices in devices.csv.
func (SeedModule) Import(tx *sql.Tx, files map[string][][]string, _ *slog.Logger) error {
	if err := applyUsers(tx, files); err != nil {
		return err
	}
	return restoreDevices(tx, files["devices.csv"])
}

// Sync does nothing: users need no maintenance.
func (SeedModule) Sync(*sql.DB, time.Time, *slog.Logger) error { return nil }

// Export returns devices.csv with the enrolled phones that are not revoked.
// users.csv is never rewritten.
func (SeedModule) Export(db *sql.DB) ([]csvdata.Table, error) {
	rows, err := db.Query(
		`SELECT u.username, d.name, d.token_hash, d.enrolled_at
		 FROM devices d
		 JOIN users u ON u.id = d.user_id
		 WHERE d.token_hash IS NOT NULL AND d.revoked_at IS NULL
		 ORDER BY u.username ASC`)
	if err != nil {
		return nil, fmt.Errorf("query devices for export: %w", err)
	}
	defer rows.Close()

	result := [][]string{devicesHeader}
	for rows.Next() {
		var username, name, tokenHash string
		var enrolledAt int64
		if err := rows.Scan(&username, &name, &tokenHash, &enrolledAt); err != nil {
			return nil, fmt.Errorf("scan device for export: %w", err)
		}
		result = append(result, []string{username, name, tokenHash, strconv.FormatInt(enrolledAt, 10)})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return []csvdata.Table{{Name: "devices.csv", Rows: result}}, nil
}

// restoreDevices adds the devices of devices.csv for the users that have no
// device row at all. A user who has one, enrolled, pending or revoked, is left
// alone, so a phone enrolled since the last export is kept and a revoked one
// does not come back. Only a rebuilt database has users without device rows.
func restoreDevices(tx *sql.Tx, rows [][]string) error {
	seen := make(map[string]bool, len(rows))
	for i, row := range rows {
		line := i + 2
		if len(row) < 4 {
			return fmt.Errorf("devices.csv row %d: expected 4 columns", line)
		}
		username, name := strings.TrimSpace(row[0]), strings.TrimSpace(row[1])
		tokenHash := strings.TrimSpace(row[2])
		enrolledAt, err := strconv.ParseInt(strings.TrimSpace(row[3]), 10, 64)
		if err != nil || enrolledAt <= 0 {
			return fmt.Errorf("devices.csv row %d: invalid enrolled_at %q: expected Unix seconds", line, row[3])
		}
		if name == "" || tokenHash == "" {
			return fmt.Errorf("devices.csv row %d: name and token_hash are required", line)
		}
		if seen[strings.ToLower(username)] {
			return fmt.Errorf("devices.csv row %d: %q has more than one device", line, username)
		}
		seen[strings.ToLower(username)] = true

		var userID int64
		err = tx.QueryRow("SELECT id FROM users WHERE username = ? COLLATE NOCASE", username).Scan(&userID)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("devices.csv row %d: unknown user %q", line, username)
		}
		if err != nil {
			return fmt.Errorf("devices.csv row %d: look up user %s: %w", line, username, err)
		}
		var has bool
		if err := tx.QueryRow("SELECT EXISTS (SELECT 1 FROM devices WHERE user_id = ?)", userID).Scan(&has); err != nil {
			return fmt.Errorf("devices.csv row %d: check devices of %s: %w", line, username, err)
		}
		if has {
			continue
		}
		if _, err := tx.Exec(
			`INSERT INTO devices (user_id, name, token_hash, created_at, enrolled_at)
			 VALUES (?, ?, ?, ?, ?)`,
			userID, name, tokenHash, enrolledAt, enrolledAt,
		); err != nil {
			return fmt.Errorf("devices.csv row %d: restore device of %s: %w", line, username, err)
		}
	}
	return nil
}

func applyUsers(tx *sql.Tx, files map[string][][]string) error {
	for i, row := range files["users.csv"] {
		if len(row) < 2 {
			return fmt.Errorf("users.csv row %d: expected 2 columns", i+2)
		}
		username := strings.TrimSpace(row[0])
		hash := strings.TrimSpace(row[1])
		if err := checkPasswordHash(hash); err != nil {
			return fmt.Errorf("users.csv row %d: %w", i+2, err)
		}

		var current string
		err := tx.QueryRow(
			"SELECT password_hash FROM users WHERE username = ? COLLATE NOCASE", username,
		).Scan(&current)

		switch {
		case errors.Is(err, sql.ErrNoRows):
			if _, err := tx.Exec(
				"INSERT INTO users (username, password_hash) VALUES (?, ?)",
				username, hash,
			); err != nil {
				return fmt.Errorf("insert user %s: %w", username, err)
			}
		case err != nil:
			return fmt.Errorf("look up user %s: %w", username, err)
		case current == hash:
			continue // same hash, nothing changed
		default:
			if _, err := tx.Exec(
				"UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE username = ? COLLATE NOCASE",
				hash, username,
			); err != nil {
				return fmt.Errorf("update user %s: %w", username, err)
			}
		}
	}
	return nil
}

// checkPasswordHash accepts only a valid bcrypt hash (it starts with $2a$, $2b$
// or $2y$). Anything else, a plain password included, is refused so that no
// password is ever kept in users.csv.
func checkPasswordHash(value string) error {
	if len(value) < 4 || value[0] != '$' || value[1] != '2' || value[3] != '$' || !strings.ContainsRune("aby", rune(value[2])) {
		return errors.New("the password must be a bcrypt hash (starting with $2a$, $2b$ or $2y$), not a plain password")
	}
	if _, err := bcrypt.Cost([]byte(value)); err != nil {
		return fmt.Errorf("the password looks like a bcrypt hash but is not valid: %w", err)
	}
	return nil
}

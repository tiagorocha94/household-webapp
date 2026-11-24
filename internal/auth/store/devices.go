package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/auth"
)

// deviceColumns are the columns scanDevice reads, from devices d joined to
// users u.
const deviceColumns = `d.id, d.user_id, u.username, d.name, d.created_at,
	d.enrol_expires_at, d.enrolled_at, d.last_seen_at, d.revoked_at`

type rowScanner interface{ Scan(dest ...any) error }

type queryRower interface {
	QueryRow(query string, args ...any) *sql.Row
}

func scanDevice(row rowScanner) (*auth.Device, error) {
	var (
		d                                    auth.Device
		created                              int64
		expires, enrolled, lastSeen, revoked sql.NullInt64
	)
	if err := row.Scan(&d.ID, &d.UserID, &d.Username, &d.Name, &created, &expires, &enrolled, &lastSeen, &revoked); err != nil {
		return nil, err
	}
	d.CreatedAt = time.Unix(created, 0)
	d.EnrolCodeExpiresAt = unixOrZero(expires)
	d.EnrolledAt = unixOrZero(enrolled)
	d.LastSeenAt = unixOrZero(lastSeen)
	d.RevokedAt = unixOrZero(revoked)
	return &d, nil
}

func unixOrZero(v sql.NullInt64) time.Time {
	if !v.Valid {
		return time.Time{}
	}
	return time.Unix(v.Int64, 0)
}

// CreateDevice stores a pending device for the user with the hash of its
// enrolment code, valid until expires. Codes nobody used in time are removed
// first, so they do not block the person. It returns auth.ErrUnknownUser,
// auth.ErrUserHasDevice or auth.ErrDeviceNameTaken when the device cannot be
// made.
func (s *Store) CreateDevice(username, name, codeHash string, now, expires time.Time) (*auth.Device, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin create device: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(
		"DELETE FROM devices WHERE token_hash IS NULL AND revoked_at IS NULL AND enrol_expires_at <= ?",
		now.Unix(),
	); err != nil {
		return nil, fmt.Errorf("delete expired enrolments: %w", err)
	}

	var userID int64
	err = tx.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, auth.ErrUnknownUser
	}
	if err != nil {
		return nil, fmt.Errorf("look up user %s: %w", username, err)
	}

	var taken bool
	if err := tx.QueryRow("SELECT EXISTS (SELECT 1 FROM devices WHERE user_id = ? AND revoked_at IS NULL)", userID).Scan(&taken); err != nil {
		return nil, fmt.Errorf("check devices of %s: %w", username, err)
	}
	if taken {
		return nil, auth.ErrUserHasDevice
	}
	if err := tx.QueryRow("SELECT EXISTS (SELECT 1 FROM devices WHERE name = ? AND revoked_at IS NULL)", name).Scan(&taken); err != nil {
		return nil, fmt.Errorf("check device name %s: %w", name, err)
	}
	if taken {
		return nil, auth.ErrDeviceNameTaken
	}

	res, err := tx.Exec(
		"INSERT INTO devices (user_id, name, enrol_code_hash, enrol_expires_at, created_at) VALUES (?, ?, ?, ?, ?)",
		userID, name, codeHash, expires.Unix(), now.Unix(),
	)
	if err != nil {
		return nil, fmt.Errorf("insert device %s: %w", name, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("read device id: %w", err)
	}
	d, err := scanDevice(tx.QueryRow("SELECT "+deviceColumns+" FROM devices d JOIN users u ON u.id = d.user_id WHERE d.id = ?", id))
	if err != nil {
		return nil, fmt.Errorf("read device %d: %w", id, err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit device %s: %w", name, err)
	}
	return d, nil
}

// EnrolDevice exchanges an unused, unexpired enrolment code for a token: it
// stores tokenHash, forgets the code, and returns the device and its user. A
// code that is wrong, expired, used or revoked returns auth.ErrInvalidEnrolCode.
// The exchange is one statement, so a code can only work once.
func (s *Store) EnrolDevice(codeHash, tokenHash string, now time.Time) (*auth.Device, *auth.User, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, nil, fmt.Errorf("begin enrol: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(
		`UPDATE devices SET token_hash = ?, enrol_code_hash = NULL, enrol_expires_at = NULL, enrolled_at = ?, last_seen_at = ?
		 WHERE enrol_code_hash = ? AND enrol_expires_at > ? AND revoked_at IS NULL`,
		tokenHash, now.Unix(), now.Unix(), codeHash, now.Unix(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("enrol device: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return nil, nil, fmt.Errorf("enrol device: %w", err)
	} else if n == 0 {
		return nil, nil, auth.ErrInvalidEnrolCode
	}

	d, u, err := deviceAndUser(tx, "d.token_hash = ?", tokenHash)
	if err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit enrol: %w", err)
	}
	return d, u, nil
}

// GetDeviceByToken returns the enrolled, not revoked device that owns the
// token hash and its user. It returns nil, nil for any other hash.
func (s *Store) GetDeviceByToken(tokenHash string) (*auth.Device, *auth.User, error) {
	d, u, err := deviceAndUser(s.db, "d.token_hash = ? AND d.revoked_at IS NULL", tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil
	}
	return d, u, err
}

// TouchDevice records that the device was used at now, unless it was already
// recorded at or after notBefore, which keeps busy devices from writing on
// every request.
func (s *Store) TouchDevice(id int64, now, notBefore time.Time) error {
	if _, err := s.db.Exec(
		"UPDATE devices SET last_seen_at = ? WHERE id = ? AND (last_seen_at IS NULL OR last_seen_at < ?)",
		now.Unix(), id, notBefore.Unix(),
	); err != nil {
		return fmt.Errorf("touch device %d: %w", id, err)
	}
	return nil
}

// ListDevices returns every device, oldest first, revoked ones included.
func (s *Store) ListDevices() ([]auth.Device, error) {
	rows, err := s.db.Query("SELECT " + deviceColumns + " FROM devices d JOIN users u ON u.id = d.user_id ORDER BY d.id")
	if err != nil {
		return nil, fmt.Errorf("query devices: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var devices []auth.Device
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}
		devices = append(devices, *d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read devices: %w", err)
	}
	return devices, nil
}

// RevokeDevice marks the device with this name revoked and removes its code
// and token, so neither works any more. It returns auth.ErrUnknownDevice when
// no device that is not revoked has the name.
func (s *Store) RevokeDevice(name string, now time.Time) error {
	res, err := s.db.Exec(
		`UPDATE devices SET revoked_at = ?, token_hash = NULL, enrol_code_hash = NULL, enrol_expires_at = NULL
		 WHERE name = ? AND revoked_at IS NULL`,
		now.Unix(), name,
	)
	if err != nil {
		return fmt.Errorf("revoke device %s: %w", name, err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("revoke device %s: %w", name, err)
	} else if n == 0 {
		return auth.ErrUnknownDevice
	}
	return nil
}

// deviceAndUser reads the one device matching where, and its user.
func deviceAndUser(q queryRower, where string, arg any) (*auth.Device, *auth.User, error) {
	var u auth.User
	var created int64
	var expires, enrolled, lastSeen, revoked sql.NullInt64
	d := &auth.Device{}
	err := q.QueryRow(
		`SELECT d.id, d.user_id, d.name, d.created_at, d.enrol_expires_at, d.enrolled_at, d.last_seen_at, d.revoked_at,
		        u.id, u.username, u.password_hash
		 FROM devices d JOIN users u ON u.id = d.user_id WHERE `+where, arg,
	).Scan(&d.ID, &d.UserID, &d.Name, &created, &expires, &enrolled, &lastSeen, &revoked, &u.ID, &u.Username, &u.PasswordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, err
		}
		return nil, nil, fmt.Errorf("read device: %w", err)
	}
	d.Username = u.Username
	d.CreatedAt = time.Unix(created, 0)
	d.EnrolCodeExpiresAt = unixOrZero(expires)
	d.EnrolledAt = unixOrZero(enrolled)
	d.LastSeenAt = unixOrZero(lastSeen)
	d.RevokedAt = unixOrZero(revoked)
	return d, &u, nil
}

package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/auth"
)

// CreateSession stores a session that is valid until expires. It takes the
// hash of the session token, never the token itself.
func (s *Store) CreateSession(tokenHash string, userID int64, expires time.Time) error {
	if _, err := s.db.Exec(
		"INSERT INTO user_sessions (token_hash, user_id, expires_at) VALUES (?, ?, ?)",
		tokenHash, userID, expires.Unix(),
	); err != nil {
		return fmt.Errorf("insert session for user %d: %w", userID, err)
	}
	return nil
}

// GetSessionUser returns the user of an unexpired session. Returns nil if the
// session is unknown or has expired as of now.
func (s *Store) GetSessionUser(tokenHash string, now time.Time) (*auth.User, error) {
	row := s.db.QueryRow(
		`SELECT u.id, u.username, u.password_hash
		 FROM user_sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = ? AND s.expires_at > ?`,
		tokenHash, now.Unix(),
	)

	u := &auth.User{}
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query session user: %w", err)
	}
	return u, nil
}

// DeleteSession removes a session. Removing one that does not exist is not an
// error.
func (s *Store) DeleteSession(tokenHash string) error {
	if _, err := s.db.Exec("DELETE FROM user_sessions WHERE token_hash = ?", tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteExpiredSessions removes the sessions that have expired as of now.
func (s *Store) DeleteExpiredSessions(now time.Time) error {
	if _, err := s.db.Exec("DELETE FROM user_sessions WHERE expires_at <= ?", now.Unix()); err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}
	return nil
}

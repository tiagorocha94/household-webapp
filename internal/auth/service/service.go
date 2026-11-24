package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/tiagorocha94/household-webapp/internal/auth"
	"github.com/tiagorocha94/household-webapp/internal/auth/store"
)

// SessionTTL is how long a session stays valid after login.
const SessionTTL = 7 * 24 * time.Hour

// ErrInvalidCredentials is returned by Authenticate when the username is
// unknown or the password does not match.
var ErrInvalidCredentials = errors.New("invalid credentials")

// dummyHash is the throwaway hash spendPasswordCheck compares against.
var (
	dummyHashOnce sync.Once
	dummyHash     []byte
)

// Service handles authentication logic including password verification
// and session management. Sessions are kept in the database, so they survive
// a restart.
type Service struct {
	store *store.Store
	now   func() time.Time
}

// New creates a new Service backed by the given Store.
func New(store *store.Store) *Service {
	return &Service{store: store, now: time.Now}
}

// Authenticate validates a username and password combination. On success it
// returns a session token and the user.
func (s *Service) Authenticate(username, password string) (string, *auth.User, error) {
	user, err := s.store.GetByUsername(username)
	if err != nil {
		return "", nil, fmt.Errorf("look up user: %w", err)
	}
	if user == nil {
		// Spend the same time as a wrong password, so the response time does
		// not show which usernames exist.
		spendPasswordCheck(password)
		return "", nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, ErrInvalidCredentials
	}

	token, err := s.createSession(user)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

// GetUserByToken retrieves the user associated with a session token.
// Returns nil if the token is unknown or expired.
func (s *Service) GetUserByToken(token string) (*auth.User, error) {
	user, err := s.store.GetSessionUser(hashToken(token), s.now())
	if err != nil {
		return nil, fmt.Errorf("get session user: %w", err)
	}
	return user, nil
}

// Logout removes the session associated with the given token.
func (s *Service) Logout(token string) error {
	if err := s.store.DeleteSession(hashToken(token)); err != nil {
		return fmt.Errorf("log out: %w", err)
	}
	return nil
}

// createSession stores a new session for user and returns its token. It also
// drops sessions that have already expired.
func (s *Service) createSession(user *auth.User) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}

	now := s.now()
	if err := s.store.DeleteExpiredSessions(now); err != nil {
		return "", err
	}
	if err := s.store.CreateSession(hashToken(token), user.ID, now.Add(SessionTTL)); err != nil {
		return "", err
	}
	return token, nil
}

// spendPasswordCheck compares password with a throwaway bcrypt hash of the
// same cost as the stored ones, and discards the result.
func spendPasswordCheck(password string) {
	dummyHashOnce.Do(func() {
		dummyHash, _ = bcrypt.GenerateFromPassword([]byte("not-a-real-password"), bcrypt.DefaultCost)
	})
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
}

// hashToken is what the database keeps of a session token.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

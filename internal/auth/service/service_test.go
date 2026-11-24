package service

import (
	"testing"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/auth"
	"github.com/tiagorocha94/household-webapp/internal/auth/store"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

// newTestService returns a Service on a fresh database with one user and a
// clock the test controls through the returned pointer.
func newTestService(t *testing.T) (*Service, *auth.User, *time.Time) {
	t.Helper()

	db := testutil.NewDB(t)
	id := testutil.InsertUser(t, db, "owner")
	svc := New(store.New(db))
	now := time.Now()
	svc.now = func() time.Time { return now }
	return svc, &auth.User{ID: id, Username: "owner"}, &now
}

func mustGetUser(t *testing.T, svc *Service, token string) *auth.User {
	t.Helper()
	user, err := svc.GetUserByToken(token)
	if err != nil {
		t.Fatalf("GetUserByToken() error = %v", err)
	}
	return user
}

func TestSessionsExpire(t *testing.T) {
	t.Parallel()

	svc, user, now := newTestService(t)
	token, err := svc.createSession(user)
	if err != nil {
		t.Fatalf("createSession() error = %v", err)
	}
	if got := mustGetUser(t, svc, token); got == nil || got.ID != user.ID {
		t.Fatalf("fresh session user = %v, want user %d", got, user.ID)
	}

	*now = now.Add(SessionTTL - time.Second)
	if mustGetUser(t, svc, token) == nil {
		t.Fatal("session expired too early")
	}

	*now = now.Add(2 * time.Second)
	if mustGetUser(t, svc, token) != nil {
		t.Fatal("expired session still valid")
	}
}

func TestSessionsSurviveANewService(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	id := testutil.InsertUser(t, db, "owner")
	token, err := New(store.New(db)).createSession(&auth.User{ID: id})
	if err != nil {
		t.Fatalf("createSession() error = %v", err)
	}

	// A second service on the same database stands in for a restart.
	if got := mustGetUser(t, New(store.New(db)), token); got == nil || got.ID != id {
		t.Fatalf("session user after restart = %v, want user %d", got, id)
	}
}

func TestLogoutEndsTheSession(t *testing.T) {
	t.Parallel()

	svc, user, _ := newTestService(t)
	token, err := svc.createSession(user)
	if err != nil {
		t.Fatalf("createSession() error = %v", err)
	}
	if err := svc.Logout(token); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if mustGetUser(t, svc, token) != nil {
		t.Fatal("session still valid after logout")
	}
	if err := svc.Logout(token); err != nil {
		t.Fatalf("second Logout() error = %v, want it to be idempotent", err)
	}
}

func TestUnknownTokenIsNotAUser(t *testing.T) {
	t.Parallel()

	svc, _, _ := newTestService(t)
	if mustGetUser(t, svc, "not-a-token") != nil {
		t.Fatal("unknown token found a user")
	}
}

func TestSessionsStoreOnlyAHashOfTheToken(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	id := testutil.InsertUser(t, db, "owner")
	token, err := New(store.New(db)).createSession(&auth.User{ID: id})
	if err != nil {
		t.Fatalf("createSession() error = %v", err)
	}

	var stored string
	if err := db.QueryRow("SELECT token_hash FROM user_sessions").Scan(&stored); err != nil {
		t.Fatalf("read stored session: %v", err)
	}
	if stored == token || stored != hashToken(token) {
		t.Fatalf("stored = %q, want the hash of the token and not the token", stored)
	}
}

func TestCreateSessionSweepsExpired(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	id := testutil.InsertUser(t, db, "owner")
	svc := New(store.New(db))
	now := time.Now()
	svc.now = func() time.Time { return now }
	user := &auth.User{ID: id}

	if _, err := svc.createSession(user); err != nil {
		t.Fatalf("createSession() error = %v", err)
	}
	now = now.Add(SessionTTL + time.Hour)
	if _, err := svc.createSession(user); err != nil {
		t.Fatalf("createSession() error = %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM user_sessions").Scan(&count); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if count != 1 {
		t.Fatalf("sessions = %d, want 1 after the sweep", count)
	}
}

func TestSpendPasswordCheckHasAUsableDummyHash(t *testing.T) {
	t.Parallel()

	spendPasswordCheck("anything")
	if len(dummyHash) == 0 {
		t.Fatal("dummy hash is empty, so the check would return before doing any work")
	}
}

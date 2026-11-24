package auth

import (
	"database/sql"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

func TestSeedModuleDeclaresUsersAndDevicesCSV(t *testing.T) {
	t.Parallel()

	m := SeedModule{}
	files := m.Files()
	if len(files) != 2 ||
		files[0].Name != "users.csv" || !reflect.DeepEqual(files[0].Header, []string{"username", "password"}) ||
		files[1].Name != "devices.csv" || !reflect.DeepEqual(files[1].Header, []string{"username", "name", "token_hash", "enrolled_at"}) {
		t.Errorf("Files() = %+v, want users.csv and devices.csv", files)
	}
	if err := m.Sync(nil, time.Now(), testutil.Logger()); err != nil {
		t.Errorf("Sync() error = %v, want nil", err)
	}
}

func TestDevicesSurviveARebuildOfTheDatabase(t *testing.T) {
	t.Parallel()

	old := testutil.NewDB(t)
	aliceID := testutil.InsertUser(t, old, "alice")
	bobID := testutil.InsertUser(t, old, "bob")
	testutil.InsertUser(t, old, "carol")
	testutil.MustExec(t, old,
		"INSERT INTO devices (user_id, name, token_hash, created_at, enrolled_at) VALUES (?, 'alice phone', 'hash-a', 100, 200)", aliceID)
	// A revoked device and a pending enrolment are not exported.
	testutil.MustExec(t, old,
		"INSERT INTO devices (user_id, name, created_at, revoked_at) VALUES (?, 'bob old phone', 100, 300)", bobID)

	tables, err := SeedModule{}.Export(old)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	want := [][]string{devicesHeader, {"alice", "alice phone", "hash-a", "200"}}
	if len(tables) != 1 || tables[0].Name != "devices.csv" || !reflect.DeepEqual(tables[0].Rows, want) {
		t.Fatalf("Export() = %+v, want devices.csv with %v", tables, want)
	}

	rebuilt := testutil.NewDB(t)
	testutil.InsertUser(t, rebuilt, "alice")
	testutil.InsertUser(t, rebuilt, "bob")
	if err := importDevices(t, rebuilt, tables[0].Rows[1:]); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	var name, hash string
	var enrolled int64
	if err := rebuilt.QueryRow(
		"SELECT d.name, d.token_hash, d.enrolled_at FROM devices d JOIN users u ON u.id = d.user_id WHERE u.username = 'alice'",
	).Scan(&name, &hash, &enrolled); err != nil {
		t.Fatalf("read restored device: %v", err)
	}
	if name != "alice phone" || hash != "hash-a" || enrolled != 200 {
		t.Errorf("restored device = %q %q %d, want alice phone hash-a 200", name, hash, enrolled)
	}
}

func TestRestoringDevicesLeavesUsersWithADeviceRowAlone(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	aliceID := testutil.InsertUser(t, db, "alice")
	// Alice's device was revoked since the file was written: it must not come back.
	testutil.MustExec(t, db,
		"INSERT INTO devices (user_id, name, created_at, revoked_at) VALUES (?, 'alice phone', 100, 300)", aliceID)

	if err := importDevices(t, db, [][]string{{"alice", "alice phone", "hash-a", "200"}}); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	var active int
	if err := db.QueryRow("SELECT COUNT(*) FROM devices WHERE revoked_at IS NULL").Scan(&active); err != nil {
		t.Fatalf("count devices: %v", err)
	}
	if active != 0 {
		t.Errorf("active devices = %d, want 0 (a revoked device stays revoked)", active)
	}
}

func TestRestoringDevicesRejectsBadRows(t *testing.T) {
	t.Parallel()

	for name, rows := range map[string][][]string{
		"unknown user":     {{"nobody", "phone", "hash", "200"}},
		"bad time":         {{"alice", "phone", "hash", "yesterday"}},
		"empty hash":       {{"alice", "phone", "", "200"}},
		"two for one user": {{"alice", "phone", "hash-1", "200"}, {"alice", "tablet", "hash-2", "201"}},
		"too few columns":  {{"alice", "phone"}},
		"zero time":        {{"alice", "phone", "hash", "0"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			db := testutil.NewDB(t)
			testutil.InsertUser(t, db, "alice")
			if err := importDevices(t, db, rows); err == nil {
				t.Errorf("Import() with %v error = nil, want a rejection", rows)
			}
		})
	}
}

// importDevices runs the module's import with only a devices.csv.
func importDevices(t *testing.T, db *sql.DB, rows [][]string) error {
	t.Helper()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := (SeedModule{}).Import(tx, map[string][][]string{"devices.csv": rows}, testutil.Logger()); err != nil {
		return err
	}
	return tx.Commit()
}

// importUsers runs the module's import with only a users.csv.
func importUsers(t *testing.T, db *sql.DB, rows [][]string) error {
	t.Helper()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := (SeedModule{}).Import(tx, map[string][][]string{"users.csv": rows}, testutil.Logger()); err != nil {
		return err
	}
	return tx.Commit()
}

func storedHash(t *testing.T, db *sql.DB, username string) string {
	t.Helper()
	var h string
	if err := db.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&h); err != nil {
		t.Fatalf("read hash of %s: %v", username, err)
	}
	return h
}

func TestUsersAreGivenAsBcryptHashes(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	// A "$2y$" hash, as `htpasswd -nbBC 10 "" secret` writes it.
	raw, err := bcrypt.GenerateFromPassword([]byte("from-hash"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	hash := "$2y$" + strings.TrimPrefix(string(raw), "$2a$")

	if err := importUsers(t, db, [][]string{{"hashed", hash}}); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if got := storedHash(t, db, "hashed"); got != hash {
		t.Errorf("stored hash = %q, want the hash from the CSV kept as it is", got)
	}
	if bcrypt.CompareHashAndPassword([]byte(storedHash(t, db, "hashed")), []byte("from-hash")) != nil {
		t.Error("the password behind the hash from the CSV does not log in")
	}

	// A new hash for an existing user replaces the old one; the same row twice is a no-op.
	other, err := bcrypt.GenerateFromPassword([]byte("other"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := importUsers(t, db, [][]string{{"hashed", string(other)}}); err != nil {
			t.Fatalf("Import() #%d error = %v", i+1, err)
		}
	}
	if got := storedHash(t, db, "hashed"); got != string(other) {
		t.Errorf("stored hash = %q, want the changed hash", got)
	}
}

func TestAPasswordThatIsNotABcryptHashIsRejected(t *testing.T) {
	t.Parallel()

	for name, value := range map[string]string{
		"plain password":         "from-plain",
		"empty":                  "",
		"too short to be a hash": "$2y$10$tooshort",
		"unknown version":        "$2x$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			db := testutil.NewDB(t)
			err := importUsers(t, db, [][]string{{"alex", value}})
			if err == nil || !strings.Contains(err.Error(), "users.csv row 2") {
				t.Fatalf("Import() error = %v, want a rejection naming users.csv row 2", err)
			}
			var n int
			if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&n); err != nil || n != 0 {
				t.Errorf("users after the rejection = %d, %v, want none", n, err)
			}
		})
	}
}

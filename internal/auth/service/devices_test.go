package service

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/auth"
	"github.com/tiagorocha94/household-webapp/internal/auth/store"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

// testingDB runs single-number queries against the test database.
type testingDB struct {
	t  *testing.T
	db *sql.DB
}

func (d *testingDB) count(query string, args ...any) int {
	d.t.Helper()
	var n int
	if err := d.db.QueryRow(query, args...).Scan(&n); err != nil {
		d.t.Fatalf("query %q: %v", query, err)
	}
	return n
}

// newDeviceService returns a Service with users "alice" and "bob" and a clock
// the test controls.
func newDeviceService(t *testing.T) (*Service, *testingDB, *time.Time) {
	t.Helper()

	db := testutil.NewDB(t)
	testutil.InsertUser(t, db, "alice")
	testutil.InsertUser(t, db, "bob")
	svc := New(store.New(db))
	now := time.Now().Truncate(time.Second)
	svc.now = func() time.Time { return now }
	return svc, &testingDB{t: t, db: db}, &now
}

func mustAdd(t *testing.T, svc *Service, username, name string) string {
	t.Helper()
	code, _, err := svc.AddDevice(username, name)
	if err != nil {
		t.Fatalf("AddDevice(%q, %q) error = %v", username, name, err)
	}
	return code
}

func TestEnrolGivesATokenThatIdentifiesTheDevice(t *testing.T) {
	t.Parallel()

	svc, db, _ := newDeviceService(t)
	code := mustAdd(t, svc, "alice", "Alice phone")
	if len(code) != enrolCodeLength+1 || code[enrolCodeLength/2] != '-' {
		t.Errorf("code = %q, want %d characters with a dash in the middle", code, enrolCodeLength)
	}

	// People type codes in lower case and without the dash.
	typed := strings.ToLower(strings.ReplaceAll(code, "-", " "))
	token, device, err := svc.Enrol(typed)
	if err != nil {
		t.Fatalf("Enrol(%q) error = %v", typed, err)
	}
	if device.Username != "alice" || device.Name != "Alice phone" {
		t.Errorf("enrolled device = %+v, want alice's Alice phone", device)
	}

	got, user, err := svc.DeviceByToken(token)
	if err != nil || got == nil || user == nil {
		t.Fatalf("DeviceByToken() = %v, %v, %v, want the device", got, user, err)
	}
	if got.ID != device.ID || user.Username != "alice" {
		t.Errorf("DeviceByToken() = device %d of %s, want %d of alice", got.ID, user.Username, device.ID)
	}

	// The database keeps hashes only: neither the code nor the token is in it.
	if n := db.count("SELECT COUNT(*) FROM devices WHERE token_hash = ? OR enrol_code_hash = ?", token, code); n != 0 {
		t.Errorf("%d rows hold the plain token or code, want 0", n)
	}
	if n := db.count("SELECT COUNT(*) FROM devices WHERE enrol_code_hash IS NOT NULL"); n != 0 {
		t.Errorf("%d rows still hold a code hash after enrolment, want 0", n)
	}
}

func TestEnrolCodeWorksOnce(t *testing.T) {
	t.Parallel()

	svc, _, _ := newDeviceService(t)
	code := mustAdd(t, svc, "alice", "phone")
	if _, _, err := svc.Enrol(code); err != nil {
		t.Fatalf("first Enrol() error = %v", err)
	}
	if _, _, err := svc.Enrol(code); !errors.Is(err, auth.ErrInvalidEnrolCode) {
		t.Errorf("second Enrol() error = %v, want ErrInvalidEnrolCode", err)
	}
}

func TestEnrolRejectsWrongAndExpiredCodes(t *testing.T) {
	t.Parallel()

	svc, _, now := newDeviceService(t)
	code := mustAdd(t, svc, "alice", "phone")

	for _, wrong := range []string{"", "AAAAA-AAAAA", strings.Repeat("Z", enrolCodeLength+5)} {
		if _, _, err := svc.Enrol(wrong); !errors.Is(err, auth.ErrInvalidEnrolCode) {
			t.Errorf("Enrol(%q) error = %v, want ErrInvalidEnrolCode", wrong, err)
		}
	}

	*now = now.Add(EnrolCodeTTL - time.Second)
	// A failed attempt must not use the code up.
	if _, _, err := svc.Enrol("AAAAA-AAAAA"); !errors.Is(err, auth.ErrInvalidEnrolCode) {
		t.Fatalf("wrong code error = %v", err)
	}
	*now = now.Add(time.Second)
	if _, _, err := svc.Enrol(code); !errors.Is(err, auth.ErrInvalidEnrolCode) {
		t.Errorf("Enrol() at expiry error = %v, want ErrInvalidEnrolCode", err)
	}
}

func TestEnrolWorksJustBeforeExpiry(t *testing.T) {
	t.Parallel()

	svc, _, now := newDeviceService(t)
	code := mustAdd(t, svc, "alice", "phone")
	*now = now.Add(EnrolCodeTTL - time.Second)
	if _, _, err := svc.Enrol(code); err != nil {
		t.Errorf("Enrol() one second before expiry error = %v, want success", err)
	}
}

func TestRevokedDeviceStopsWorking(t *testing.T) {
	t.Parallel()

	svc, _, _ := newDeviceService(t)
	token, _, err := svc.Enrol(mustAdd(t, svc, "alice", "phone"))
	if err != nil {
		t.Fatalf("Enrol() error = %v", err)
	}
	if err := svc.RevokeDevice("PHONE"); err != nil {
		t.Fatalf("RevokeDevice() error = %v (names are not case-sensitive)", err)
	}
	if got, _, err := svc.DeviceByToken(token); err != nil || got != nil {
		t.Errorf("DeviceByToken() after revoke = %v, %v, want nil", got, err)
	}
	if err := svc.RevokeDevice("phone"); !errors.Is(err, auth.ErrUnknownDevice) {
		t.Errorf("second RevokeDevice() error = %v, want ErrUnknownDevice", err)
	}

	// The person can be given a new device, even under the same name.
	if _, _, err := svc.Enrol(mustAdd(t, svc, "alice", "phone")); err != nil {
		t.Errorf("enrolling a replacement error = %v", err)
	}
	devices, err := svc.ListDevices()
	if err != nil || len(devices) != 2 {
		t.Fatalf("ListDevices() = %d devices, %v, want 2 (the revoked one stays listed)", len(devices), err)
	}
	if devices[0].Status(time.Now()) != "revoked" || devices[1].Status(time.Now()) != "enrolled" {
		t.Errorf("statuses = %s, %s, want revoked, enrolled", devices[0].Status(time.Now()), devices[1].Status(time.Now()))
	}
}

func TestRevokingAPendingDeviceKillsItsCode(t *testing.T) {
	t.Parallel()

	svc, _, _ := newDeviceService(t)
	code := mustAdd(t, svc, "alice", "phone")
	if err := svc.RevokeDevice("phone"); err != nil {
		t.Fatalf("RevokeDevice() error = %v", err)
	}
	if _, _, err := svc.Enrol(code); !errors.Is(err, auth.ErrInvalidEnrolCode) {
		t.Errorf("Enrol() with a revoked code error = %v, want ErrInvalidEnrolCode", err)
	}
}

func TestEachPersonHasOneActiveDevice(t *testing.T) {
	t.Parallel()

	svc, _, now := newDeviceService(t)
	mustAdd(t, svc, "alice", "phone")

	if _, _, err := svc.AddDevice("alice", "tablet"); !errors.Is(err, auth.ErrUserHasDevice) {
		t.Errorf("second device for alice error = %v, want ErrUserHasDevice", err)
	}
	if _, _, err := svc.AddDevice("bob", "PHONE"); !errors.Is(err, auth.ErrDeviceNameTaken) {
		t.Errorf("same name for bob error = %v, want ErrDeviceNameTaken", err)
	}
	if _, _, err := svc.AddDevice("nobody", "x"); !errors.Is(err, auth.ErrUnknownUser) {
		t.Errorf("unknown user error = %v, want ErrUnknownUser", err)
	}

	// A code nobody used does not block the person for ever.
	*now = now.Add(EnrolCodeTTL)
	if _, _, err := svc.AddDevice("alice", "tablet"); err != nil {
		t.Errorf("AddDevice() after the old code expired error = %v", err)
	}
	if devices, _ := svc.ListDevices(); len(devices) != 1 || devices[0].Name != "tablet" {
		t.Errorf("devices = %+v, want only tablet", devices)
	}
}

func TestAddDeviceChecksTheName(t *testing.T) {
	t.Parallel()

	svc, _, _ := newDeviceService(t)
	for _, name := range []string{"", "   ", "-phone", strings.Repeat("a", 41), "pho\nne", "pho/ne"} {
		if _, _, err := svc.AddDevice("alice", name); !errors.Is(err, apperr.ErrBadRequest) {
			t.Errorf("AddDevice(%q) error = %v, want a bad-request error", name, err)
		}
	}
	if _, _, err := svc.AddDevice("alice", "  Alice phone 2  "); err != nil {
		t.Errorf("AddDevice() with a padded name error = %v", err)
	}
}

func TestLastSeenIsRecordedAtMostOncePerMinute(t *testing.T) {
	t.Parallel()

	svc, db, now := newDeviceService(t)
	token, _, err := svc.Enrol(mustAdd(t, svc, "alice", "phone"))
	if err != nil {
		t.Fatalf("Enrol() error = %v", err)
	}
	enrolledAt := now.Unix()

	*now = now.Add(30 * time.Second)
	if _, _, err := svc.DeviceByToken(token); err != nil {
		t.Fatalf("DeviceByToken() error = %v", err)
	}
	if got := db.count("SELECT last_seen_at FROM devices"); got != int(enrolledAt) {
		t.Errorf("last_seen_at after 30s = %d, want unchanged %d", got, enrolledAt)
	}

	*now = now.Add(31 * time.Second)
	if _, _, err := svc.DeviceByToken(token); err != nil {
		t.Fatalf("DeviceByToken() error = %v", err)
	}
	if got := db.count("SELECT last_seen_at FROM devices"); got != int(now.Unix()) {
		t.Errorf("last_seen_at after 61s = %d, want %d", got, now.Unix())
	}
}

func TestDeviceStatus(t *testing.T) {
	t.Parallel()

	now := time.Now()
	tests := []struct {
		name string
		d    auth.Device
		want string
	}{
		{"waiting", auth.Device{EnrolCodeExpiresAt: now.Add(time.Minute)}, "waiting"},
		{"expired", auth.Device{EnrolCodeExpiresAt: now}, "expired"},
		{"enrolled", auth.Device{EnrolledAt: now.Add(-time.Hour)}, "enrolled"},
		{"revoked beats enrolled", auth.Device{EnrolledAt: now.Add(-time.Hour), RevokedAt: now}, "revoked"},
	}
	for _, tt := range tests {
		if got := tt.d.Status(now); got != tt.want {
			t.Errorf("%s: Status() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

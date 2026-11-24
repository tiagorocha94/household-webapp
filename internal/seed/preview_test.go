package seed

import (
	"reflect"
	"strings"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

func TestSnapshotDiffComparesRowsAsMultisets(t *testing.T) {
	t.Parallel()

	before := Snapshot{"a": {"x", "y", "y"}, "b": {"k"}, "gone": {"z"}}
	after := Snapshot{"a": {"x", "y", "w"}, "b": {"k"}, "new": {"n"}}

	got := before.Diff(after)
	want := []TableChange{
		{Table: "a", Added: []string{"w"}, Removed: []string{"y"}},
		{Table: "gone", Removed: []string{"z"}},
		{Table: "new", Added: []string{"n"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Diff() = %#v, want %#v", got, want)
	}
}

func TestSnapshotDiffOfEqualSnapshotsIsEmpty(t *testing.T) {
	t.Parallel()

	s := Snapshot{"a": {"x", "y"}, "b": nil}
	if got := s.Diff(Snapshot{"a": {"x", "y"}, "b": nil}); len(got) != 0 {
		t.Fatalf("Diff() = %#v, want no changes", got)
	}
}

func TestCaptureLeavesOutIDsAndMasksPasswordHashes(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	testutil.InsertUser(t, db, "ana")

	snapshot, err := Capture(db)
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if _, ok := snapshot["schema_migrations"]; ok {
		t.Error("Capture() includes schema_migrations, want data tables only")
	}
	users := snapshot["users"]
	if len(users) != 1 || !strings.HasPrefix(users[0], "ana | #") || len(users[0]) != len("ana | #")+8 {
		t.Fatalf("users = %q, want one row \"ana | #<fingerprint>\"", users)
	}
}

func TestCaptureShowsRowsAddedLater(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	before, err := Capture(db)
	if err != nil {
		t.Fatalf("Capture() before error = %v", err)
	}
	testutil.MustExec(t, db, "INSERT INTO goal_accounts (name, monthly_deposit_cents) VALUES ('Household', 100)")
	after, err := Capture(db)
	if err != nil {
		t.Fatalf("Capture() after error = %v", err)
	}

	got := before.Diff(after)
	want := []TableChange{{Table: "goal_accounts", Added: []string{"Household | 100"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Diff() = %#v, want %#v", got, want)
	}
}

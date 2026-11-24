package csvdata

import (
	"database/sql"
	"reflect"
	"slices"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

func usernames(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query("SELECT username FROM users ORDER BY username")
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			t.Fatalf("scan user: %v", err)
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("list users: %v", err)
	}
	return out
}

func scanUser(rows *sql.Rows) (int64, string, error) {
	var id int64
	var name string
	err := rows.Scan(&id, &name)
	return id, name, err
}

func TestQueryInt64s(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	a := testutil.InsertUser(t, db, "a")
	b := testutil.InsertUser(t, db, "b")

	got, err := QueryInt64s(db, "SELECT id FROM users ORDER BY id")
	if err != nil || !slices.Equal(got, []int64{a, b}) {
		t.Fatalf("QueryInt64s() = %v, %v, want [%d %d]", got, err, a, b)
	}

	none, err := QueryInt64s(db, "SELECT id FROM users WHERE 1 = 0")
	if err != nil || len(none) != 0 {
		t.Fatalf("QueryInt64s(no rows) = %v, %v, want empty", none, err)
	}
	if _, err := QueryInt64s(db, "SELECT nope FROM nowhere"); err == nil {
		t.Error("QueryInt64s(bad SQL) succeeded, want an error")
	}
}

func TestDeleteUnlistedKeepsListedRowsOnly(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	for _, u := range []string{"keep", "drop1", "drop2"} {
		testutil.InsertUser(t, db, u)
	}

	err := DeleteUnlisted(testutil.Logger(), db, "users", "SELECT id, username FROM users",
		map[string]bool{"keep": true}, scanUser)
	if err != nil {
		t.Fatalf("DeleteUnlisted() error = %v", err)
	}
	if got := usernames(t, db); !reflect.DeepEqual(got, []string{"keep"}) {
		t.Errorf("users = %v, want [keep]", got)
	}
	// Nothing to delete is not an error.
	if err := DeleteUnlisted(testutil.Logger(), db, "users", "SELECT id, username FROM users",
		map[string]bool{"keep": true}, scanUser); err != nil {
		t.Errorf("second DeleteUnlisted() error = %v", err)
	}
}

func TestClearIfChangedReplacesOnlyWhenKeysDiffer(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	if _, err := db.Exec("CREATE TABLE child (id INTEGER PRIMARY KEY, parent INTEGER, k TEXT)"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	for _, r := range []struct {
		parent int
		k      string
	}{{1, "a"}, {1, "a"}, {1, "b"}, {2, "c"}} {
		if _, err := db.Exec("INSERT INTO child (parent, k) VALUES (?, ?)", r.parent, r.k); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	scan := func(rs *sql.Rows) (string, error) {
		var k string
		err := rs.Scan(&k)
		return k, err
	}
	const query = "SELECT k FROM child WHERE parent = ?"
	count := func() int {
		var n int
		if err := db.QueryRow("SELECT COUNT(*) FROM child").Scan(&n); err != nil {
			t.Fatalf("count: %v", err)
		}
		return n
	}

	// The same keys in another order, with the same repeats: nothing changes.
	changed, err := ClearIfChanged(db, "child", "parent", 1, query, scan, []string{"b", "a", "a"})
	if err != nil || changed || count() != 4 {
		t.Fatalf("same keys: changed=%v err=%v rows=%d, want false, nil, 4", changed, err, count())
	}
	// A different number of repeats is a change: the parent's rows go, the
	// other parent's stay.
	changed, err = ClearIfChanged(db, "child", "parent", 1, query, scan, []string{"a", "b"})
	if err != nil || !changed || count() != 1 {
		t.Fatalf("different keys: changed=%v err=%v rows=%d, want true, nil, 1", changed, err, count())
	}
	if _, err := ClearIfChanged(db, "child", "parent", 1, "SELECT nope FROM nowhere WHERE ? = 1", scan, nil); err == nil {
		t.Error("ClearIfChanged(bad SQL) succeeded, want an error")
	}
}

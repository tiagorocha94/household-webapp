package service

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/cashflow"
	"github.com/tiagorocha94/household-webapp/internal/cashflow/store"
	"github.com/tiagorocha94/household-webapp/internal/csvdata"
	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

var movementsHeader = []string{"user", "category", "date", "amount_cents", "description", "sync_id"}

type syncFixture struct {
	svc            *Service
	db             *sql.DB
	dir            string
	aliceID, bobID int64
}

type failingMirror struct{ calls int }

func sid(n int) string { return fmt.Sprintf("00000000-0000-4000-8000-%012d", n) }

// newSyncFixture returns a service wired to the real CSV mirror, over a
// database with the users alice and bob and two categories.
func newSyncFixture(t *testing.T) *syncFixture {
	t.Helper()

	db := testutil.NewDB(t)
	f := &syncFixture{
		db:      db,
		dir:     t.TempDir(),
		aliceID: testutil.InsertUser(t, db, "alice"),
		bobID:   testutil.InsertUser(t, db, "bob"),
	}
	testutil.MustExec(t, db, `INSERT INTO cashflow_categories (name, direction, icon, color) VALUES ('salary', 'income', 'briefcase', '#16A34A')`)
	testutil.MustExec(t, db, `INSERT INTO cashflow_categories (name, direction, icon, color) VALUES ('food', 'expense', 'utensils', '#E76F51')`)
	f.svc = New(store.New(db))
	f.svc.EnableMirror(cashflow.NewCSVMirror(db, f.dir), testutil.Logger())
	return f
}

func food(n int, date string, cents int64, note string) SyncOperation {
	return SyncOperation{Op: OpUpsert, SyncID: sid(n), Category: "food", Date: date, AmountCents: money.Cents(cents), Description: note}
}

func (f *syncFixture) apply(t *testing.T, userID int64, ops ...SyncOperation) []SyncResult {
	t.Helper()
	results, err := f.svc.ApplySync(userID, ops)
	if err != nil {
		t.Fatalf("ApplySync() error = %v", err)
	}
	if len(results) != len(ops) {
		t.Fatalf("ApplySync() gave %d results for %d operations", len(results), len(ops))
	}
	return results
}

func (f *syncFixture) requireOK(t *testing.T, results []SyncResult) {
	t.Helper()
	for i, r := range results {
		if r.Rejected != "" {
			t.Fatalf("result %d (%s) rejected: %s", i, r.SyncID, r.Rejected)
		}
	}
}

// rows returns the movements in the database as user|category|date|cents|note|id.
func (f *syncFixture) rows(t *testing.T) []string {
	t.Helper()
	r, err := f.db.Query(
		`SELECT u.username, c.name, m.date, m.amount_cents, COALESCE(m.description, ''), m.sync_id
		 FROM cashflow_movements m JOIN users u ON u.id = m.user_id JOIN cashflow_categories c ON c.id = m.category_id
		 ORDER BY m.sync_id`)
	if err != nil {
		t.Fatalf("query movements: %v", err)
	}
	defer r.Close()
	var out []string
	for r.Next() {
		var user, cat, date, note, id string
		var cents int64
		if err := r.Scan(&user, &cat, &date, &cents, &note, &id); err != nil {
			t.Fatalf("scan movement: %v", err)
		}
		out = append(out, strings.Join([]string{user, cat, date, fmt.Sprint(cents), note, id}, "|"))
	}
	return out
}

// csvRows returns the movements file as the same strings, ordered by sync ID.
func (f *syncFixture) csvRows(t *testing.T) []string {
	t.Helper()
	rows, err := csvdata.Read(filepath.Join(f.dir, "cashflow_movements.csv"), movementsHeader)
	if err != nil {
		t.Fatalf("read movements file: %v", err)
	}
	var out []string
	for _, row := range rows {
		out = append(out, strings.Join([]string{row[0], row[1], row[2], row[3], row[4], row[5]}, "|"))
	}
	return sortedCopy(out)
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Slice(out, func(i, j int) bool { return syncIDOf(out[i]) < syncIDOf(out[j]) })
	return out
}

func syncIDOf(row string) string { return row[strings.LastIndex(row, "|")+1:] }

// requireFileMatchesDatabase is the check made after every case: the file holds
// exactly what the database holds, and is marked as up to date.
func (f *syncFixture) requireFileMatchesDatabase(t *testing.T) {
	t.Helper()
	if db, file := f.rows(t), f.csvRows(t); !reflect.DeepEqual(db, file) {
		t.Errorf("movements file = %v, database = %v, want them equal", file, db)
	}
	var revision, written int
	if err := f.db.QueryRow("SELECT revision, csv_revision FROM cashflow_csv_state WHERE id = 1").Scan(&revision, &written); err != nil {
		t.Fatalf("read csv state: %v", err)
	}
	if revision != written {
		t.Errorf("csv revision = %d, file holds %d, want them equal", revision, written)
	}
}

func (f *syncFixture) revision(t *testing.T) int {
	t.Helper()
	var revision int
	err := f.db.QueryRow("SELECT COALESCE((SELECT revision FROM cashflow_csv_state WHERE id = 1), 0)").Scan(&revision)
	if err != nil {
		t.Fatalf("read revision: %v", err)
	}
	return revision
}

func TestApplySyncAddsEditsAndDeletes(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)

	f.requireOK(t, f.apply(t, f.aliceID, food(1, "2026-03-05", 4000, "  lunch  ")))
	want := []string{"alice|food|2026-03-05|4000|lunch|" + sid(1)}
	if got := f.rows(t); !reflect.DeepEqual(got, want) {
		t.Fatalf("after add: rows = %v, want %v (the note is trimmed)", got, want)
	}
	f.requireFileMatchesDatabase(t)

	// An edit after the add: the same sync ID with other values.
	f.requireOK(t, f.apply(t, f.aliceID, food(1, "2026-03-06", 4500, "")))
	want = []string{"alice|food|2026-03-06|4500||" + sid(1)}
	if got := f.rows(t); !reflect.DeepEqual(got, want) {
		t.Fatalf("after edit: rows = %v, want %v", got, want)
	}
	f.requireFileMatchesDatabase(t)

	f.requireOK(t, f.apply(t, f.aliceID, SyncOperation{Op: OpDelete, SyncID: sid(1)}))
	if got := f.rows(t); len(got) != 0 {
		t.Fatalf("after delete: rows = %v, want none", got)
	}
	f.requireFileMatchesDatabase(t)
}

func TestApplySyncIsIdempotent(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	batch := []SyncOperation{
		food(1, "2026-03-05", 4000, "lunch"),
		food(2, "2026-03-05", 1500, ""),
		{Op: OpDelete, SyncID: sid(3)},
	}
	f.requireOK(t, f.apply(t, f.aliceID, batch...))
	rows, revision := f.rows(t), f.revision(t)

	// A retry after a lost answer changes nothing, not even the revision.
	f.requireOK(t, f.apply(t, f.aliceID, batch...))
	if got := f.rows(t); !reflect.DeepEqual(got, rows) {
		t.Errorf("rows after retry = %v, want %v", got, rows)
	}
	if got := f.revision(t); got != revision {
		t.Errorf("revision after retry = %d, want unchanged %d", got, revision)
	}
	f.requireFileMatchesDatabase(t)
}

func TestApplySyncDeletingAnUnknownMovementIsFine(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	f.requireOK(t, f.apply(t, f.aliceID, SyncOperation{Op: OpDelete, SyncID: sid(9)}))
	if got := f.revision(t); got != 0 {
		t.Errorf("revision = %d, want 0: nothing changed", got)
	}
}

func TestApplySyncOneRejectionDoesNotStopTheRest(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	// Bob owns movement 50, which alice must not be able to touch.
	f.requireOK(t, f.apply(t, f.bobID, food(50, "2026-03-01", 700, "bob's")))

	bad := func(mutate func(*SyncOperation)) SyncOperation {
		op := food(2, "2026-03-05", 100, "")
		mutate(&op)
		return op
	}
	results := f.apply(t, f.aliceID,
		food(1, "2026-03-05", 4000, "ok"),
		bad(func(o *SyncOperation) { o.Category = "lottery" }),
		bad(func(o *SyncOperation) { o.Date = "2026-02-30" }),
		bad(func(o *SyncOperation) { o.AmountCents = 0 }),
		bad(func(o *SyncOperation) { o.SyncID = "NOT-A-UUID" }),
		bad(func(o *SyncOperation) { o.Op = "merge" }),
		food(50, "2026-03-05", 1, "takeover"),
		SyncOperation{Op: OpDelete, SyncID: sid(50)},
		SyncOperation{Op: OpDelete, SyncID: "x"},
		food(3, "2026-03-07", 300, "also ok"),
	)

	wantRejected := []bool{false, true, true, true, true, true, true, true, true, false}
	for i, r := range results {
		if (r.Rejected != "") != wantRejected[i] {
			t.Errorf("result %d (%s) rejected = %q, want rejected = %v", i, r.SyncID, r.Rejected, wantRejected[i])
		}
	}
	for _, i := range []int{1, 2, 3, 4, 5, 6, 7, 8} {
		if strings.Contains(results[i].Rejected, "bad request") {
			t.Errorf("result %d reason = %q, want no internal marker", i, results[i].Rejected)
		}
	}
	if !strings.Contains(results[1].Rejected, "lottery") || !strings.Contains(results[2].Rejected, "date") ||
		!strings.Contains(results[6].Rejected, "another person") {
		t.Errorf("reasons do not say what is wrong: %q, %q, %q", results[1].Rejected, results[2].Rejected, results[6].Rejected)
	}

	want := []string{
		"alice|food|2026-03-05|4000|ok|" + sid(1),
		"alice|food|2026-03-07|300|also ok|" + sid(3),
		"bob|food|2026-03-01|700|bob's|" + sid(50),
	}
	if got := f.rows(t); !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %v, want %v (bob's movement untouched)", got, want)
	}
	f.requireFileMatchesDatabase(t)
}

func TestApplySyncLastOperationOnASyncIDWins(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	f.requireOK(t, f.apply(t, f.aliceID, food(1, "2026-03-05", 100, "first"), food(1, "2026-03-05", 200, "second")))
	if got, want := f.rows(t), []string{"alice|food|2026-03-05|200|second|" + sid(1)}; !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %v, want %v", got, want)
	}
	f.requireFileMatchesDatabase(t)
}

func TestApplySyncRefusesTooLargeABatch(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	ops := make([]SyncOperation, 0, MaxSyncOperations+1)
	for i := range MaxSyncOperations + 1 {
		ops = append(ops, food(i+1, "2026-03-05", 100, ""))
	}

	if _, err := f.svc.ApplySync(f.aliceID, ops); !errors.Is(err, apperr.ErrBadRequest) {
		t.Errorf("ApplySync(%d operations) error = %v, want a bad-request error", len(ops), err)
	}
	if got := f.rows(t); len(got) != 0 {
		t.Errorf("%d movements were applied from a refused batch", len(got))
	}

	f.requireOK(t, f.apply(t, f.aliceID, ops[:MaxSyncOperations]...))
	if got := f.rows(t); len(got) != MaxSyncOperations {
		t.Errorf("%d movements after a full batch, want %d", len(got), MaxSyncOperations)
	}
	f.requireFileMatchesDatabase(t)
}

func (m *failingMirror) Flush() error { m.calls++; return errors.New("disk full") }

func TestApplySyncKeepsTheChangesWhenTheFileCannotBeWritten(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	mirror := &failingMirror{}
	f.svc.EnableMirror(mirror, testutil.Logger())

	f.requireOK(t, f.apply(t, f.aliceID, food(1, "2026-03-05", 4000, "")))
	if mirror.calls != 1 {
		t.Errorf("mirror flushed %d times, want 1", mirror.calls)
	}
	if got := f.rows(t); len(got) != 1 {
		t.Fatalf("rows = %v, want the movement kept", got)
	}
	// The file is marked as behind, so a restart exports instead of importing.
	var revision, written int
	if err := f.db.QueryRow("SELECT revision, csv_revision FROM cashflow_csv_state WHERE id = 1").Scan(&revision, &written); err != nil {
		t.Fatalf("read csv state: %v", err)
	}
	if revision <= written {
		t.Errorf("revision = %d, file holds %d, want the file marked as behind", revision, written)
	}
}

func TestApplySyncWithoutAMirrorStillWrites(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	id := testutil.InsertUser(t, db, "alice")
	testutil.MustExec(t, db, `INSERT INTO cashflow_categories (name, direction, icon, color) VALUES ('food', 'expense', 'utensils', '#E76F51')`)
	results, err := New(store.New(db)).ApplySync(id, []SyncOperation{food(1, "2026-03-05", 100, "")})
	if err != nil || len(results) != 1 || results[0].Rejected != "" {
		t.Fatalf("ApplySync() = %v, %v, want one accepted result", results, err)
	}
}

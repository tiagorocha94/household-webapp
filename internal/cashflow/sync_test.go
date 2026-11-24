package cashflow_test

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/cashflow"
	"github.com/tiagorocha94/household-webapp/internal/cashflow/store"
	"github.com/tiagorocha94/household-webapp/internal/csvdata"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

// userID returns the ID of an imported user.
func userID(t *testing.T, db *sql.DB, name string) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow("SELECT id FROM users WHERE username = ?", name).Scan(&id); err != nil {
		t.Fatalf("find user %s: %v", name, err)
	}
	return id
}

// upsert changes the database the way a device would, through the store.
func upsert(t *testing.T, st *store.Store, m cashflow.MovementWrite) (created bool) {
	t.Helper()
	err := st.Write(func(w *store.Writer) error {
		var err error
		created, err = w.UpsertMovement(m)
		return err
	})
	if err != nil {
		t.Fatalf("UpsertMovement(%s) error = %v", m.SyncID, err)
	}
	return created
}

// importedSample returns a database holding the sample data, and the directory
// whose files match it.
func importedSample(t *testing.T) (*sql.DB, string) {
	t.Helper()
	db := testutil.NewDB(t)
	dir := t.TempDir()
	writeSample(t, dir)
	runner := newRunner()
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if err := runner.Export(db, dir); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	return db, dir
}

func movementRows(t *testing.T, dir string) [][]string {
	t.Helper()
	rows, err := csvdata.Read(filepath.Join(dir, "cashflow_movements.csv"), movementsHeader)
	if err != nil {
		t.Fatalf("read movements file: %v", err)
	}
	return rows
}

func TestMovementWriteValidate(t *testing.T) {
	t.Parallel()

	good := cashflow.MovementWrite{SyncID: sid(1), UserID: 1, Category: "salary", Date: "2026-03-01", AmountCents: 100}
	if err := good.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	tests := map[string]func(m *cashflow.MovementWrite){
		"missing sync id":    func(m *cashflow.MovementWrite) { m.SyncID = "" },
		"sync id not a uuid": func(m *cashflow.MovementWrite) { m.SyncID = "abc" },
		"uppercase sync id":  func(m *cashflow.MovementWrite) { m.SyncID = "AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA" },
		"impossible day":     func(m *cashflow.MovementWrite) { m.Date = "2026-02-30" },
		"unpadded day":       func(m *cashflow.MovementWrite) { m.Date = "2026-1-5" },
		"zero amount":        func(m *cashflow.MovementWrite) { m.AmountCents = 0 },
		"negative amount":    func(m *cashflow.MovementWrite) { m.AmountCents = -5 },
	}
	for name, change := range tests {
		m := good
		change(&m)
		if err := m.Validate(); !errors.Is(err, apperr.ErrBadRequest) {
			t.Errorf("%s: Validate() error = %v, want a bad-request error", name, err)
		}
	}
}

func TestWriterUpsertAndDelete(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	aliceID := testutil.InsertUser(t, db, "alice")
	bobID := testutil.InsertUser(t, db, "bob")
	testutil.MustExec(t, db,
		`INSERT INTO cashflow_categories (name, direction, icon, color) VALUES ('salary', 'income', 'briefcase', '#16A34A')`)
	testutil.MustExec(t, db,
		`INSERT INTO cashflow_categories (name, direction, icon, color) VALUES ('food', 'expense', 'utensils', '#E76F51')`)
	st := store.New(db)

	revision := func() string {
		return queryString(t, db, "SELECT COALESCE((SELECT revision FROM cashflow_csv_state WHERE id = 1), 0)")
	}
	m := cashflow.MovementWrite{SyncID: sid(1), UserID: aliceID, Category: "food", Date: "2026-03-02", AmountCents: 1250, Description: "Lunch"}

	if !upsert(t, st, m) {
		t.Error("first upsert created = false, want true")
	}
	if got := revision(); got != "1" {
		t.Errorf("revision after the first write = %s, want 1", got)
	}

	// The same movement again changes nothing, and does not count as a change.
	if upsert(t, st, m) {
		t.Error("repeated upsert created = true, want false")
	}
	if got := revision(); got != "1" {
		t.Errorf("revision after a repeated write = %s, want 1", got)
	}
	if got := queryString(t, db, "SELECT COUNT(*) FROM cashflow_movements"); got != "1" {
		t.Errorf("movement count = %s, want 1", got)
	}

	// Editing keeps the row and counts as a change.
	m.AmountCents = 1500
	m.Category = "salary"
	m.Description = ""
	if upsert(t, st, m) {
		t.Error("editing upsert created = true, want false")
	}
	got := queryString(t, db,
		`SELECT c.name || '/' || m.amount_cents || '/' || COALESCE(m.description, 'NULL')
		 FROM cashflow_movements m JOIN cashflow_categories c ON c.id = m.category_id WHERE m.sync_id = ?`, sid(1))
	if got != "salary/1500/NULL" {
		t.Errorf("edited movement = %s, want salary/1500/NULL", got)
	}
	if got := revision(); got != "2" {
		t.Errorf("revision after an edit = %s, want 2", got)
	}

	// Another person cannot touch it, and an unknown category is refused.
	other := m
	other.UserID = bobID
	err := st.Write(func(w *store.Writer) error {
		_, err := w.UpsertMovement(other)
		return err
	})
	if !errors.Is(err, cashflow.ErrSyncIDTaken) {
		t.Errorf("upsert by another person error = %v, want ErrSyncIDTaken", err)
	}
	unknown := m
	unknown.SyncID = sid(2)
	unknown.Category = "lottery"
	err = st.Write(func(w *store.Writer) error {
		_, err := w.UpsertMovement(unknown)
		return err
	})
	if !errors.Is(err, cashflow.ErrUnknownCategory) {
		t.Errorf("upsert in an unknown category error = %v, want ErrUnknownCategory", err)
	}

	// Deleting: not another person's, then the owner's, then again.
	err = st.Write(func(w *store.Writer) error {
		_, err := w.DeleteMovement(bobID, sid(1))
		return err
	})
	if !errors.Is(err, cashflow.ErrSyncIDTaken) {
		t.Errorf("delete by another person error = %v, want ErrSyncIDTaken", err)
	}
	for i, want := range []bool{true, false} {
		var deleted bool
		err := st.Write(func(w *store.Writer) error {
			var err error
			deleted, err = w.DeleteMovement(aliceID, sid(1))
			return err
		})
		if err != nil {
			t.Fatalf("delete %d error = %v", i+1, err)
		}
		if deleted != want {
			t.Errorf("delete %d deleted = %v, want %v", i+1, deleted, want)
		}
	}
	if got := revision(); got != "3" {
		t.Errorf("revision after a delete and a repeated delete = %s, want 3", got)
	}
	if got := queryString(t, db, "SELECT COUNT(*) FROM cashflow_movements"); got != "0" {
		t.Errorf("movement count = %s, want 0", got)
	}
}

func TestWriteKeepsNothingWhenTheBatchFails(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	aliceID := testutil.InsertUser(t, db, "alice")
	testutil.MustExec(t, db,
		`INSERT INTO cashflow_categories (name, direction, icon, color) VALUES ('food', 'expense', 'utensils', '#E76F51')`)
	st := store.New(db)

	boom := errors.New("boom")
	err := st.Write(func(w *store.Writer) error {
		if _, err := w.UpsertMovement(cashflow.MovementWrite{
			SyncID: sid(1), UserID: aliceID, Category: "food", Date: "2026-03-02", AmountCents: 100,
		}); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("Write() error = %v, want boom", err)
	}
	if got := queryString(t, db, "SELECT COUNT(*) FROM cashflow_movements"); got != "0" {
		t.Errorf("movement count = %s, want 0", got)
	}
	if got := queryString(t, db, "SELECT COUNT(*) FROM cashflow_csv_state"); got != "0" {
		t.Errorf("csv state rows = %s, want 0", got)
	}
}

func TestImportMovesASyncIDBetweenPeople(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	runner := newRunner()
	dir := t.TempDir()
	writeUsers(t, dir, "alice", "bob")
	writeCSV(t, filepath.Join(dir, "cashflow_budget_groups.csv"), sampleGroups)
	writeCSV(t, filepath.Join(dir, "cashflow_categories.csv"), sampleCategories)

	// alice is imported before bob, so handing the movement to her would clash
	// with bob's copy if the rows were replaced one person at a time.
	writeCSV(t, filepath.Join(dir, "cashflow_movements.csv"), [][]string{
		movementsHeader,
		{"bob", "salary", "2026-03-01", "100", "", sid(1)},
	})
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("first Import() error = %v", err)
	}
	writeCSV(t, filepath.Join(dir, "cashflow_movements.csv"), [][]string{
		movementsHeader,
		{"alice", "salary", "2026-03-01", "100", "", sid(1)},
	})
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("second Import() error = %v", err)
	}
	got := queryString(t, db,
		`SELECT u.username FROM cashflow_movements m JOIN users u ON u.id = m.user_id WHERE m.sync_id = ?`, sid(1))
	if got != "alice" {
		t.Errorf("owner after the file moved it = %s, want alice", got)
	}
}

func TestImportKeepsChangesThatAreNotInTheFileYet(t *testing.T) {
	t.Parallel()

	db, dir := importedSample(t)
	st := store.New(db)
	aliceID := userID(t, db, "alice")

	// A device adds a movement; the file does not have it yet.
	upsert(t, st, cashflow.MovementWrite{
		SyncID: sid(50), UserID: aliceID, Category: "groceries", Date: "2026-06-01",
		AmountCents: 1234, Description: "From the phone",
	})

	// Starting the server now must not undo it, even though the file is older.
	runner := newRunner()
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if got := queryString(t, db, "SELECT COUNT(*) FROM cashflow_movements"); got != "11" {
		t.Fatalf("movement count after Import() = %s, want 11", got)
	}

	// The export that follows brings the file up to date...
	if err := runner.Export(db, dir); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if got := len(movementRows(t, dir)); got != 11 {
		t.Errorf("rows in the file = %d, want 11", got)
	}
	if got := queryString(t, db, "SELECT revision || '/' || csv_revision FROM cashflow_csv_state"); got != "1/1" {
		t.Errorf("revision / csv_revision = %s, want 1/1", got)
	}

	// ...and the file is in charge again: dropping the row from it removes it.
	writeCSV(t, filepath.Join(dir, "cashflow_movements.csv"), sampleMovements)
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("Import() after editing the file error = %v", err)
	}
	if got := queryString(t, db, "SELECT COUNT(*) FROM cashflow_movements"); got != "10" {
		t.Errorf("movement count after editing the file = %s, want 10", got)
	}
}

func TestCSVMirrorFlushWritesOnlyWhenBehind(t *testing.T) {
	t.Parallel()

	db, dir := importedSample(t)
	mirror := cashflow.NewCSVMirror(db, dir)
	path := filepath.Join(dir, "cashflow_movements.csv")
	st := store.New(db)

	// Up to date: nothing is written.
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove the file: %v", err)
	}
	if err := mirror.Flush(); err != nil {
		t.Fatalf("Flush() when up to date error = %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Flush() when up to date wrote the file (stat error = %v)", err)
	}

	// A change makes the next flush write everything, including sync IDs.
	upsert(t, st, cashflow.MovementWrite{
		SyncID: sid(50), UserID: userID(t, db, "alice"), Category: "groceries", Date: "2026-06-01",
		AmountCents: 1234, Description: "From the phone",
	})
	if err := mirror.Flush(); err != nil {
		t.Fatalf("Flush() after a change error = %v", err)
	}
	rows := movementRows(t, dir)
	if len(rows) != 11 {
		t.Fatalf("rows in the file = %d, want 11", len(rows))
	}
	found := false
	for _, row := range rows {
		if row[5] == sid(50) {
			found = true
		}
	}
	if !found {
		t.Errorf("the file has no row with sync id %s", sid(50))
	}
	if got := queryString(t, db, "SELECT revision || '/' || csv_revision FROM cashflow_csv_state"); got != "1/1" {
		t.Errorf("revision / csv_revision = %s, want 1/1", got)
	}

	// Flushing again does nothing.
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove the file: %v", err)
	}
	if err := mirror.Flush(); err != nil {
		t.Fatalf("second Flush() error = %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("second Flush() wrote the file (stat error = %v)", err)
	}
}

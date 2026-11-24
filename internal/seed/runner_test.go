package seed

import (
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/csvdata"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

// fakeModule records how the Runner drives it. It stands in for a real module,
// so the orchestration is tested without any module's CSV rules.
type fakeModule struct {
	name      string
	files     []csvdata.File
	tables    []csvdata.Table
	importErr error
	syncErr   error
	insertSQL string // run inside the import transaction, if set

	log       *[]string
	gotFiles  map[string][][]string
	gotSyncAt time.Time
}

func (f *fakeModule) Name() string          { return f.name }
func (f *fakeModule) Files() []csvdata.File { return f.files }

func (f *fakeModule) Import(tx *sql.Tx, files map[string][][]string, _ *slog.Logger) error {
	*f.log = append(*f.log, "import "+f.name)
	f.gotFiles = files
	if f.insertSQL != "" {
		if _, err := tx.Exec(f.insertSQL); err != nil {
			return err
		}
	}
	return f.importErr
}

func (f *fakeModule) Sync(_ *sql.DB, now time.Time, _ *slog.Logger) error {
	*f.log = append(*f.log, "sync "+f.name)
	f.gotSyncAt = now
	return f.syncErr
}

func (f *fakeModule) Export(*sql.DB) ([]csvdata.Table, error) {
	*f.log = append(*f.log, "export "+f.name)
	return f.tables, nil
}

func newFake(name string, log *[]string, files ...csvdata.File) *fakeModule {
	return &fakeModule{name: name, files: files, log: log}
}

func TestRunnerImportsModulesInOrderWithTheirFiles(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	dir := t.TempDir()
	writeTestCSV(t, filepath.Join(dir, "a.csv"), [][]string{{"x"}, {"1"}, {"2"}})
	writeTestCSV(t, filepath.Join(dir, "b.csv"), [][]string{{"y"}})

	var log []string
	a := newFake("a", &log, csvdata.File{Name: "a.csv", Header: []string{"x"}}, csvdata.File{Name: "missing.csv", Header: []string{"z"}})
	b := newFake("b", &log, csvdata.File{Name: "b.csv", Header: []string{"y"}})
	if err := NewRunner(testutil.Logger(), a, b).Import(db, dir); err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if want := []string{"import a", "import b"}; !reflect.DeepEqual(log, want) {
		t.Errorf("calls = %v, want %v", log, want)
	}
	if want := map[string][][]string{"a.csv": {{"1"}, {"2"}}}; !reflect.DeepEqual(a.gotFiles, want) {
		t.Errorf("a got files %v, want %v (missing files skipped)", a.gotFiles, want)
	}
	if got, ok := b.gotFiles["b.csv"]; !ok || len(got) != 0 {
		t.Errorf("b got files %v, want an empty b.csv", b.gotFiles)
	}
}

func TestRunnerSkipsModulesWithoutFiles(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	var log []string
	m := newFake("m", &log, csvdata.File{Name: "m.csv", Header: []string{"x"}})
	if err := NewRunner(testutil.Logger(), m).Import(db, t.TempDir()); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if len(log) != 0 {
		t.Errorf("calls = %v, want none when no file exists", log)
	}
}

func TestRunnerChecksEveryFileBeforeImportingAnything(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	dir := t.TempDir()
	writeTestCSV(t, filepath.Join(dir, "a.csv"), [][]string{{"x"}, {"1"}})
	writeTestCSV(t, filepath.Join(dir, "b.csv"), [][]string{{"wrong"}, {"1"}})

	var log []string
	a := newFake("a", &log, csvdata.File{Name: "a.csv", Header: []string{"x"}})
	b := newFake("b", &log, csvdata.File{Name: "b.csv", Header: []string{"y"}})
	err := NewRunner(testutil.Logger(), a, b).Import(db, dir)
	if err == nil || !strings.Contains(err.Error(), "b.csv") {
		t.Fatalf("Import() error = %v, want a header error naming b.csv", err)
	}
	if len(log) != 0 {
		t.Errorf("calls = %v, want none: a later file failed the check", log)
	}
}

func TestRunnerRollsBackAModuleThatFails(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	dir := t.TempDir()
	writeTestCSV(t, filepath.Join(dir, "a.csv"), [][]string{{"x"}, {"1"}})

	var log []string
	boom := errors.New("boom")
	a := newFake("a", &log, csvdata.File{Name: "a.csv", Header: []string{"x"}})
	a.insertSQL = "INSERT INTO users (username, password_hash) VALUES ('ghost', 'h')"
	a.importErr = boom

	err := NewRunner(testutil.Logger(), a).Import(db, dir)
	if !errors.Is(err, boom) || !strings.Contains(err.Error(), "import a") {
		t.Fatalf("Import() error = %v, want it to wrap boom and name module a", err)
	}
	if got := queryString(t, db, "SELECT CAST(COUNT(*) AS TEXT) FROM users WHERE username = 'ghost'"); got != "0" {
		t.Errorf("ghost users = %s, want 0: the module's transaction must roll back", got)
	}
}

func TestRunnerCommitsEachModuleSeparately(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	dir := t.TempDir()
	writeTestCSV(t, filepath.Join(dir, "a.csv"), [][]string{{"x"}, {"1"}})
	writeTestCSV(t, filepath.Join(dir, "b.csv"), [][]string{{"y"}, {"1"}})

	var log []string
	a := newFake("a", &log, csvdata.File{Name: "a.csv", Header: []string{"x"}})
	a.insertSQL = "INSERT INTO users (username, password_hash) VALUES ('kept', 'h')"
	b := newFake("b", &log, csvdata.File{Name: "b.csv", Header: []string{"y"}})
	b.importErr = errors.New("boom")

	if err := NewRunner(testutil.Logger(), a, b).Import(db, dir); err == nil {
		t.Fatal("Import() succeeded, want the error from b")
	}
	if got := queryString(t, db, "SELECT CAST(COUNT(*) AS TEXT) FROM users WHERE username = 'kept'"); got != "1" {
		t.Errorf("kept users = %s, want 1: module a was already committed", got)
	}
}

func TestRunnerSyncPassesTheCurrentTimeToEveryModuleInOrder(t *testing.T) {
	t.Parallel()

	var log []string
	a, b := newFake("a", &log), newFake("b", &log)
	now := time.Date(2026, 3, 1, 0, 30, 0, 0, time.UTC)
	r := NewRunner(testutil.Logger(), a, b)
	r.now = func() time.Time { return now }

	if err := r.Sync(nil); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if want := []string{"sync a", "sync b"}; !reflect.DeepEqual(log, want) {
		t.Errorf("calls = %v, want %v", log, want)
	}
	if !a.gotSyncAt.Equal(now) || !b.gotSyncAt.Equal(now) {
		t.Errorf("sync times = %v and %v, want %v for both", a.gotSyncAt, b.gotSyncAt, now)
	}
}

func TestRunnerSyncStopsAtTheFirstFailure(t *testing.T) {
	t.Parallel()

	var log []string
	boom := errors.New("boom")
	a, b := newFake("a", &log), newFake("b", &log)
	a.syncErr = boom

	err := NewRunner(testutil.Logger(), a, b).Sync(nil)
	if !errors.Is(err, boom) || !strings.Contains(err.Error(), "sync a") {
		t.Fatalf("Sync() error = %v, want it to wrap boom and name module a", err)
	}
	if want := []string{"sync a"}; !reflect.DeepEqual(log, want) {
		t.Errorf("calls = %v, want %v: b must not run after a failed", log, want)
	}
}

func TestRunnerExportWritesEveryModulesTables(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "nested", "csv")
	var log []string
	a, b, none := newFake("a", &log), newFake("b", &log), newFake("none", &log)
	a.tables = []csvdata.Table{{Name: "a1.csv", Rows: [][]string{{"h"}, {"1"}}}, {Name: "a2.csv", Rows: [][]string{{"h"}}}}
	b.tables = []csvdata.Table{{Name: "b.csv", Rows: [][]string{{"k"}, {"v"}}}}

	if err := NewRunner(testutil.Logger(), a, none, b).Export(nil, dir); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if want := []string{"export a", "export none", "export b"}; !reflect.DeepEqual(log, want) {
		t.Errorf("calls = %v, want %v", log, want)
	}
	assertCSVData(t, filepath.Join(dir, "a1.csv"), []string{"h"}, [][]string{{"1"}})
	assertCSVData(t, filepath.Join(dir, "a2.csv"), []string{"h"}, nil)
	assertCSVData(t, filepath.Join(dir, "b.csv"), []string{"k"}, [][]string{{"v"}})
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 3 {
		t.Fatalf("directory = %v, %v, want exactly the 3 exported files", entries, err)
	}
}

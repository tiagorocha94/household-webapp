package backup

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

func day(s string) time.Time {
	d, err := time.Parse(dayLayout, s)
	if err != nil {
		panic(err)
	}
	return d.Add(3 * time.Hour)
}

func names(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

func dbWithNote(t *testing.T, note string) *sql.DB {
	t.Helper()
	db := testutil.NewDB(t)
	if _, err := db.Exec("CREATE TABLE backup_note (text TEXT NOT NULL)"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec("INSERT INTO backup_note (text) VALUES (?)", note); err != nil {
		t.Fatalf("insert: %v", err)
	}
	return db
}

func noteIn(t *testing.T, path string) string {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open backup: %v", err)
	}
	defer func() { _ = db.Close() }()
	var text string
	if err := db.QueryRow("SELECT text FROM backup_note").Scan(&text); err != nil {
		t.Fatalf("read backup: %v", err)
	}
	return text
}

func TestRunWritesAFileThatOpensAsTheDatabase(t *testing.T) {
	t.Parallel()

	db := dbWithNote(t, "hello")
	dir := filepath.Join(t.TempDir(), "backups", "nested")

	res, err := Run(context.Background(), db, dir, 7, day("2026-11-27"))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if want := filepath.Join(dir, "household-2026-11-27.db"); res.File != want {
		t.Fatalf("File = %q, want %q", res.File, want)
	}
	if got := noteIn(t, res.File); got != "hello" {
		t.Fatalf("backup holds %q, want hello", got)
	}
	if got := names(t, dir); !slices.Equal(got, []string{"household-2026-11-27.db"}) {
		t.Fatalf("files = %v, want only the backup (no temporary file)", got)
	}
}

func TestRunKeepsTheFileAndTheDirectoryPrivate(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("file modes are not enforced on Windows")
	}

	db := dbWithNote(t, "x")
	dir := filepath.Join(t.TempDir(), "backups")
	res, err := Run(context.Background(), db, dir, 3, day("2026-11-27"))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, p := range []string{dir, res.File} {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("stat %s: %v", p, err)
		}
		if info.Mode().Perm()&0o077 != 0 {
			t.Errorf("%s mode = %v, want no access for group or others", p, info.Mode().Perm())
		}
	}
}

func TestALaterRunOfTheSameDayReplacesTheBackup(t *testing.T) {
	t.Parallel()

	db := dbWithNote(t, "first")
	dir := t.TempDir()
	if _, err := Run(context.Background(), db, dir, 3, day("2026-11-27")); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	if _, err := db.Exec("UPDATE backup_note SET text = 'second'"); err != nil {
		t.Fatalf("update: %v", err)
	}
	res, err := Run(context.Background(), db, dir, 3, day("2026-11-27").Add(5*time.Hour))
	if err != nil {
		t.Fatalf("second Run() error = %v", err)
	}
	if got := noteIn(t, res.File); got != "second" {
		t.Fatalf("backup holds %q, want the later state", got)
	}
	if got := names(t, dir); len(got) != 1 {
		t.Fatalf("files = %v, want one a day", got)
	}
}

func TestRunKeepsOnlyTheNewestBackups(t *testing.T) {
	t.Parallel()

	db := dbWithNote(t, "x")
	dir := t.TempDir()
	var last Result
	for _, d := range []string{"2026-11-24", "2026-11-25", "2026-11-26", "2026-11-27"} {
		res, err := Run(context.Background(), db, dir, 2, day(d))
		if err != nil {
			t.Fatalf("Run(%s) error = %v", d, err)
		}
		last = res
	}
	want := []string{"household-2026-11-26.db", "household-2026-11-27.db"}
	if got := names(t, dir); !slices.Equal(got, want) {
		t.Fatalf("files = %v, want %v", got, want)
	}
	if last.Removed != 1 {
		t.Fatalf("Removed = %d, want 1", last.Removed)
	}
}

func TestPruningNeverTouchesFilesThatAreNotBackups(t *testing.T) {
	t.Parallel()

	db := dbWithNote(t, "x")
	dir := t.TempDir()
	others := []string{"notes.txt", "household-2026-11-01.db.bak", "household.db", "household-latest.db"}
	for _, name := range others {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("mine"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "household-2026-01-01.db"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	for _, d := range []string{"2026-11-25", "2026-11-26"} {
		if _, err := Run(context.Background(), db, dir, 1, day(d)); err != nil {
			t.Fatalf("Run(%s) error = %v", d, err)
		}
	}
	got := names(t, dir)
	for _, name := range append(others, "household-2026-01-01.db", "household-2026-11-26.db") {
		if !slices.Contains(got, name) {
			t.Errorf("%s is gone; files = %v", name, got)
		}
	}
	if slices.Contains(got, "household-2026-11-25.db") {
		t.Errorf("the older backup was kept; files = %v", got)
	}
}

func TestALeftoverTemporaryFileDoesNotBlockTheBackup(t *testing.T) {
	t.Parallel()

	db := dbWithNote(t, "fresh")
	dir := t.TempDir()
	leftover := filepath.Join(dir, "household-2026-11-27.db.tmp")
	if err := os.WriteFile(leftover, []byte("cut short"), 0o600); err != nil {
		t.Fatalf("write leftover: %v", err)
	}
	res, err := Run(context.Background(), db, dir, 3, day("2026-11-27"))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := noteIn(t, res.File); got != "fresh" {
		t.Fatalf("backup holds %q", got)
	}
	if _, err := os.Stat(leftover); !os.IsNotExist(err) {
		t.Fatalf("leftover still there: %v", err)
	}
}

func TestAFailedRunLeavesTheEarlierBackupsAlone(t *testing.T) {
	t.Parallel()

	db := dbWithNote(t, "x")
	dir := t.TempDir()
	if _, err := Run(context.Background(), db, dir, 3, day("2026-11-26")); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Run(ctx, db, dir, 3, day("2026-11-27")); err == nil {
		t.Fatal("Run() with a cancelled context succeeded")
	}
	if got := names(t, dir); !slices.Equal(got, []string{"household-2026-11-26.db"}) {
		t.Fatalf("files = %v, want the earlier backup only", got)
	}
}

func TestRunRefusesAKeepBelowOne(t *testing.T) {
	t.Parallel()

	db := dbWithNote(t, "x")
	dir := t.TempDir()
	if _, err := Run(context.Background(), db, dir, 0, day("2026-11-27")); err == nil {
		t.Fatal("Run() with keep 0 succeeded")
	}
	if got := names(t, dir); len(got) != 0 {
		t.Fatalf("files = %v, want none", got)
	}
}

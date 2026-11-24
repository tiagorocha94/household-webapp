package savings_test

import (
	"database/sql"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/csvdata"
	"github.com/tiagorocha94/household-webapp/internal/savings"
	"github.com/tiagorocha94/household-webapp/internal/seed"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

var (
	accountsHeader  = []string{"name", "owner", "target_cents", "notes"}
	movementsHeader = []string{"account", "month", "type", "amount_cents", "description", "movement"}
	snapshotsHeader = []string{"account", "month", "value_cents", "description"}
)

func writeCSV(t *testing.T, path string, rows [][]string) {
	t.Helper()
	if err := csvdata.Write(path, rows); err != nil {
		t.Fatalf("write test CSV %s: %v", path, err)
	}
}

func queryString(t *testing.T, db *sql.DB, query string) string {
	t.Helper()
	var got sql.NullString
	if err := db.QueryRow(query).Scan(&got); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return got.String
}

func writeSample(t *testing.T, dir string) {
	t.Helper()
	writeCSV(t, filepath.Join(dir, "savings_accounts.csv"), [][]string{
		accountsHeader,
		{"Emergency fund", "", "500000", "Certificados de Aforro"},
		{"Holiday pot", "", "", ""},
	})
	writeCSV(t, filepath.Join(dir, "savings_movements.csv"), [][]string{
		movementsHeader,
		{"Emergency fund", "2026-01", "deposit", "300000", "Opening", ""},
		{"Emergency fund", "2026-02", "withdrawal", "50000", "Car repair", ""},
		{"Emergency fund", "2026-03", "deposit", "10000", "", ""},
	})
	writeCSV(t, filepath.Join(dir, "savings_snapshots.csv"), [][]string{
		snapshotsHeader,
		{"Emergency fund", "2026-03", "265000", ""},
	})
}

func TestSavingsImportAndExportRoundTrip(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	runner := seed.NewRunner(testutil.Logger(), savings.SeedModule{})
	dir := t.TempDir()
	writeSample(t, dir)

	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if got := queryString(t, db, "SELECT target_cents FROM savings_accounts WHERE name = 'Emergency fund'"); got != "500000" {
		t.Errorf("target = %q, want 500000", got)
	}
	if got := queryString(t, db, "SELECT COALESCE(target_cents, 'none') FROM savings_accounts WHERE name = 'Holiday pot'"); got != "none" {
		t.Errorf("target of an account without one = %q, want none", got)
	}

	exportDir := t.TempDir()
	if err := runner.Export(db, exportDir); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	for name, header := range map[string][]string{
		"savings_accounts.csv":  accountsHeader,
		"savings_movements.csv": movementsHeader,
		"savings_snapshots.csv": snapshotsHeader,
	} {
		want, err := csvdata.Read(filepath.Join(dir, name), header)
		if err != nil {
			t.Fatalf("read source %s: %v", name, err)
		}
		got, err := csvdata.Read(filepath.Join(exportDir, name), header)
		if err != nil {
			t.Fatalf("read export %s: %v", name, err)
		}
		// The export is ordered by name, so compare as sets of rows.
		if !sameRows(got, want) {
			t.Errorf("exported %s = %#v, want %#v", name, got, want)
		}
	}
}

func sameRows(a, b [][]string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]int)
	for _, r := range a {
		seen[strings.Join(r, "\x00")]++
	}
	for _, r := range b {
		seen[strings.Join(r, "\x00")]--
	}
	return reflect.DeepEqual(nonZero(seen), map[string]int{})
}

func nonZero(m map[string]int) map[string]int {
	out := make(map[string]int)
	for k, v := range m {
		if v != 0 {
			out[k] = v
		}
	}
	return out
}

func TestSavingsImportIsIdempotentAndMirrorsCSV(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	runner := seed.NewRunner(testutil.Logger(), savings.SeedModule{})
	dir := t.TempDir()
	writeSample(t, dir)
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	ids := "SELECT group_concat(id) FROM (SELECT id FROM savings_movements ORDER BY id)"
	before := queryString(t, db, ids)
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("second Import() error = %v", err)
	}
	if got := queryString(t, db, ids); got != before {
		t.Errorf("movement ids = %s, want %s", got, before)
	}

	// Removing a row from a file removes it from the database.
	writeCSV(t, filepath.Join(dir, "savings_movements.csv"), [][]string{
		movementsHeader,
		{"Emergency fund", "2026-01", "deposit", "300000", "Opening", ""},
	})
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("third Import() error = %v", err)
	}
	if got := queryString(t, db, "SELECT COUNT(*) FROM savings_movements"); got != "1" {
		t.Errorf("movements = %s, want 1", got)
	}
}

func TestSavingsImportRejectsBadRows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		accounts [][]string
		moves    [][]string
		want     string
	}{
		{"missing name", [][]string{accountsHeader, {"", "", "", ""}}, [][]string{movementsHeader}, "name is required"},
		{"duplicate account", [][]string{accountsHeader, {"A", "", "", ""}, {"A", "", "", ""}}, [][]string{movementsHeader}, "more than once"},
		{"zero target", [][]string{accountsHeader, {"A", "", "0", ""}}, [][]string{movementsHeader}, "target_cents must be positive"},
		{"bad target", [][]string{accountsHeader, {"A", "", "-5", ""}}, [][]string{movementsHeader}, "target_cents"},
		{"unknown owner", [][]string{accountsHeader, {"A", "nobody", "", ""}}, [][]string{movementsHeader}, "user"},
		{"unknown account", [][]string{accountsHeader, {"A", "", "", ""}}, [][]string{movementsHeader, {"B", "2026-01", "deposit", "100", "", ""}}, "not found"},
		{"bad type", [][]string{accountsHeader, {"A", "", "", ""}}, [][]string{movementsHeader, {"A", "2026-01", "gift", "100", "", ""}}, "invalid type"},
		{"zero amount", [][]string{accountsHeader, {"A", "", "", ""}}, [][]string{movementsHeader, {"A", "2026-01", "deposit", "0", "", ""}}, "must be positive"},
		{"bad month", [][]string{accountsHeader, {"A", "", "", ""}}, [][]string{movementsHeader, {"A", "2026-13", "deposit", "100", "", ""}}, "invalid month"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			writeCSV(t, filepath.Join(dir, "savings_accounts.csv"), tt.accounts)
			writeCSV(t, filepath.Join(dir, "savings_movements.csv"), tt.moves)
			runner := seed.NewRunner(testutil.Logger(), savings.SeedModule{})
			err := runner.Import(testutil.NewDB(t), dir)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Import() error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestSavingsMovementKeepsItsLinkToACashflowMovement(t *testing.T) {
	t.Parallel()

	const link = "123e4567-e89b-12d3-a456-426614174000"
	db := testutil.NewDB(t)
	runner := seed.NewRunner(testutil.Logger(), savings.SeedModule{})
	dir := t.TempDir()
	writeCSV(t, filepath.Join(dir, "savings_accounts.csv"), [][]string{accountsHeader, {"Pot", "", "", ""}})
	writeCSV(t, filepath.Join(dir, "savings_movements.csv"), [][]string{
		movementsHeader,
		{"Pot", "2026-01", "deposit", "10000", "", link},
		{"Pot", "2026-01", "deposit", "10000", "", ""},
	})
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if got := queryString(t, db, "SELECT movement_sync_id FROM savings_movements WHERE movement_sync_id IS NOT NULL"); got != link {
		t.Errorf("movement_sync_id = %q, want %q", got, link)
	}

	exportDir := t.TempDir()
	if err := runner.Export(db, exportDir); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	want, err := csvdata.Read(filepath.Join(dir, "savings_movements.csv"), movementsHeader)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	got, err := csvdata.Read(filepath.Join(exportDir, "savings_movements.csv"), movementsHeader)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	if !sameRows(got, want) {
		t.Errorf("exported movements = %#v, want %#v", got, want)
	}

	writeCSV(t, filepath.Join(dir, "savings_movements.csv"), [][]string{
		movementsHeader,
		{"Pot", "2026-01", "deposit", "10000", "", "not-a-sync-id"},
	})
	err = runner.Import(db, dir)
	if err == nil || !strings.Contains(err.Error(), "movement") {
		t.Errorf("Import() with a bad link error = %v, want one naming the movement", err)
	}
}

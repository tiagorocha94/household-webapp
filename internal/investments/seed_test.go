package investments_test

import (
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/auth"
	"github.com/tiagorocha94/household-webapp/internal/csvdata"
	"github.com/tiagorocha94/household-webapp/internal/investments"
	"github.com/tiagorocha94/household-webapp/internal/seed"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

var (
	entitiesHeader      = []string{"name", "kind", "user"}
	instrumentsHeader   = []string{"entity", "name", "type", "notes"}
	contributionsHeader = []string{"entity", "instrument", "month", "amount_cents", "description", "movement"}
	snapshotsHeader     = []string{"entity", "instrument", "month", "value_cents", "description"}
)

func writeCSV(t *testing.T, path string, rows [][]string) {
	t.Helper()
	if err := csvdata.Write(path, rows); err != nil {
		t.Fatalf("write test CSV %s: %v", path, err)
	}
}

func queryString(t *testing.T, db *sql.DB, query string, args ...any) string {
	t.Helper()
	var got sql.NullString
	if err := db.QueryRow(query, args...).Scan(&got); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return got.String
}

// writeSample writes one broker with an ETF (two contributions in one month,
// three snapshots with a drop) and one state entity with a single-contribution
// instrument.
func writeSample(t *testing.T, dir string) {
	t.Helper()
	writeCSV(t, filepath.Join(dir, "investment_entities.csv"), [][]string{
		entitiesHeader,
		{"Broker", "broker", ""},
		{"State", "state", ""},
	})
	writeCSV(t, filepath.Join(dir, "investment_instruments.csv"), [][]string{
		instrumentsHeader,
		{"Broker", "World ETF", "etf", "Core holding"},
		{"State", "Certificados", "certificados_aforro", ""},
	})
	writeCSV(t, filepath.Join(dir, "investment_contributions.csv"), [][]string{
		contributionsHeader,
		{"Broker", "World ETF", "2026-01", "10000", "", ""},
		{"Broker", "World ETF", "2026-01", "5000", "Extra", ""},
		{"Broker", "World ETF", "2026-02", "10000", "", ""},
		{"State", "Certificados", "2026-01", "20000", "", ""},
	})
	writeCSV(t, filepath.Join(dir, "investment_snapshots.csv"), [][]string{
		snapshotsHeader,
		{"Broker", "World ETF", "2026-01", "15500", ""},
		{"Broker", "World ETF", "2026-02", "26000", ""},
		{"Broker", "World ETF", "2026-03", "24000", "Market drop"},
		{"State", "Certificados", "2026-01", "20000", ""},
	})
}

func TestInvestmentDerivedFiguresAndRoundTrip(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	runner := seed.NewRunner(testutil.Logger(), investments.SeedModule{})
	dir := t.TempDir()
	writeSample(t, dir)

	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	// Two contributions in one month both count; the latest snapshot wins and
	// the gain percentage is scaled by 100: (24000-25000)/25000 = -4.00%.
	var allocated, current, gain int64
	var pct, contributions, snapshots int
	var last string
	err := db.QueryRow(
		`SELECT allocated_cents, current_value_cents, gain_cents, gain_pct,
		        contribution_count, snapshot_count, last_snapshot_month
		 FROM investment_instrument_summary_view WHERE name = 'World ETF'`,
	).Scan(&allocated, &current, &gain, &pct, &contributions, &snapshots, &last)
	if err != nil {
		t.Fatalf("query instrument summary: %v", err)
	}
	if allocated != 25000 || current != 24000 || gain != -1000 || pct != -400 ||
		contributions != 3 || snapshots != 3 || last != "2026-03" {
		t.Errorf("instrument summary = (%d, %d, %d, %d, %d, %d, %s)",
			allocated, current, gain, pct, contributions, snapshots, last)
	}

	// The entity view aggregates its instruments without fan-out.
	err = db.QueryRow(
		`SELECT instrument_count, allocated_cents, current_value_cents, gain_pct
		 FROM investment_entity_summary_view WHERE name = 'Broker'`,
	).Scan(&contributions, &allocated, &current, &pct)
	if err != nil {
		t.Fatalf("query entity summary: %v", err)
	}
	if contributions != 1 || allocated != 25000 || current != 24000 || pct != -400 {
		t.Errorf("entity summary = (%d, %d, %d, %d)", contributions, allocated, current, pct)
	}

	exportDir := t.TempDir()
	if err := runner.Export(db, exportDir); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	wantFiles := map[string][]string{
		"investment_entities.csv":      entitiesHeader,
		"investment_instruments.csv":   instrumentsHeader,
		"investment_contributions.csv": contributionsHeader,
		"investment_snapshots.csv":     snapshotsHeader,
	}
	for name, header := range wantFiles {
		want, err := csvdata.Read(filepath.Join(dir, name), header)
		if err != nil {
			t.Fatalf("read source %s: %v", name, err)
		}
		got, err := csvdata.Read(filepath.Join(exportDir, name), header)
		if err != nil {
			t.Fatalf("read export %s: %v", name, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("exported %s = %#v, want %#v", name, got, want)
		}
	}
}

func TestInvestmentImportIsIdempotentAndMirrorsCSV(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	runner := seed.NewRunner(testutil.Logger(), investments.SeedModule{})
	dir := t.TempDir()
	writeSample(t, dir)

	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	const oldStamp = "2000-01-01 00:00:00"
	if _, err := db.Exec("UPDATE investment_entities SET updated_at = ?", oldStamp); err != nil {
		t.Fatalf("set updated_at: %v", err)
	}
	ids := "SELECT group_concat(id) FROM (SELECT id FROM investment_contributions ORDER BY id)"
	before := queryString(t, db, ids)

	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("second Import() error = %v", err)
	}
	if got := queryString(t, db, ids); got != before {
		t.Errorf("contribution ids = %s, want %s", got, before)
	}
	if got := queryString(t, db, "SELECT MAX(CAST(updated_at AS TEXT)) FROM investment_entities"); got != oldStamp {
		t.Errorf("entity updated_at = %s, want untouched", got)
	}

	// Removing a row from a file removes it from the database.
	writeCSV(t, filepath.Join(dir, "investment_contributions.csv"), [][]string{
		contributionsHeader,
		{"Broker", "World ETF", "2026-01", "10000", "", ""},
	})
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("pruning Import() error = %v", err)
	}
	if got := queryString(t, db, "SELECT COUNT(*) FROM investment_contributions"); got != "1" {
		t.Errorf("contribution count = %s, want 1", got)
	}

	// Removing an entity cascades to its instruments and their rows.
	writeCSV(t, filepath.Join(dir, "investment_entities.csv"), [][]string{
		entitiesHeader,
		{"Broker", "broker", ""},
	})
	writeCSV(t, filepath.Join(dir, "investment_instruments.csv"), [][]string{
		instrumentsHeader,
		{"Broker", "World ETF", "etf", "Core holding"},
	})
	// The child files may not name the removed entity any more.
	writeCSV(t, filepath.Join(dir, "investment_snapshots.csv"), [][]string{
		snapshotsHeader,
		{"Broker", "World ETF", "2026-01", "15500", ""},
		{"Broker", "World ETF", "2026-02", "26000", ""},
		{"Broker", "World ETF", "2026-03", "24000", "Market drop"},
	})
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("entity-removing Import() error = %v", err)
	}
	for table, want := range map[string]string{
		"investment_entities": "1", "investment_instruments": "1",
		"investment_snapshots": "3",
	} {
		if got := queryString(t, db, "SELECT COUNT(*) FROM "+table); got != want {
			t.Errorf("%s count = %s, want %s", table, got, want)
		}
	}
}

func TestInvestmentImportRejectsBadRows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file string
		rows [][]string
	}{
		{"unknown kind", "investment_entities.csv", [][]string{entitiesHeader, {"Broker", "casino", ""}}},
		{"duplicate entity", "investment_entities.csv", [][]string{entitiesHeader, {"Broker", "broker", ""}, {"Broker", "bank", ""}}},
		{"unknown type", "investment_instruments.csv", [][]string{instrumentsHeader, {"Broker", "X", "crypto", ""}}},
		{"cash is no instrument type", "investment_instruments.csv", [][]string{instrumentsHeader, {"Broker", "X", "cash", ""}}},
		{"unknown entity", "investment_instruments.csv", [][]string{instrumentsHeader, {"Nope", "X", "etf", ""}}},
		{"bad month", "investment_contributions.csv", [][]string{contributionsHeader, {"Broker", "ETF", "2026-13", "100", "", ""}}},
		{"zero contribution", "investment_contributions.csv", [][]string{contributionsHeader, {"Broker", "ETF", "2026-01", "0", "", ""}}},
		{"decimal cents", "investment_snapshots.csv", [][]string{snapshotsHeader, {"Broker", "ETF", "2026-01", "10.50", ""}}},
		{"unknown instrument", "investment_snapshots.csv", [][]string{snapshotsHeader, {"Broker", "Nope", "2026-01", "100", ""}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := testutil.NewDB(t)
			runner := seed.NewRunner(testutil.Logger(), investments.SeedModule{})
			dir := t.TempDir()
			writeCSV(t, filepath.Join(dir, "investment_entities.csv"), [][]string{entitiesHeader, {"Broker", "broker", ""}})
			writeCSV(t, filepath.Join(dir, "investment_instruments.csv"), [][]string{instrumentsHeader, {"Broker", "ETF", "etf", ""}})
			writeCSV(t, filepath.Join(dir, tt.file), tt.rows)
			if err := runner.Import(db, dir); err == nil {
				t.Fatal("Import() error = nil, want an error")
			}
		})
	}
}

func TestInvestmentEntityOwner(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	runner := seed.NewRunner(testutil.Logger(), auth.SeedModule{}, investments.SeedModule{})
	dir := t.TempDir()
	writeCSV(t, filepath.Join(dir, "users.csv"), [][]string{
		{"username", "password"},
		{"alice", testutil.PasswordHash},
	})
	writeCSV(t, filepath.Join(dir, "investment_entities.csv"), [][]string{
		entitiesHeader,
		{"Alice Broker", "broker", "alice"},
		{"Shared Bank", "bank", ""},
	})
	writeCSV(t, filepath.Join(dir, "investment_instruments.csv"), [][]string{
		instrumentsHeader,
		{"Alice Broker", "Alice ETF", "etf", ""},
		{"Shared Bank", "Shared Fund", "fund", ""},
	})
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	// An account names its owner; one without belongs to the household, and its
	// instruments take the account's owner.
	account := "SELECT COALESCE(username, '') FROM investment_entity_summary_view WHERE name = ?"
	if got := queryString(t, db, account, "Alice Broker"); got != "alice" {
		t.Errorf("owner of Alice Broker = %q, want alice", got)
	}
	if got := queryString(t, db, account, "Shared Bank"); got != "" {
		t.Errorf("owner of Shared Bank = %q, want none", got)
	}
	holding := "SELECT COALESCE(username, '') FROM investment_instrument_summary_view WHERE name = ?"
	if got := queryString(t, db, holding, "Alice ETF"); got != "alice" {
		t.Errorf("owner of Alice ETF = %q, want alice", got)
	}
	if got := queryString(t, db, holding, "Shared Fund"); got != "" {
		t.Errorf("owner of Shared Fund = %q, want none", got)
	}

	exportDir := t.TempDir()
	if err := runner.Export(db, exportDir); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	got, err := csvdata.Read(filepath.Join(exportDir, "investment_entities.csv"), entitiesHeader)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	want := [][]string{
		{"Alice Broker", "broker", "alice"},
		{"Shared Bank", "bank", ""},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("exported entities = %#v, want %#v", got, want)
	}

	// Clearing the user hands the account back to the household.
	writeCSV(t, filepath.Join(dir, "investment_entities.csv"), [][]string{
		entitiesHeader,
		{"Alice Broker", "broker", ""},
		{"Shared Bank", "bank", ""},
	})
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("Import(cleared owner) error = %v", err)
	}
	if got := queryString(t, db, account, "Alice Broker"); got != "" {
		t.Errorf("owner after clearing = %q, want none", got)
	}

	// An unknown user is an error, not a silently unowned account.
	writeCSV(t, filepath.Join(dir, "investment_entities.csv"), [][]string{
		entitiesHeader,
		{"Alice Broker", "broker", "nobody"},
	})
	if err := runner.Import(db, dir); err == nil {
		t.Fatal("Import(unknown user) error = nil, want an error")
	}
}

func TestContributionKeepsItsLinkToACashflowMovement(t *testing.T) {
	t.Parallel()

	const link = "123e4567-e89b-12d3-a456-426614174000"
	db := testutil.NewDB(t)
	runner := seed.NewRunner(testutil.Logger(), investments.SeedModule{})
	dir := t.TempDir()
	writeSample(t, dir)
	writeCSV(t, filepath.Join(dir, "investment_contributions.csv"), [][]string{
		contributionsHeader,
		{"Broker", "World ETF", "2026-01", "10000", "", link},
		{"Broker", "World ETF", "2026-01", "10000", "", ""},
	})
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if got := queryString(t, db, "SELECT movement_sync_id FROM investment_contributions WHERE movement_sync_id IS NOT NULL"); got != link {
		t.Errorf("movement_sync_id = %q, want %q", got, link)
	}

	exportDir := t.TempDir()
	if err := runner.Export(db, exportDir); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	want, err := csvdata.Read(filepath.Join(dir, "investment_contributions.csv"), contributionsHeader)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	got, err := csvdata.Read(filepath.Join(exportDir, "investment_contributions.csv"), contributionsHeader)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("exported %d rows, want %d", len(got), len(want))
	}
	linked := 0
	for _, row := range got {
		if row[5] == link {
			linked++
		}
	}
	if linked != 1 {
		t.Errorf("%d exported rows carry the link, want 1", linked)
	}

	writeCSV(t, filepath.Join(dir, "investment_contributions.csv"), [][]string{
		contributionsHeader,
		{"Broker", "World ETF", "2026-01", "10000", "", "not-a-sync-id"},
	})
	if err := runner.Import(db, dir); err == nil {
		t.Error("Import() with a bad link error = nil, want a rejection")
	}
}

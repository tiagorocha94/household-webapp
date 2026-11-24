package credits_test

import (
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/auth"
	"github.com/tiagorocha94/household-webapp/internal/credits"
	"github.com/tiagorocha94/household-webapp/internal/csvdata"
	"github.com/tiagorocha94/household-webapp/internal/seed"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

func TestCreditCSVWorkflowAndDerivedStatus(t *testing.T) {
	t.Parallel()

	runner := seed.NewRunner(testutil.Logger(), auth.SeedModule{}, credits.SeedModule{})
	db := testutil.NewDB(t)

	seedDir := t.TempDir()
	writeCSV(t, filepath.Join(seedDir, "credit_contracts.csv"), [][]string{
		{"name", "category", "contract_number", "lender", "original_principal_cents", "start_month", "expected_end_month", "notes", "expected_day"},
		{"Home", "mortgage", "M-001", "Example Bank", "100000", "2026-01", "2030-12", "Main home", "5"},
	})
	writeCSV(t, filepath.Join(seedDir, "credit_payments.csv"), [][]string{
		{"credit", "month", "type", "principal_cents", "interest_cents", "description", "movement"},
		{"Home", "2026-01", "regular", "90000", "10000", "January instalment", ""},
		{"Home", "2026-01", "extraordinary", "10000", "0", "Extra payment", ""},
	})

	if err := runner.Import(db, seedDir); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	assertCreditState(t, db, "paid_off", 2, 100_000)

	exportDir := t.TempDir()
	if err := runner.Export(db, exportDir); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	assertCreditCSVRows(t, filepath.Join(exportDir, "credit_contracts.csv"), [][]string{
		{"Home", "mortgage", "M-001", "Example Bank", "100000", "2026-01", "2030-12", "Main home", "5"},
	})
	assertCreditCSVRows(t, filepath.Join(exportDir, "credit_payments.csv"), [][]string{
		{"Home", "2026-01", "extraordinary", "10000", "0", "Extra payment", ""},
		{"Home", "2026-01", "regular", "90000", "10000", "January instalment", ""},
	})

	writeCSV(t, filepath.Join(seedDir, "credit_payments.csv"), [][]string{
		{"credit", "month", "type", "principal_cents", "interest_cents", "description", "movement"},
		{"Home", "2026-02", "regular", "90000", "5000", "Corrected history", ""},
	})
	if err := runner.Import(db, seedDir); err != nil {
		t.Fatalf("Import(corrected history) error = %v", err)
	}
	assertCreditState(t, db, "active", 1, 90_000)
}

func assertCreditCSVRows(t *testing.T, path string, want [][]string) {
	t.Helper()
	var header []string
	switch filepath.Base(path) {
	case "credit_contracts.csv":
		header = []string{"name", "category", "contract_number", "lender", "original_principal_cents", "start_month", "expected_end_month", "notes", "expected_day"}
	case "credit_payments.csv":
		header = []string{"credit", "month", "type", "principal_cents", "interest_cents", "description", "movement"}
	default:
		t.Fatalf("no expected header for %s", path)
	}
	got, err := csvdata.Read(path, header)
	if err != nil {
		t.Fatalf("read exported CSV %s: %v", path, err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("exported CSV %s rows = %#v, want %#v", path, got, want)
	}
}

func assertCreditState(t *testing.T, db *sql.DB, wantStatus string, wantPayments int, wantPrincipal int64) {
	t.Helper()

	var status string
	var paymentCount int
	var principal int64
	err := db.QueryRow(
		`SELECT status, payment_count, total_principal_paid_cents
		 FROM credit_contract_summary_view WHERE name = 'Home'`,
	).Scan(&status, &paymentCount, &principal)
	if err != nil {
		t.Fatalf("query credit state: %v", err)
	}
	if status != wantStatus || paymentCount != wantPayments || principal != wantPrincipal {
		t.Fatalf(
			"credit state = (%q, %d, %d), want (%q, %d, %d)",
			status, paymentCount, principal, wantStatus, wantPayments, wantPrincipal,
		)
	}
}

func writeCSV(t *testing.T, path string, rows [][]string) {
	t.Helper()
	if err := csvdata.Write(path, rows); err != nil {
		t.Fatalf("write test CSV %s: %v", path, err)
	}
}

func TestPaymentKeepsItsLinkToACashflowMovement(t *testing.T) {
	t.Parallel()

	const link = "123e4567-e89b-12d3-a456-426614174000"
	db := testutil.NewDB(t)
	runner := seed.NewRunner(testutil.Logger(), credits.SeedModule{})
	dir := t.TempDir()
	writeCSV(t, filepath.Join(dir, "credit_contracts.csv"), [][]string{
		{"name", "category", "contract_number", "lender", "original_principal_cents", "start_month", "expected_end_month", "notes", "expected_day"},
		{"Home", "mortgage", "M-001", "Example Bank", "100000", "2026-01", "2030-12", "", "5"},
	})
	writeCSV(t, filepath.Join(dir, "credit_payments.csv"), [][]string{
		{"credit", "month", "type", "principal_cents", "interest_cents", "description", "movement"},
		{"Home", "2026-01", "regular", "9000", "1000", "", link},
		{"Home", "2026-02", "regular", "9000", "1000", "", ""},
	})
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	var got string
	if err := db.QueryRow("SELECT movement_sync_id FROM credit_payments WHERE movement_sync_id IS NOT NULL").Scan(&got); err != nil || got != link {
		t.Errorf("movement_sync_id = %q, %v; want %q", got, err, link)
	}

	exportDir := t.TempDir()
	if err := runner.Export(db, exportDir); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	assertCreditCSVRows(t, filepath.Join(exportDir, "credit_payments.csv"), [][]string{
		{"Home", "2026-01", "regular", "9000", "1000", "", link},
		{"Home", "2026-02", "regular", "9000", "1000", "", ""},
	})

	writeCSV(t, filepath.Join(dir, "credit_payments.csv"), [][]string{
		{"credit", "month", "type", "principal_cents", "interest_cents", "description", "movement"},
		{"Home", "2026-01", "regular", "9000", "1000", "", "not-a-sync-id"},
	})
	if err := runner.Import(db, dir); err == nil {
		t.Error("Import() with a bad link error = nil, want a rejection")
	}
}

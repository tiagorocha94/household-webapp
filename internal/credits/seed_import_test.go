package credits_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/credits"
	"github.com/tiagorocha94/household-webapp/internal/seed"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

func TestCreditImportIsIdempotentAndMirrorsCSV(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	runner := seed.NewRunner(testutil.Logger(), credits.SeedModule{})
	dir := t.TempDir()

	contractsHeader := []string{"name", "category", "contract_number", "lender", "original_principal_cents", "start_month", "expected_end_month", "notes", "expected_day"}
	paymentsHeader := []string{"credit", "month", "type", "principal_cents", "interest_cents", "description", "movement"}

	writeCSV(t, filepath.Join(dir, "credit_contracts.csv"), [][]string{
		contractsHeader,
		{"Home", "mortgage", "M-1", "Bank", "1000000", "2026-01", "", "", ""},
	})
	writeCSV(t, filepath.Join(dir, "credit_payments.csv"), [][]string{
		paymentsHeader,
		{"Home", "2026-01", "regular", "10000", "5000", "", ""},
		{"Home", "2026-02", "regular", "10000", "5000", "", ""},
	})

	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	const oldStamp = "2000-01-01 00:00:00"
	if _, err := db.Exec("UPDATE credit_contracts SET updated_at = ?", oldStamp); err != nil {
		t.Fatalf("set updated_at: %v", err)
	}
	paymentIDs := "SELECT group_concat(id) FROM (SELECT id FROM credit_payments ORDER BY id)"
	before := queryString(t, db, paymentIDs)

	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("second Import() error = %v", err)
	}
	if got := queryString(t, db, paymentIDs); got != before {
		t.Errorf("payment ids = %s, want %s", got, before)
	}
	if got := queryString(t, db, "SELECT CAST(updated_at AS TEXT) FROM credit_contracts"); got != oldStamp {
		t.Errorf("contract updated_at = %s, want untouched", got)
	}

	writeCSV(t, filepath.Join(dir, "credit_payments.csv"), [][]string{
		paymentsHeader,
		{"Home", "2026-01", "regular", "10000", "5000", "", ""},
	})
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("pruning Import() error = %v", err)
	}
	if got := queryString(t, db, "SELECT COUNT(*) FROM credit_payments"); got != "1" {
		t.Errorf("payments after prune = %s, want 1", got)
	}

	// Removing the contract and its payments clears both tables.
	writeCSV(t, filepath.Join(dir, "credit_payments.csv"), [][]string{paymentsHeader})
	writeCSV(t, filepath.Join(dir, "credit_contracts.csv"), [][]string{contractsHeader})
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("clearing Import() error = %v", err)
	}
	if got := queryString(t, db, "SELECT COUNT(*) FROM credit_contracts"); got != "0" {
		t.Errorf("contracts after empty file = %s, want 0", got)
	}
}

func queryString(t *testing.T, db *sql.DB, query string) string {
	t.Helper()
	var s string
	if err := db.QueryRow(query).Scan(&s); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return s
}

func TestCreditImportRejectsBadExpectedDay(t *testing.T) {
	t.Parallel()

	for name, day := range map[string]string{"too late": "29", "zero": "0", "negative": "-1", "not a number": "first"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			writeCSV(t, filepath.Join(dir, "credit_contracts.csv"), [][]string{
				{"name", "category", "contract_number", "lender", "original_principal_cents", "start_month", "expected_end_month", "notes", "expected_day"},
				{"Home", "mortgage", "M-1", "Bank", "1000000", "2026-01", "", "", day},
			})
			writeCSV(t, filepath.Join(dir, "credit_payments.csv"), [][]string{
				{"credit", "month", "type", "principal_cents", "interest_cents", "description", "movement"},
			})
			runner := seed.NewRunner(testutil.Logger(), credits.SeedModule{})
			if err := runner.Import(testutil.NewDB(t), dir); err == nil {
				t.Fatalf("Import() with expected_day %q error = nil, want a rejection", day)
			}
		})
	}
}

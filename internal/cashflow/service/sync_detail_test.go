package service

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/csvdata"
	"github.com/tiagorocha94/household-webapp/internal/csvstate"
	"github.com/tiagorocha94/household-webapp/internal/details"
	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/savings"
	savingsdetail "github.com/tiagorocha94/household-webapp/internal/savings/detail"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

var savingsMovementsHeader = []string{"account", "month", "type", "amount_cents", "description", "movement"}

// newDetailFixture adds a savings category and three savings accounts: one of
// alice, one of bob and one of the household.
func newDetailFixture(t *testing.T) (f *syncFixture, aliceAcc, bobAcc, houseAcc int64) {
	t.Helper()
	f = newSyncFixture(t)
	testutil.MustExec(t, f.db, `INSERT INTO cashflow_categories (name, direction, icon, color, role) VALUES ('rainy day', 'expense', 'piggy-bank', '#0EA5E9', 'savings')`)
	aliceAcc = testutil.MustExec(t, f.db, `INSERT INTO savings_accounts (name, user_id) VALUES ('Alice fund', ?)`, f.aliceID)
	bobAcc = testutil.MustExec(t, f.db, `INSERT INTO savings_accounts (name, user_id) VALUES ('Bob fund', ?)`, f.bobID)
	houseAcc = testutil.MustExec(t, f.db, `INSERT INTO savings_accounts (name) VALUES ('Household fund')`)
	f.svc.EnableDetails(details.NewRegistry(savingsdetail.Applier{}), csvstate.NewMirror(f.db, f.dir, savings.SeedModule{}))
	return f, aliceAcc, bobAcc, houseAcc
}

func rainy(n int, date string, cents int64, d *details.Detail) SyncOperation {
	return SyncOperation{Op: OpUpsert, SyncID: sid(n), Category: "rainy day", Date: date, AmountCents: money.Cents(cents), Detail: d}
}

func deposit(account int64) *details.Detail {
	return &details.Detail{Savings: &details.Savings{AccountID: account}}
}

// deposits returns the savings movements linked to a movement as
// account|month|type|cents.
func (f *syncFixture) deposits(t *testing.T, syncID string) []string {
	t.Helper()
	r, err := f.db.Query(
		`SELECT a.name, m.month, m.type, m.amount_cents FROM savings_movements m
		 JOIN savings_accounts a ON a.id = m.account_id
		 WHERE m.movement_sync_id = ? ORDER BY m.id`, syncID)
	if err != nil {
		t.Fatalf("query savings movements: %v", err)
	}
	defer r.Close()
	var out []string
	for r.Next() {
		var name, month, typ string
		var cents int64
		if err := r.Scan(&name, &month, &typ, &cents); err != nil {
			t.Fatalf("scan savings movement: %v", err)
		}
		out = append(out, strings.Join([]string{name, month, typ, fmt.Sprint(cents)}, "|"))
	}
	return out
}

func (f *syncFixture) savingsRevision(t *testing.T) int {
	t.Helper()
	var n int
	if err := f.db.QueryRow(`SELECT COALESCE((SELECT revision FROM module_csv_state WHERE module = 'savings'), 0)`).Scan(&n); err != nil {
		t.Fatalf("read savings revision: %v", err)
	}
	return n
}

func TestSyncWritesASavingsDepositWithTheMovement(t *testing.T) {
	f, aliceAcc, _, _ := newDetailFixture(t)

	f.requireOK(t, f.apply(t, f.aliceID, rainy(1, "2026-03-05", 25000, deposit(aliceAcc))))

	if got, want := strings.Join(f.deposits(t, sid(1)), ";"), "Alice fund|2026-03|deposit|25000"; got != want {
		t.Errorf("deposit = %q, want %q", got, want)
	}
	rows, err := csvdata.Read(filepath.Join(f.dir, "savings_movements.csv"), savingsMovementsHeader)
	if err != nil {
		t.Fatalf("read savings movements file: %v", err)
	}
	if len(rows) != 1 || rows[0][0] != "Alice fund" || rows[0][5] != sid(1) {
		t.Errorf("savings movements file = %v, want the deposit linked to %s", rows, sid(1))
	}
	var revision, written int
	if err := f.db.QueryRow(`SELECT revision, csv_revision FROM module_csv_state WHERE module = 'savings'`).Scan(&revision, &written); err != nil {
		t.Fatalf("read savings csv state: %v", err)
	}
	if revision != written {
		t.Errorf("savings csv revision = %d, file holds %d, want them equal", revision, written)
	}
}

func TestSyncSavingsDetailFollowsTheMovement(t *testing.T) {
	f, aliceAcc, _, houseAcc := newDetailFixture(t)
	f.requireOK(t, f.apply(t, f.aliceID, rainy(1, "2026-03-05", 25000, deposit(aliceAcc))))

	// A change without a detail moves the deposit with the movement.
	f.requireOK(t, f.apply(t, f.aliceID, rainy(1, "2026-04-02", 30000, nil)))
	if got, want := strings.Join(f.deposits(t, sid(1)), ";"), "Alice fund|2026-04|deposit|30000"; got != want {
		t.Errorf("after the change, deposit = %q, want %q", got, want)
	}

	// Another account replaces the first; it does not add a second deposit.
	f.requireOK(t, f.apply(t, f.aliceID, rainy(1, "2026-04-02", 30000, deposit(houseAcc))))
	if got, want := strings.Join(f.deposits(t, sid(1)), ";"), "Household fund|2026-04|deposit|30000"; got != want {
		t.Errorf("after moving it, deposit = %q, want %q", got, want)
	}

	// An empty detail removes it, and the movement stays.
	f.requireOK(t, f.apply(t, f.aliceID, rainy(1, "2026-04-02", 30000, &details.Detail{})))
	if got := f.deposits(t, sid(1)); len(got) != 0 {
		t.Errorf("after removing it, deposits = %v, want none", got)
	}
	if got := f.rows(t); len(got) != 1 {
		t.Errorf("movements = %v, want the movement kept", got)
	}
}

func TestSyncDeletingAMovementRemovesItsDeposit(t *testing.T) {
	f, aliceAcc, _, _ := newDetailFixture(t)
	f.requireOK(t, f.apply(t, f.aliceID, rainy(1, "2026-03-05", 25000, deposit(aliceAcc))))

	f.requireOK(t, f.apply(t, f.aliceID, SyncOperation{Op: OpDelete, SyncID: sid(1)}))

	if got := f.deposits(t, sid(1)); len(got) != 0 {
		t.Errorf("deposits = %v, want none after the delete", got)
	}
	rows, err := csvdata.Read(filepath.Join(f.dir, "savings_movements.csv"), savingsMovementsHeader)
	if err != nil {
		t.Fatalf("read savings movements file: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("savings movements file = %v, want no rows", rows)
	}
}

func TestSyncSavingsDetailIsIdempotent(t *testing.T) {
	f, aliceAcc, _, _ := newDetailFixture(t)
	value := money.Cents(120000)
	op := rainy(1, "2026-03-05", 25000, &details.Detail{Savings: &details.Savings{AccountID: aliceAcc, ValueCents: &value}})
	f.requireOK(t, f.apply(t, f.aliceID, op))
	before := f.savingsRevision(t)

	f.requireOK(t, f.apply(t, f.aliceID, op))

	if after := f.savingsRevision(t); after != before {
		t.Errorf("savings revision = %d after sending again, want %d", after, before)
	}
	var snapshots int
	if err := f.db.QueryRow(`SELECT COUNT(*) FROM savings_snapshots WHERE account_id = ? AND value_cents = 120000`, aliceAcc).Scan(&snapshots); err != nil {
		t.Fatalf("count snapshots: %v", err)
	}
	if snapshots != 1 {
		t.Errorf("snapshots = %d, want 1", snapshots)
	}
}

func TestSyncRefusesADetailAndKeepsTheMovementOut(t *testing.T) {
	f, aliceAcc, bobAcc, houseAcc := newDetailFixture(t)
	negative := money.Cents(-1)

	tests := []struct {
		name    string
		op      SyncOperation
		wantMsg string
	}{
		{"another person's account", rainy(1, "2026-03-05", 25000, deposit(bobAcc)), "belongs to another person"},
		{"unknown account", rainy(2, "2026-03-05", 25000, deposit(9999)), "does not exist"},
		{"negative value", rainy(3, "2026-03-05", 25000, &details.Detail{Savings: &details.Savings{AccountID: aliceAcc, ValueCents: &negative}}), "cannot be negative"},
		{"category without the role", food4(deposit(houseAcc)), "has no savings role"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := f.apply(t, f.aliceID, tt.op)
			if !strings.Contains(res[0].Rejected, tt.wantMsg) {
				t.Errorf("reason = %q, want it to contain %q", res[0].Rejected, tt.wantMsg)
			}
		})
	}
	if got := f.rows(t); len(got) != 0 {
		t.Errorf("movements = %v, want none: a refused detail refuses the movement", got)
	}
}

func food4(d *details.Detail) SyncOperation {
	op := food(4, "2026-03-05", 25000, "")
	op.Detail = d
	return op
}

func TestSyncAHouseholdAccountTakesMoneyFromAnyone(t *testing.T) {
	f, _, _, houseAcc := newDetailFixture(t)

	f.requireOK(t, f.apply(t, f.bobID, rainy(1, "2026-03-05", 25000, deposit(houseAcc))))

	if got, want := strings.Join(f.deposits(t, sid(1)), ";"), "Household fund|2026-03|deposit|25000"; got != want {
		t.Errorf("deposit = %q, want %q", got, want)
	}
}

func TestSavingsImportKeepsDeviceChangesWhileTheFilesAreBehind(t *testing.T) {
	f, aliceAcc, _, _ := newDetailFixture(t)
	f.requireOK(t, f.apply(t, f.aliceID, rainy(1, "2026-03-05", 25000, deposit(aliceAcc))))
	// A change the files do not have yet.
	if err := csvstate.Raise(f.db, "savings"); err != nil {
		t.Fatalf("Raise() error = %v", err)
	}

	tx, err := f.db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback()
	files := map[string][][]string{"savings_movements.csv": {}}
	if err := (savings.SeedModule{}).Import(tx, files, testutil.Logger()); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	if got := f.deposits(t, sid(1)); len(got) != 1 {
		t.Errorf("deposits = %v, want the deposit kept: an empty file must not undo device changes", got)
	}
}

// A module that does not run has no applier, so cashflow alone still takes
// movements, and a detail for the absent module is refused.
func TestSyncWithoutTheDetailModulesTakesMovementsButNotDetails(t *testing.T) {
	f := newSyncFixture(t)
	testutil.MustExec(t, f.db, `INSERT INTO cashflow_categories (name, direction, icon, color, role) VALUES ('rainy day', 'expense', 'piggy-bank', '#0EA5E9', 'savings')`)
	f.svc.EnableDetails(details.NewRegistry())

	f.requireOK(t, f.apply(t, f.aliceID, rainy(1, "2026-03-05", 25000, nil)))
	if got := f.rows(t); len(got) != 1 {
		t.Fatalf("movements = %v, want the one without a detail", got)
	}

	res := f.apply(t, f.aliceID, rainy(2, "2026-03-06", 1000, deposit(1)))
	if !strings.Contains(res[0].Rejected, "does not run") {
		t.Errorf("reason = %q, want it to say the module does not run", res[0].Rejected)
	}
	if got := f.rows(t); len(got) != 1 {
		t.Errorf("movements = %v, want the refused one left out", got)
	}
}

package service

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/csvdata"
	"github.com/tiagorocha94/household-webapp/internal/csvstate"
	"github.com/tiagorocha94/household-webapp/internal/details"
	"github.com/tiagorocha94/household-webapp/internal/investments"
	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

var contributionsHeader = []string{"entity", "instrument", "month", "amount_cents", "description", "movement"}

// investFixture has an investing category and three accounts: Alice's broker
// with two instruments, Bob's bank with one, and a household account with one.
type investFixture struct {
	*syncFixture
	aliceEntity, bobEntity, houseEntity int64
	etf, fund, bobFund, houseFund       int64
}

func newInvestFixture(t *testing.T) *investFixture {
	t.Helper()
	f := newSyncFixture(t)
	testutil.MustExec(t, f.db, `INSERT INTO cashflow_categories (name, direction, icon, color, role) VALUES ('investing', 'expense', 'chart', '#0EA5E9', 'investments')`)
	inv := &investFixture{syncFixture: f}
	inv.aliceEntity = testutil.MustExec(t, f.db, `INSERT INTO investment_entities (name, kind, user_id) VALUES ('Alice Broker', 'broker', ?)`, f.aliceID)
	inv.bobEntity = testutil.MustExec(t, f.db, `INSERT INTO investment_entities (name, kind, user_id) VALUES ('Bob Bank', 'bank', ?)`, f.bobID)
	inv.houseEntity = testutil.MustExec(t, f.db, `INSERT INTO investment_entities (name, kind) VALUES ('Household Bank', 'bank')`)
	inv.etf = testutil.MustExec(t, f.db, `INSERT INTO investment_instruments (entity_id, name, type) VALUES (?, 'World ETF', 'etf')`, inv.aliceEntity)
	inv.fund = testutil.MustExec(t, f.db, `INSERT INTO investment_instruments (entity_id, name, type) VALUES (?, 'Bond fund', 'fund')`, inv.aliceEntity)
	inv.bobFund = testutil.MustExec(t, f.db, `INSERT INTO investment_instruments (entity_id, name, type) VALUES (?, 'Bob fund', 'fund')`, inv.bobEntity)
	inv.houseFund = testutil.MustExec(t, f.db, `INSERT INTO investment_instruments (entity_id, name, type) VALUES (?, 'House fund', 'fund')`, inv.houseEntity)
	f.svc.EnableDetailMirrors(csvstate.NewMirror(f.db, f.dir, investments.SeedModule{}))
	return inv
}

func investing(n int, date string, cents int64, d *details.Detail) SyncOperation {
	return SyncOperation{Op: OpUpsert, SyncID: sid(n), Category: "investing", Date: date, AmountCents: money.Cents(cents), Detail: d}
}

func split(entity int64, lines ...details.InvestmentLine) *details.Detail {
	return &details.Detail{Investments: &details.Investments{EntityID: entity, Lines: lines}}
}

func line(instrument, cents int64) details.InvestmentLine {
	return details.InvestmentLine{InstrumentID: instrument, AmountCents: money.Cents(cents)}
}

// contributions returns the contributions linked to a movement as
// instrument|month|cents.
func (f *syncFixture) contributions(t *testing.T, syncID string) []string {
	t.Helper()
	r, err := f.db.Query(
		`SELECT i.name, c.month, c.amount_cents FROM investment_contributions c
		 JOIN investment_instruments i ON i.id = c.instrument_id
		 WHERE c.movement_sync_id = ? ORDER BY i.name`, syncID)
	if err != nil {
		t.Fatalf("query contributions: %v", err)
	}
	defer r.Close()
	var out []string
	for r.Next() {
		var name, month string
		var cents int64
		if err := r.Scan(&name, &month, &cents); err != nil {
			t.Fatalf("scan contribution: %v", err)
		}
		out = append(out, strings.Join([]string{name, month, fmt.Sprint(cents)}, "|"))
	}
	return out
}

func (f *syncFixture) investmentsRevision(t *testing.T) int {
	t.Helper()
	var n int
	if err := f.db.QueryRow(`SELECT COALESCE((SELECT revision FROM module_csv_state WHERE module = 'investments'), 0)`).Scan(&n); err != nil {
		t.Fatalf("read investments revision: %v", err)
	}
	return n
}

func TestSyncSplitsAMovementAcrossInstruments(t *testing.T) {
	f := newInvestFixture(t)

	f.requireOK(t, f.apply(t, f.aliceID, investing(1, "2026-03-05", 50000, split(f.aliceEntity, line(f.etf, 30000), line(f.fund, 20000)))))

	if got, want := strings.Join(f.contributions(t, sid(1)), ";"), "Bond fund|2026-03|20000;World ETF|2026-03|30000"; got != want {
		t.Errorf("contributions = %q, want %q", got, want)
	}
	rows, err := csvdata.Read(filepath.Join(f.dir, "investment_contributions.csv"), contributionsHeader)
	if err != nil {
		t.Fatalf("read contributions file: %v", err)
	}
	if len(rows) != 2 || rows[0][0] != "Alice Broker" || rows[0][5] != sid(1) {
		t.Errorf("contributions file = %v, want two rows linked to %s", rows, sid(1))
	}
}

func TestSyncKeepsAPartialSplitAndReplacesItOnEdit(t *testing.T) {
	f := newInvestFixture(t)

	// Less than the movement is a split that is still open.
	f.requireOK(t, f.apply(t, f.aliceID, investing(1, "2026-03-05", 50000, split(f.aliceEntity, line(f.etf, 30000)))))
	if got, want := strings.Join(f.contributions(t, sid(1)), ";"), "World ETF|2026-03|30000"; got != want {
		t.Errorf("partial contributions = %q, want %q", got, want)
	}

	// Sending the lines again replaces them; it does not add to them.
	f.requireOK(t, f.apply(t, f.aliceID, investing(1, "2026-03-05", 50000, split(f.aliceEntity, line(f.etf, 10000), line(f.fund, 40000)))))
	if got, want := strings.Join(f.contributions(t, sid(1)), ";"), "Bond fund|2026-03|40000;World ETF|2026-03|10000"; got != want {
		t.Errorf("replaced contributions = %q, want %q", got, want)
	}

	// A change without a detail moves the contributions to the new month.
	f.requireOK(t, f.apply(t, f.aliceID, investing(1, "2026-04-02", 60000, nil)))
	if got, want := strings.Join(f.contributions(t, sid(1)), ";"), "Bond fund|2026-04|40000;World ETF|2026-04|10000"; got != want {
		t.Errorf("moved contributions = %q, want %q", got, want)
	}

	// An empty detail removes them and keeps the movement.
	f.requireOK(t, f.apply(t, f.aliceID, investing(1, "2026-04-02", 60000, &details.Detail{})))
	if got := f.contributions(t, sid(1)); len(got) != 0 {
		t.Errorf("contributions = %v, want none after removing the detail", got)
	}
	if got := f.rows(t); len(got) != 1 {
		t.Errorf("movements = %v, want the movement kept", got)
	}
}

func TestSyncRefusesAnEditThatMakesTheSplitTooBig(t *testing.T) {
	f := newInvestFixture(t)
	f.requireOK(t, f.apply(t, f.aliceID, investing(1, "2026-03-05", 50000, split(f.aliceEntity, line(f.etf, 50000)))))

	res := f.apply(t, f.aliceID, investing(1, "2026-03-05", 40000, nil))

	if !strings.Contains(res[0].Rejected, "more than the movement") {
		t.Errorf("reason = %q, want it to say the split is more than the movement", res[0].Rejected)
	}
	if got, want := strings.Join(f.rows(t), ""), "alice|investing|2026-03-05|50000||"+sid(1); got != want {
		t.Errorf("movement = %q, want %q unchanged", got, want)
	}
}

func TestSyncDeletingAMovementRemovesItsContributions(t *testing.T) {
	f := newInvestFixture(t)
	f.requireOK(t, f.apply(t, f.aliceID, investing(1, "2026-03-05", 50000, split(f.aliceEntity, line(f.etf, 50000)))))

	f.requireOK(t, f.apply(t, f.aliceID, SyncOperation{Op: OpDelete, SyncID: sid(1)}))

	if got := f.contributions(t, sid(1)); len(got) != 0 {
		t.Errorf("contributions = %v, want none after the delete", got)
	}
}

func TestSyncInvestmentsDetailIsIdempotent(t *testing.T) {
	f := newInvestFixture(t)
	value := money.Cents(123000)
	op := investing(1, "2026-03-05", 50000, split(f.aliceEntity,
		details.InvestmentLine{InstrumentID: f.etf, AmountCents: 50000, ValueCents: &value}))
	f.requireOK(t, f.apply(t, f.aliceID, op))
	before := f.investmentsRevision(t)

	f.requireOK(t, f.apply(t, f.aliceID, op))

	if after := f.investmentsRevision(t); after != before {
		t.Errorf("investments revision = %d after sending again, want %d", after, before)
	}
	var snapshots int
	if err := f.db.QueryRow(`SELECT COUNT(*) FROM investment_snapshots WHERE instrument_id = ?`, f.etf).Scan(&snapshots); err != nil {
		t.Fatalf("count snapshots: %v", err)
	}
	if snapshots != 1 {
		t.Errorf("snapshots = %d, want 1", snapshots)
	}
}

func TestSyncRefusesAnInvestmentsDetail(t *testing.T) {
	f := newInvestFixture(t)
	negative := money.Cents(-1)

	tests := []struct {
		name    string
		op      SyncOperation
		wantMsg string
	}{
		{"another person's account", investing(1, "2026-03-05", 50000, split(f.bobEntity, line(f.bobFund, 1000))), "belongs to another person"},
		{"unknown account", investing(2, "2026-03-05", 50000, split(9999, line(f.etf, 1000))), "does not exist"},
		{"unknown instrument", investing(3, "2026-03-05", 50000, split(f.aliceEntity, line(9999, 1000))), "does not exist"},
		{"instrument of another account", investing(4, "2026-03-05", 50000, split(f.aliceEntity, line(f.houseFund, 1000))), "is not in investment account"},
		{"more than the movement", investing(5, "2026-03-05", 50000, split(f.aliceEntity, line(f.etf, 30000), line(f.fund, 30000))), "more than the movement"},
		{"zero amount", investing(6, "2026-03-05", 50000, split(f.aliceEntity, line(f.etf, 0))), "above zero"},
		{"same instrument twice", investing(7, "2026-03-05", 50000, split(f.aliceEntity, line(f.etf, 100), line(f.etf, 100))), "more than once"},
		{"negative value", investing(8, "2026-03-05", 50000, split(f.aliceEntity, details.InvestmentLine{InstrumentID: f.etf, AmountCents: 100, ValueCents: &negative})), "cannot be negative"},
		{"category without the role", func() SyncOperation {
			op := food(9, "2026-03-05", 50000, "")
			op.Detail = split(f.aliceEntity, line(f.etf, 100))
			return op
		}(), "has no investments role"},
		{"savings and investments together", investing(10, "2026-03-05", 50000, &details.Detail{
			Savings:     &details.Savings{AccountID: 1},
			Investments: &details.Investments{EntityID: f.aliceEntity},
		}), "not several"},
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

func TestSyncAHouseholdInvestmentAccountTakesMoneyFromAnyone(t *testing.T) {
	f := newInvestFixture(t)

	f.requireOK(t, f.apply(t, f.bobID, investing(1, "2026-03-05", 50000, split(f.houseEntity, line(f.houseFund, 50000)))))

	if got, want := strings.Join(f.contributions(t, sid(1)), ";"), "House fund|2026-03|50000"; got != want {
		t.Errorf("contributions = %q, want %q", got, want)
	}
}

func TestInvestmentsExportImportRoundTrip(t *testing.T) {
	f := newInvestFixture(t)
	f.requireOK(t, f.apply(t, f.aliceID, investing(1, "2026-03-05", 50000, split(f.aliceEntity, line(f.etf, 30000), line(f.fund, 20000)))))

	tables, err := (investments.SeedModule{}).Export(f.db)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	files := make(map[string][][]string, len(tables))
	for _, tb := range tables {
		files[tb.Name] = tb.Rows[1:]
	}

	// A fresh database gets everything back from the exported files alone.
	other := testutil.NewDB(t)
	testutil.InsertUser(t, other, "alice")
	testutil.InsertUser(t, other, "bob")
	tx, err := other.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback()
	if err := (investments.SeedModule{}).Import(tx, files, testutil.Logger()); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	again := &syncFixture{db: other}
	if got, want := strings.Join(again.contributions(t, sid(1)), ";"), "Bond fund|2026-03|20000;World ETF|2026-03|30000"; got != want {
		t.Errorf("contributions after the round trip = %q, want %q", got, want)
	}
}

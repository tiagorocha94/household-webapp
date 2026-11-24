package service

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/credits"
	creditsdetail "github.com/tiagorocha94/household-webapp/internal/credits/detail"
	"github.com/tiagorocha94/household-webapp/internal/csvdata"
	"github.com/tiagorocha94/household-webapp/internal/csvstate"
	"github.com/tiagorocha94/household-webapp/internal/details"
	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

var creditPaymentsHeader = []string{"credit", "month", "type", "principal_cents", "interest_cents", "description", "movement"}

// newCreditFixture adds a credits category and a credit that started in 2026-01.
func newCreditFixture(t *testing.T) (f *syncFixture, creditID int64) {
	t.Helper()
	f = newSyncFixture(t)
	testutil.MustExec(t, f.db, `INSERT INTO cashflow_categories (name, direction, icon, color, role) VALUES ('loan', 'expense', 'landmark', '#0EA5E9', 'credits')`)
	creditID = testutil.MustExec(t, f.db, `INSERT INTO credit_contracts
		(name, category, contract_number, lender, original_principal_cents, start_month)
		VALUES ('Car loan', 'car', 'C-1', 'Bank', 1000000, '2026-01')`)
	f.svc.EnableDetails(details.NewRegistry(creditsdetail.Applier{}), csvstate.NewMirror(f.db, f.dir, credits.SeedModule{}))
	return f, creditID
}

func loan(n int, date string, cents int64, d *details.Detail) SyncOperation {
	return SyncOperation{Op: OpUpsert, SyncID: sid(n), Category: "loan", Date: date, AmountCents: money.Cents(cents), Detail: d}
}

func payment(credit, principal, interest int64, extraordinary bool) *details.Detail {
	return &details.Detail{Credit: &details.Credit{
		CreditID: credit, PrincipalCents: money.Cents(principal), InterestCents: money.Cents(interest), Extraordinary: extraordinary,
	}}
}

// payments returns the credit payments linked to a movement as
// credit|month|type|principal|interest.
func (f *syncFixture) payments(t *testing.T, syncID string) []string {
	t.Helper()
	r, err := f.db.Query(
		`SELECT c.name, p.month, p.type, p.principal_cents, p.interest_cents FROM credit_payments p
		 JOIN credit_contracts c ON c.id = p.contract_id
		 WHERE p.movement_sync_id = ? ORDER BY p.id`, syncID)
	if err != nil {
		t.Fatalf("query credit payments: %v", err)
	}
	defer r.Close()
	var out []string
	for r.Next() {
		var name, month, kind string
		var principal, interest int64
		if err := r.Scan(&name, &month, &kind, &principal, &interest); err != nil {
			t.Fatalf("scan credit payment: %v", err)
		}
		out = append(out, strings.Join([]string{name, month, kind, fmt.Sprint(principal), fmt.Sprint(interest)}, "|"))
	}
	return out
}

func TestSyncWritesACreditPaymentWithTheMovement(t *testing.T) {
	f, credit := newCreditFixture(t)

	f.requireOK(t, f.apply(t, f.aliceID, loan(1, "2026-03-05", 30000, payment(credit, 25000, 5000, false))))

	if got, want := strings.Join(f.payments(t, sid(1)), ";"), "Car loan|2026-03|regular|25000|5000"; got != want {
		t.Errorf("payment = %q, want %q", got, want)
	}
	rows, err := csvdata.Read(filepath.Join(f.dir, "credit_payments.csv"), creditPaymentsHeader)
	if err != nil {
		t.Fatalf("read credit payments file: %v", err)
	}
	if len(rows) != 1 || rows[0][0] != "Car loan" || rows[0][6] != sid(1) {
		t.Errorf("credit payments file = %v, want the payment linked to %s", rows, sid(1))
	}
}

func TestSyncCreditPaymentFollowsTheMovement(t *testing.T) {
	f, credit := newCreditFixture(t)
	f.requireOK(t, f.apply(t, f.aliceID, loan(1, "2026-03-05", 30000, payment(credit, 25000, 5000, false))))

	// A new month without a detail moves the payment.
	f.requireOK(t, f.apply(t, f.aliceID, loan(1, "2026-04-02", 30000, nil)))
	if got, want := strings.Join(f.payments(t, sid(1)), ";"), "Car loan|2026-04|regular|25000|5000"; got != want {
		t.Errorf("moved payment = %q, want %q", got, want)
	}

	// Sending the detail replaces the payment; it does not add a second one.
	f.requireOK(t, f.apply(t, f.aliceID, loan(1, "2026-04-02", 40000, payment(credit, 40000, 0, true))))
	if got, want := strings.Join(f.payments(t, sid(1)), ";"), "Car loan|2026-04|extraordinary|40000|0"; got != want {
		t.Errorf("replaced payment = %q, want %q", got, want)
	}

	// An empty detail removes it and keeps the movement.
	f.requireOK(t, f.apply(t, f.aliceID, loan(1, "2026-04-02", 40000, &details.Detail{})))
	if got := f.payments(t, sid(1)); len(got) != 0 {
		t.Errorf("payments = %v, want none", got)
	}
	if got := f.rows(t); len(got) != 1 {
		t.Errorf("movements = %v, want the movement kept", got)
	}
}

func TestSyncDeletingAMovementRemovesItsCreditPayment(t *testing.T) {
	f, credit := newCreditFixture(t)
	f.requireOK(t, f.apply(t, f.aliceID, loan(1, "2026-03-05", 30000, payment(credit, 25000, 5000, false))))

	f.requireOK(t, f.apply(t, f.aliceID, SyncOperation{Op: OpDelete, SyncID: sid(1)}))

	if got := f.payments(t, sid(1)); len(got) != 0 {
		t.Errorf("payments = %v, want none after the delete", got)
	}
}

func TestSyncRefusesACreditPayment(t *testing.T) {
	f, credit := newCreditFixture(t)
	f.requireOK(t, f.apply(t, f.aliceID, loan(1, "2026-03-05", 30000, payment(credit, 25000, 5000, false))))

	tests := []struct {
		name    string
		op      SyncOperation
		wantMsg string
	}{
		{"parts that do not add up", loan(2, "2026-04-05", 30000, payment(credit, 20000, 5000, false)), "add up to 25000"},
		{"negative part", loan(3, "2026-04-05", 30000, payment(credit, 40000, -10000, false)), "cannot be negative"},
		{"unknown credit", loan(4, "2026-04-05", 30000, payment(9999, 25000, 5000, false)), "does not exist"},
		{"before the credit starts", loan(5, "2025-12-05", 30000, payment(credit, 25000, 5000, false)), "before the credit starts"},
		{"second regular payment in a month", loan(6, "2026-03-20", 30000, payment(credit, 25000, 5000, false)), "already has a regular payment"},
		{"category without the role", func() SyncOperation {
			op := food(7, "2026-04-05", 30000, "")
			op.Detail = payment(credit, 25000, 5000, false)
			return op
		}(), "has no credits role"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := f.apply(t, f.aliceID, tt.op)
			if !strings.Contains(res[0].Rejected, tt.wantMsg) {
				t.Errorf("reason = %q, want it to contain %q", res[0].Rejected, tt.wantMsg)
			}
		})
	}
	if got := f.rows(t); len(got) != 1 {
		t.Errorf("movements = %v, want only the first: a refused detail refuses the movement", got)
	}
}

func TestSyncAnExtraordinaryCreditPaymentCanShareAMonth(t *testing.T) {
	f, credit := newCreditFixture(t)
	f.requireOK(t, f.apply(t, f.aliceID, loan(1, "2026-03-05", 30000, payment(credit, 25000, 5000, false))))

	f.requireOK(t, f.apply(t, f.bobID, loan(2, "2026-03-20", 100000, payment(credit, 100000, 0, true))))
}

func TestSyncRefusesAnEditThatBreaksTheCreditPayment(t *testing.T) {
	f, credit := newCreditFixture(t)
	f.requireOK(t, f.apply(t, f.aliceID, loan(1, "2026-03-05", 30000, payment(credit, 25000, 5000, false))))

	res := f.apply(t, f.aliceID, loan(1, "2026-03-05", 31000, nil))

	if !strings.Contains(res[0].Rejected, "send the detail again") {
		t.Errorf("reason = %q, want it to ask for the detail again", res[0].Rejected)
	}
}

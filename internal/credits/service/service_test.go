package service

import (
	"errors"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/credits"
	"github.com/tiagorocha94/household-webapp/internal/credits/store"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

func TestReadTotalsAndNotFound(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	service := New(store.New(db))

	contractID := testutil.MustExec(t, db,
		`INSERT INTO credit_contracts (name, category, contract_number, lender, original_principal_cents, start_month)
		 VALUES ('Home', 'mortgage', 'M-1', 'Bank', 10000000, '2026-01')`)
	testutil.MustExec(t, db,
		`INSERT INTO credit_payments (contract_id, month, type, principal_cents, interest_cents)
		 VALUES (?, '2026-01', 'regular', 65000, 35000)`, contractID)
	testutil.MustExec(t, db,
		`INSERT INTO credit_payments (contract_id, month, type, principal_cents, interest_cents)
		 VALUES (?, '2026-01', 'extraordinary', 100000, 0)`, contractID)

	summary, err := service.GetContractSummary(contractID)
	if err != nil {
		t.Fatalf("GetContractSummary() error = %v", err)
	}
	if summary.PaymentCount != 2 || summary.TotalPrincipalPaidCents != 165_000 ||
		summary.TotalInterestPaidCents != 35_000 || summary.TotalPaidCents != 200_000 ||
		summary.OutstandingPrincipalCents != 9_835_000 {
		t.Fatalf("GetContractSummary() totals = %#v", summary)
	}
	list, err := service.ListContracts()
	if err != nil || len(list) != 1 {
		t.Fatalf("ListContracts() = %v, %v, want one contract", list, err)
	}
	payments, err := service.ListPayments(contractID)
	if err != nil || len(payments) != 2 {
		t.Fatalf("ListPayments() = %v, %v, want two payments", payments, err)
	}

	if _, err := service.GetContractSummary(9999); !errors.Is(err, apperr.ErrNotFound) {
		t.Errorf("GetContractSummary(missing) error = %v, want not found", err)
	}
	if _, err := service.ListPayments(9999); !errors.Is(err, apperr.ErrNotFound) {
		t.Errorf("ListPayments(missing) error = %v, want not found", err)
	}
}

func TestStatusIsDerivedFromPayments(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	service := New(store.New(db))
	contractID := testutil.MustExec(t, db,
		`INSERT INTO credit_contracts (name, category, contract_number, lender, original_principal_cents, start_month)
		 VALUES ('Car', 'car', 'C-1', 'Bank', 100000, '2026-01')`)

	assertStatus := func(want string) {
		t.Helper()
		summary, err := service.GetContractSummary(contractID)
		if err != nil || summary.Status != want {
			t.Fatalf("status = %v, %v, want %q", summary, err, want)
		}
	}

	assertStatus("active")
	paymentID := testutil.MustExec(t, db,
		`INSERT INTO credit_payments (contract_id, month, type, principal_cents, interest_cents)
		 VALUES (?, '2026-01', 'regular', 100000, 0)`, contractID)
	assertStatus("paid_off")
	testutil.MustExec(t, db, "DELETE FROM credit_payments WHERE id = ?", paymentID)
	assertStatus("active")
}

func TestListDue(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	service := New(store.New(db))
	insert := func(name string, day any, start string) int64 {
		return testutil.MustExec(t, db,
			`INSERT INTO credit_contracts (name, category, contract_number, lender, original_principal_cents, start_month, expected_day)
			 VALUES (?, 'car', ?, 'Bank', 1000000, ?, ?)`, name, name, start, day)
	}
	pay := func(id int64, month, kind string, principal, interest int) {
		testutil.MustExec(t, db,
			`INSERT INTO credit_payments (contract_id, month, type, principal_cents, interest_cents) VALUES (?, ?, ?, ?, ?)`,
			id, month, kind, principal, interest)
	}

	due := insert("due", 5, "2026-01")
	pay(due, "2026-01", "regular", 60000, 10000)
	pay(due, "2026-02", "regular", 61000, 9000)
	pay(due, "2026-02", "extraordinary", 500000, 0)
	paid := insert("paid", 5, "2026-01")
	pay(paid, "2026-03", "regular", 60000, 10000)
	insert("no day", nil, "2026-01")
	insert("not started", 5, "2026-06")
	// An extraordinary payment alone does not settle the month.
	onlyExtra := insert("only extra", 10, "2026-01")
	pay(onlyExtra, "2026-03", "extraordinary", 10000, 0)

	got, err := service.ListDue("2026-03")
	if err != nil {
		t.Fatalf("ListDue() error = %v", err)
	}
	names := map[string]credits.DueCredit{}
	for _, d := range got {
		names[d.Name] = d
	}
	if len(got) != 2 || names["due"].ContractID == 0 || names["only extra"].ContractID == 0 {
		t.Fatalf("ListDue() = %v, want only %q and %q", got, "due", "only extra")
	}
	if d := names["due"]; d.PrincipalCents != 61000 || d.InterestCents != 9000 || d.ExpectedDay != 5 || d.Month != "2026-03" {
		t.Errorf("due credit = %+v, want the instalment of the last regular payment (61000 + 9000), day 5, month 2026-03", d)
	}
	if d := names["only extra"]; d.PrincipalCents != 0 || d.InterestCents != 0 {
		t.Errorf("credit without a regular payment = %+v, want no instalment", d)
	}
}

package service

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/cashflow"
	"github.com/tiagorocha94/household-webapp/internal/cashflow/store"
	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

func TestGetMonthTotalsAndErrors(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	service := New(store.New(db))

	aliceID := testutil.InsertUser(t, db, "alice")
	salaryID := testutil.MustExec(t, db,
		`INSERT INTO cashflow_categories (name, direction, icon, color) VALUES ('salary', 'income', 'briefcase', '#16A34A')`)
	foodID := testutil.MustExec(t, db,
		`INSERT INTO cashflow_categories (name, direction, icon, color) VALUES ('food', 'expense', 'utensils', '#E76F51')`)
	for i, m := range []struct {
		categoryID int64
		date       string
		cents      int64
	}{
		{salaryID, "2026-03-01", 300_000},
		{foodID, "2026-03-02", 10_000},
		{foodID, "2026-03-02", 10_000},
	} {
		testutil.MustExec(t, db,
			`INSERT INTO cashflow_movements (sync_id, user_id, category_id, date, amount_cents) VALUES (?, ?, ?, ?, ?)`,
			fmt.Sprintf("00000000-0000-4000-8000-%012d", i+1), aliceID, m.categoryID, m.date, m.cents)
	}

	detail, err := service.GetMonth("2026-03")
	if err != nil {
		t.Fatalf("GetMonth() error = %v", err)
	}
	if detail.IncomeCents != 300_000 || detail.ExpenseCents != 20_000 || detail.NetCents != 280_000 ||
		detail.SavingsRatePct != 9333 || detail.MovementCount != 3 {
		t.Fatalf("GetMonth() summary = %#v", detail.MonthSummary)
	}
	if len(detail.ByCategory) != 2 || len(detail.ByUser) != 1 || detail.ByUser[0].Username != "alice" {
		t.Fatalf("GetMonth() byCategory/byUser = %#v / %#v", detail.ByCategory, detail.ByUser)
	}
	if len(detail.ByGroup) != 0 {
		t.Errorf("GetMonth() byGroup = %#v, want none while no groups exist", detail.ByGroup)
	}

	months, err := service.ListMonths()
	if err != nil || len(months) != 1 {
		t.Fatalf("ListMonths() = %v, %v, want one month", months, err)
	}
	movements, err := service.ListMonthMovements("2026-03")
	if err != nil || len(movements) != 3 {
		t.Fatalf("ListMonthMovements() = %v, %v, want three movements", movements, err)
	}

	if _, err := service.GetMonth("2026-04"); !errors.Is(err, apperr.ErrNotFound) {
		t.Errorf("GetMonth(empty month) error = %v, want not found", err)
	}
	empty, err := service.ListMonthMovements("2026-04")
	if err != nil || len(empty) != 0 {
		t.Errorf("ListMonthMovements(empty month) = %v, %v, want no movements and no error", empty, err)
	}
	for _, bad := range []string{"", "2026", "2026-3", "2026-13", "2026-03-01"} {
		if _, err := service.GetMonth(bad); !errors.Is(err, apperr.ErrBadRequest) {
			t.Errorf("GetMonth(%q) error = %v, want bad request", bad, err)
		}
		if _, err := service.ListMonthMovements(bad); !errors.Is(err, apperr.ErrBadRequest) {
			t.Errorf("ListMonthMovements(%q) error = %v, want bad request", bad, err)
		}
	}
}

func TestGetMonthBudgetByGroup(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	service := New(store.New(db))

	aliceID := testutil.InsertUser(t, db, "alice")
	essentialID := testutil.MustExec(t, db,
		`INSERT INTO cashflow_budget_groups (name, share_pct, position) VALUES ('Essential', 60, 1)`)
	testutil.MustExec(t, db,
		`INSERT INTO cashflow_budget_groups (name, share_pct, position) VALUES ('Fun', 40, 2)`)
	salaryID := testutil.MustExec(t, db,
		`INSERT INTO cashflow_categories (name, direction, icon, color) VALUES ('salary', 'income', 'briefcase', '#16A34A')`)
	rentID := testutil.MustExec(t, db,
		`INSERT INTO cashflow_categories (name, direction, icon, color, budget_group_id) VALUES ('rent', 'expense', 'house', '#E76F51', ?)`, essentialID)
	foodID := testutil.MustExec(t, db,
		`INSERT INTO cashflow_categories (name, direction, icon, color, budget_group_id) VALUES ('food', 'expense', 'utensils', '#E76F51', ?)`, essentialID)
	for i, m := range []struct {
		categoryID int64
		date       string
		cents      int64
	}{
		{salaryID, "2026-03-01", 100_001},
		{rentID, "2026-03-02", 40_000},
		{foodID, "2026-03-03", 30_000},
		// No income in April: the budget is zero but spending still shows.
		{foodID, "2026-04-03", 5_000},
	} {
		testutil.MustExec(t, db,
			`INSERT INTO cashflow_movements (sync_id, user_id, category_id, date, amount_cents) VALUES (?, ?, ?, ?, ?)`,
			fmt.Sprintf("00000000-0000-4000-8000-%012d", i+1), aliceID, m.categoryID, m.date, m.cents)
	}

	detail, err := service.GetMonth("2026-03")
	if err != nil {
		t.Fatalf("GetMonth() error = %v", err)
	}
	if len(detail.ByGroup) != 2 {
		t.Fatalf("GetMonth() byGroup = %#v, want two groups", detail.ByGroup)
	}
	essential, fun := detail.ByGroup[0], detail.ByGroup[1]
	// 60% of 1000.01 is 600.006 and 40% is 400.004: both round down and the
	// leftover cent goes to the largest group.
	if essential.Name != "Essential" || essential.SharePct != 60 || essential.BudgetCents != 60_001 ||
		essential.SpentCents != 70_000 || essential.LeftCents != -9_999 || len(essential.Categories) != 2 ||
		essential.Categories[0].Name != "rent" {
		t.Errorf("Essential = %#v", essential)
	}
	if fun.Name != "Fun" || fun.BudgetCents != 40_000 || fun.SpentCents != 0 || fun.LeftCents != 40_000 ||
		fun.Categories == nil || len(fun.Categories) != 0 {
		t.Errorf("Fun = %#v", fun)
	}

	april, err := service.GetMonth("2026-04")
	if err != nil {
		t.Fatalf("GetMonth(april) error = %v", err)
	}
	if got := april.ByGroup[0]; got.BudgetCents != 0 || got.SpentCents != 5_000 || got.LeftCents != -5_000 {
		t.Errorf("April Essential = %#v, want no budget and 50,00 spent", got)
	}
}

func TestGetBudgetForAMonthWithAndWithoutMovements(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	service := New(store.New(db))

	aliceID := testutil.InsertUser(t, db, "alice")
	groupID := testutil.MustExec(t, db,
		`INSERT INTO cashflow_budget_groups (name, share_pct, position) VALUES ('Essential', 100, 1)`)
	salaryID := testutil.MustExec(t, db,
		`INSERT INTO cashflow_categories (name, direction, icon, color) VALUES ('salary', 'income', 'briefcase', '#16A34A')`)
	foodID := testutil.MustExec(t, db,
		`INSERT INTO cashflow_categories (name, direction, icon, color, budget_group_id) VALUES ('food', 'expense', 'utensils', '#E76F51', ?)`, groupID)
	for i, m := range []struct {
		categoryID int64
		date       string
		cents      int64
	}{
		{salaryID, "2026-03-01", 100_000},
		{foodID, "2026-03-03", 30_000},
		{foodID, "2026-04-03", 5_000},
	} {
		testutil.MustExec(t, db,
			`INSERT INTO cashflow_movements (sync_id, user_id, category_id, date, amount_cents) VALUES (?, ?, ?, ?, ?)`,
			fmt.Sprintf("00000000-0000-4000-8000-%012d", i+1), aliceID, m.categoryID, m.date, m.cents)
	}

	march, err := service.GetBudget("2026-03")
	if err != nil {
		t.Fatalf("GetBudget(march) error = %v", err)
	}
	if march.IncomeCents != 100_000 || march.ExpenseCents != 30_000 || march.NetCents != 70_000 ||
		len(march.ByGroup) != 1 || march.ByGroup[0].BudgetCents != 100_000 || march.ByGroup[0].SpentCents != 30_000 {
		t.Errorf("GetBudget(march) = %#v", march)
	}
	if want := []string{"2026-04", "2026-03"}; !slices.Equal(march.Months, want) {
		t.Errorf("GetBudget(march) months = %v, want %v", march.Months, want)
	}

	// No income in April: no plan, but the spending still shows.
	april, err := service.GetBudget("2026-04")
	if err != nil {
		t.Fatalf("GetBudget(april) error = %v", err)
	}
	if april.IncomeCents != 0 || april.ByGroup[0].BudgetCents != 0 || april.ByGroup[0].SpentCents != 5_000 {
		t.Errorf("GetBudget(april) = %#v, want no budget and 50,00 spent", april)
	}

	// A month without movements is empty, not missing, and still lists the groups.
	may, err := service.GetBudget("2026-05")
	if err != nil {
		t.Fatalf("GetBudget(may) error = %v, want none", err)
	}
	if may.Month != "2026-05" || may.IncomeCents != 0 || may.ExpenseCents != 0 ||
		len(may.ByGroup) != 1 || may.ByGroup[0].SpentCents != 0 || may.ByGroup[0].Categories == nil || len(may.Months) != 2 {
		t.Errorf("GetBudget(may) = %#v", may)
	}

	if _, err := service.GetBudget("2026-13"); !errors.Is(err, apperr.ErrBadRequest) {
		t.Errorf("GetBudget(bad month) error = %v, want a bad request", err)
	}
}

func TestSplitBudget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		income money.Cents
		shares []int
		want   []money.Cents
	}{
		{"even", 100_000, []int{60, 20, 10, 10}, []money.Cents{60_000, 20_000, 10_000, 10_000}},
		{"remainder to the largest", 100_001, []int{60, 40}, []money.Cents{60_001, 40_000}},
		{"remainder to the first of equal largest", 3, []int{50, 50}, []money.Cents{2, 1}},
		{"small income", 1, []int{60, 20, 10, 10}, []money.Cents{1, 0, 0, 0}},
		{"no income", 0, []int{60, 40}, []money.Cents{0, 0}},
		{"one group", 12_345, []int{100}, []money.Cents{12_345}},
	}
	for _, tt := range tests {
		got := splitBudget(tt.income, tt.shares)
		var sum money.Cents
		for i, v := range got {
			sum += v
			if v != tt.want[i] {
				t.Errorf("%s: splitBudget(%d, %v) = %v, want %v", tt.name, tt.income, tt.shares, got, tt.want)
				break
			}
		}
		if sum != tt.income {
			t.Errorf("%s: budgets add up to %d, want %d", tt.name, sum, tt.income)
		}
	}
}

func TestBudgetByGroupWithoutGroups(t *testing.T) {
	t.Parallel()

	if got := budgetByGroup(1000, nil, nil, []cashflow.CategoryTotal{{CategoryID: 1, Direction: "expense", TotalCents: 5}}); got != nil {
		t.Errorf("budgetByGroup(no groups) = %#v, want nil", got)
	}
}

func TestGetMonthAddsUpSavingsCategories(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	service := New(store.New(db))

	aliceID := testutil.InsertUser(t, db, "alice")
	foodID := testutil.MustExec(t, db,
		`INSERT INTO cashflow_categories (name, direction, icon, color) VALUES ('food', 'expense', 'utensils', '#E76F51')`)
	savingID := testutil.MustExec(t, db,
		`INSERT INTO cashflow_categories (name, direction, icon, color, role) VALUES ('saving', 'expense', 'piggy-bank', '#0E5A78', 'savings')`)
	for i, m := range []struct {
		categoryID int64
		cents      int64
	}{{foodID, 10_000}, {savingID, 25_000}, {savingID, 5_000}} {
		testutil.MustExec(t, db,
			`INSERT INTO cashflow_movements (sync_id, user_id, category_id, date, amount_cents) VALUES (?, ?, ?, '2026-03-02', ?)`,
			fmt.Sprintf("00000000-0000-4000-8000-%012d", i+1), aliceID, m.categoryID, m.cents)
	}

	detail, err := service.GetMonth("2026-03")
	if err != nil {
		t.Fatalf("GetMonth() error = %v", err)
	}
	if detail.SavedCents != 30_000 || detail.ExpenseCents != 40_000 {
		t.Errorf("saved = %d of %d expenses, want 30000 of 40000 (savings stay expenses)", detail.SavedCents, detail.ExpenseCents)
	}
	for _, c := range detail.ByCategory {
		want := ""
		if c.Name == "saving" {
			want = cashflow.RoleSavings
		}
		if c.Role != want {
			t.Errorf("category %q Role = %q, want %q", c.Name, c.Role, want)
		}
	}
}

func TestGetMonthSplitsEachPersonsExpensesIntoSpentAndPutAside(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	service := New(store.New(db))

	aliceID := testutil.InsertUser(t, db, "alice")
	bobID := testutil.InsertUser(t, db, "bob")
	salary := testutil.MustExec(t, db, `INSERT INTO cashflow_categories (name, direction, icon, color) VALUES ('salary', 'income', 'briefcase', '#16A34A')`)
	food := testutil.MustExec(t, db, `INSERT INTO cashflow_categories (name, direction, icon, color) VALUES ('food', 'expense', 'utensils', '#E76F51')`)
	savings := testutil.MustExec(t, db, `INSERT INTO cashflow_categories (name, direction, icon, color, role) VALUES ('savings', 'expense', 'piggy-bank', '#0284C7', 'savings')`)
	invest := testutil.MustExec(t, db, `INSERT INTO cashflow_categories (name, direction, icon, color, role) VALUES ('invest', 'expense', 'trending-up', '#0284C7', 'investments')`)
	loan := testutil.MustExec(t, db, `INSERT INTO cashflow_categories (name, direction, icon, color, role) VALUES ('mortgage', 'expense', 'house', '#8D6E63', 'credits')`)
	for i, m := range []struct {
		userID     int64
		categoryID int64
		date       string
		cents      int64
	}{
		{aliceID, salary, "2026-03-01", 300_000},
		{aliceID, food, "2026-03-02", 40_000},
		{aliceID, savings, "2026-03-03", 60_000},
		{aliceID, invest, "2026-03-03", 20_000},
		{aliceID, loan, "2026-03-04", 63_200}, // a credit payment is spent, not put aside
		{bobID, salary, "2026-03-01", 200_000},
		{bobID, savings, "2026-03-03", 30_000},
		{aliceID, salary, "2025-09-01", 100_000}, // outside the six months
		{aliceID, savings, "2025-09-02", 10_000},
		{aliceID, salary, "2026-01-01", 300_000},
		{aliceID, savings, "2026-01-02", 50_000},
	} {
		testutil.MustExec(t, db,
			`INSERT INTO cashflow_movements (sync_id, user_id, category_id, date, amount_cents) VALUES (?, ?, ?, ?, ?)`,
			fmt.Sprintf("00000000-0000-4000-8000-%012d", i+1), m.userID, m.categoryID, m.date, m.cents)
	}
	// Alice has a typed balance at the end of January; Bob has none and counts from zero.
	testutil.MustExec(t, db, `INSERT INTO cashflow_balances (user_id, month, balance_cents) VALUES (?, '2026-01', 1000000)`, aliceID)

	detail, err := service.GetMonth("2026-03")
	if err != nil {
		t.Fatalf("GetMonth() error = %v", err)
	}
	alice, bob := detail.ByUser[0], detail.ByUser[1]
	if alice.Username != "alice" || alice.PutAsideCents != 80_000 || alice.ExpenseCents != 183_200 {
		t.Errorf("alice = %+v, want 80000 put aside of 183200 expenses", alice)
	}
	if bob.PutAsideCents != 30_000 {
		t.Errorf("bob put aside = %d, want 30000", bob.PutAsideCents)
	}
	// February has no movements, so March is January's balance plus March's net: 300000-183200.
	if alice.CashCents == nil || *alice.CashCents != 1_000_000+116_800 {
		t.Errorf("alice cash = %v, want 1116800", alice.CashCents)
	}
	// Bob has no typed balance, so his cash starts from zero: 200000 in, 30000 put aside.
	if bob.CashCents == nil || *bob.CashCents != 170_000 {
		t.Errorf("bob cash = %v, want 170000 counted from zero", bob.CashCents)
	}

	want := []cashflow.UserMonthTotal{
		{Month: "2026-01", Username: "alice", IncomeCents: 300_000, PutAsideCents: 50_000},
		{Month: "2026-03", Username: "alice", IncomeCents: 300_000, PutAsideCents: 80_000},
		{Month: "2026-03", Username: "bob", IncomeCents: 200_000, PutAsideCents: 30_000},
	}
	if !slices.Equal(detail.PeopleHistory, want) {
		t.Errorf("PeopleHistory = %+v, want %+v", detail.PeopleHistory, want)
	}
}

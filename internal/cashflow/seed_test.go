package cashflow_test

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/auth"
	"github.com/tiagorocha94/household-webapp/internal/cashflow"
	"github.com/tiagorocha94/household-webapp/internal/csvdata"
	"github.com/tiagorocha94/household-webapp/internal/seed"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

var (
	groupsHeader     = []string{"name", "share_pct"}
	categoriesHeader = []string{"name", "direction", "icon", "color", "budget_group", "role"}
	movementsHeader  = []string{"user", "category", "date", "amount_cents", "description", "sync_id"}
	balancesHeader   = []string{"user", "month", "balance_cents"}
)

// sampleCategories and sampleMovements are in the exact order Export writes
// them, so a round trip must give the same rows back.
var (
	sampleCategories = [][]string{
		categoriesHeader,
		{"groceries", "expense", "shopping-basket", "#65A30D", "Essential", ""},
		{"refunds", "income", "gift", "#EC4899", "", ""},
		{"rent", "expense", "house", "#8D6E63", "Essential", ""},
		{"salary", "income", "briefcase", "#16A34A", "", ""},
		{"travel", "expense", "car-front", "#0284C7", "Fun", ""},
	}
	// The groups come out in the order of the file, not by name.
	sampleGroups = [][]string{
		groupsHeader,
		{"Essential", "70"},
		{"Fun", "30"},
	}
	// March: income 501000, expense 93000 (alice groceries twice on the 5th).
	// April: expenses above income. May: no income at all.
	sampleMovements = [][]string{
		movementsHeader,
		{"alice", "salary", "2026-03-01", "300000", "", sid(1)},
		{"bob", "salary", "2026-03-01", "200000", "", sid(2)},
		{"alice", "rent", "2026-03-02", "80000", "March rent", sid(3)},
		{"alice", "groceries", "2026-03-05", "4000", "", sid(4)},
		{"alice", "groceries", "2026-03-05", "4000", "", sid(5)},
		{"bob", "groceries", "2026-03-06", "5000", "", sid(6)},
		{"bob", "refunds", "2026-03-20", "1000", "Returned shoes", sid(7)},
		{"bob", "salary", "2026-04-01", "50000", "", sid(8)},
		{"bob", "rent", "2026-04-02", "90000", "", sid(9)},
		{"alice", "groceries", "2026-05-03", "3000", "", sid(10)},
	}
)

type monthFigures struct {
	income, expense, net int64
	ratePct, count       int
}

// sid makes a sync ID for test movement n.
func sid(n int) string { return fmt.Sprintf("00000000-0000-4000-8000-%012d", n) }

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

func newRunner() *seed.Runner {
	return seed.NewRunner(testutil.Logger(), auth.SeedModule{}, cashflow.SeedModule{})
}

func writeUsers(t *testing.T, dir string, usernames ...string) {
	t.Helper()
	rows := [][]string{{"username", "password"}}
	for _, u := range usernames {
		rows = append(rows, []string{u, testutil.PasswordHash})
	}
	writeCSV(t, filepath.Join(dir, "users.csv"), rows)
}

func writeSample(t *testing.T, dir string) {
	t.Helper()
	writeUsers(t, dir, "alice", "bob")
	writeCSV(t, filepath.Join(dir, "cashflow_budget_groups.csv"), sampleGroups)
	writeCSV(t, filepath.Join(dir, "cashflow_categories.csv"), sampleCategories)
	writeCSV(t, filepath.Join(dir, "cashflow_movements.csv"), sampleMovements)
}

func TestCashflowDerivedFiguresAndRoundTrip(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	runner := newRunner()
	dir := t.TempDir()
	writeSample(t, dir)

	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	// The savings rate is scaled by 100: 408000 / 501000 = 81.43%. A month with
	// expenses above income is negative (-40000 / 50000 = -80.00%), and a month
	// without income has a rate of 0.
	want := map[string]monthFigures{
		"2026-03": {501000, 93000, 408000, 8143, 7},
		"2026-04": {50000, 90000, -40000, -8000, 2},
		"2026-05": {0, 3000, -3000, 0, 1},
	}
	for month, w := range want {
		var got monthFigures
		err := db.QueryRow(
			`SELECT income_cents, expense_cents, net_cents, savings_rate_pct, movement_count
			 FROM cashflow_month_summary_view WHERE month = ?`, month,
		).Scan(&got.income, &got.expense, &got.net, &got.ratePct, &got.count)
		if err != nil {
			t.Fatalf("query summary for %s: %v", month, err)
		}
		if got != w {
			t.Errorf("summary for %s = %+v, want %+v", month, got, w)
		}
	}
	if got := queryString(t, db, "SELECT COUNT(*) FROM cashflow_month_summary_view"); got != "3" {
		t.Errorf("months with movements = %s, want 3", got)
	}

	// Two identical movements on one day both count.
	identical := `SELECT COUNT(*) FROM cashflow_movements WHERE date = '2026-03-05'`
	if got := queryString(t, db, identical); got != "2" {
		t.Errorf("movements on 2026-03-05 = %s, want 2", got)
	}

	// Per category in March: both groceries copies and the bob purchase add up.
	categoryTotal := `SELECT total_cents || '/' || movement_count || '/' || direction
	                  FROM cashflow_month_category_view WHERE month = '2026-03' AND category_name = ?`
	for category, wantTotal := range map[string]string{
		"groceries": "13000/3/expense",
		"salary":    "500000/2/income",
		"refunds":   "1000/1/income",
		"rent":      "80000/1/expense",
	} {
		if got := queryString(t, db, categoryTotal, category); got != wantTotal {
			t.Errorf("March total for %s = %s, want %s", category, got, wantTotal)
		}
	}

	// Per person in March.
	userTotal := `SELECT income_cents || '/' || expense_cents || '/' || net_cents
	              FROM cashflow_month_user_view WHERE month = '2026-03' AND username = ?`
	for user, wantTotal := range map[string]string{
		"alice": "300000/88000/212000",
		"bob":   "201000/5000/196000",
	} {
		if got := queryString(t, db, userTotal, user); got != wantTotal {
			t.Errorf("March total for %s = %s, want %s", user, got, wantTotal)
		}
	}

	exportDir := t.TempDir()
	if err := runner.Export(db, exportDir); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	for name, header := range map[string][]string{
		"cashflow_budget_groups.csv": groupsHeader,
		"cashflow_categories.csv":    categoriesHeader,
		"cashflow_movements.csv":     movementsHeader,
	} {
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

	// Importing the exported files again changes nothing.
	before := queryString(t, db, "SELECT group_concat(id) FROM (SELECT id FROM cashflow_movements ORDER BY id)")
	if err := runner.Import(db, exportDir); err != nil {
		t.Fatalf("re-import of the export error = %v", err)
	}
	after := queryString(t, db, "SELECT group_concat(id) FROM (SELECT id FROM cashflow_movements ORDER BY id)")
	if before != after {
		t.Errorf("movement ids after re-importing the export = %s, want %s", after, before)
	}
}

func TestCashflowImportIsIdempotentAndMirrorsCSV(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	runner := newRunner()
	dir := t.TempDir()
	writeSample(t, dir)

	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	const oldStamp = "2000-01-01 00:00:00"
	for _, table := range []string{"cashflow_budget_groups", "cashflow_categories", "cashflow_movements"} {
		if _, err := db.Exec("UPDATE "+table+" SET updated_at = ?", oldStamp); err != nil {
			t.Fatalf("set updated_at on %s: %v", table, err)
		}
	}
	ids := "SELECT group_concat(id) FROM (SELECT id FROM cashflow_movements ORDER BY id)"
	before := queryString(t, db, ids)

	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("second Import() error = %v", err)
	}
	if got := queryString(t, db, ids); got != before {
		t.Errorf("movement ids = %s, want %s", got, before)
	}
	for _, table := range []string{"cashflow_budget_groups", "cashflow_categories", "cashflow_movements"} {
		if got := queryString(t, db, "SELECT MAX(CAST(updated_at AS TEXT)) FROM "+table); got != oldStamp {
			t.Errorf("%s updated_at = %s, want untouched", table, got)
		}
	}

	// A changed direction updates the category.
	changed := append([][]string{}, sampleCategories...)
	changed[2] = []string{"refunds", "expense", "gift", "#EC4899", "Essential", ""}
	writeCSV(t, filepath.Join(dir, "cashflow_categories.csv"), changed)
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("direction-changing Import() error = %v", err)
	}
	if got := queryString(t, db, "SELECT direction FROM cashflow_categories WHERE name = 'refunds'"); got != "expense" {
		t.Errorf("refunds direction = %s, want expense", got)
	}

	// A changed icon or colour updates the category too.
	restyled := append([][]string{}, sampleCategories...)
	restyled[2] = []string{"refunds", "income", "award", "#F59E0B", "", ""}
	writeCSV(t, filepath.Join(dir, "cashflow_categories.csv"), restyled)
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("restyling Import() error = %v", err)
	}
	if got := queryString(t, db, "SELECT icon || ' ' || color FROM cashflow_categories WHERE name = 'refunds'"); got != "award #F59E0B" {
		t.Errorf("refunds icon and colour = %s, want award #F59E0B", got)
	}
	writeCSV(t, filepath.Join(dir, "cashflow_categories.csv"), sampleCategories)

	// Removing a row from the file removes it from the database.
	writeCSV(t, filepath.Join(dir, "cashflow_movements.csv"), append(
		[][]string{movementsHeader}, sampleMovements[1:9]...))
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("pruning Import() error = %v", err)
	}
	if got := queryString(t, db, "SELECT COUNT(*) FROM cashflow_movements"); got != "8" {
		t.Errorf("movement count = %s, want 8", got)
	}

	// A user with no rows left in the file ends up with no movements.
	writeCSV(t, filepath.Join(dir, "cashflow_movements.csv"), [][]string{
		movementsHeader,
		{"bob", "salary", "2026-03-01", "200000", "", sid(2)},
	})
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("user-clearing Import() error = %v", err)
	}
	if got := queryString(t, db, "SELECT COUNT(*) FROM cashflow_movements WHERE user_id = (SELECT id FROM users WHERE username = 'alice')"); got != "0" {
		t.Errorf("alice movement count = %s, want 0", got)
	}
	if got := queryString(t, db, "SELECT COUNT(*) FROM cashflow_movements"); got != "1" {
		t.Errorf("movement count = %s, want 1", got)
	}

	// Removing a category cascades to its movements; the movements file may no
	// longer name it.
	writeCSV(t, filepath.Join(dir, "cashflow_categories.csv"), [][]string{
		categoriesHeader,
		{"groceries", "expense", "shopping-basket", "#65A30D", "Essential", ""},
		{"salary", "income", "briefcase", "#16A34A", "", ""},
	})
	writeCSV(t, filepath.Join(dir, "cashflow_movements.csv"), [][]string{
		movementsHeader,
		{"bob", "salary", "2026-03-01", "200000", "", sid(2)},
		{"bob", "groceries", "2026-03-06", "5000", "", sid(6)},
	})
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("category-removing Import() error = %v", err)
	}
	for table, want := range map[string]string{
		"cashflow_categories": "2", "cashflow_movements": "2",
	} {
		if got := queryString(t, db, "SELECT COUNT(*) FROM "+table); got != want {
			t.Errorf("%s count = %s, want %s", table, got, want)
		}
	}
}

func TestCashflowImportRejectsBadRows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file string
		rows [][]string
	}{
		{"unknown direction", "cashflow_categories.csv", [][]string{categoriesHeader, {"salary", "transfer", "briefcase", "#16A34A", "", ""}}},
		{"empty icon", "cashflow_categories.csv", [][]string{categoriesHeader, {"salary", "income", "", "#16A34A", "", ""}}},
		{"icon not in kebab-case", "cashflow_categories.csv", [][]string{categoriesHeader, {"salary", "income", "Briefcase", "#16A34A", "", ""}}},
		{"colour without hash", "cashflow_categories.csv", [][]string{categoriesHeader, {"salary", "income", "briefcase", "16A34A", "", ""}}},
		{"short colour", "cashflow_categories.csv", [][]string{categoriesHeader, {"salary", "income", "briefcase", "#16A", "", ""}}},
		{"empty category name", "cashflow_categories.csv", [][]string{categoriesHeader, {"", "income", "briefcase", "#16A34A", "", ""}}},
		{"duplicate category", "cashflow_categories.csv", [][]string{categoriesHeader, {"salary", "income", "briefcase", "#16A34A", "", ""}, {"salary", "expense", "briefcase", "#16A34A", "", ""}}},
		{"missing sync id", "cashflow_movements.csv", [][]string{movementsHeader, {"alice", "salary", "2026-03-01", "100", "", ""}}},
		{"sync id not a uuid", "cashflow_movements.csv", [][]string{movementsHeader, {"alice", "salary", "2026-03-01", "100", "", "not-a-uuid"}}},
		{"uppercase sync id", "cashflow_movements.csv", [][]string{movementsHeader, {"alice", "salary", "2026-03-01", "100", "", "AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA"}}},
		{"duplicate sync id", "cashflow_movements.csv", [][]string{movementsHeader, {"alice", "salary", "2026-03-01", "100", "", sid(1)}, {"alice", "salary", "2026-03-02", "100", "", sid(1)}}},
		{"old five-column file", "cashflow_movements.csv", [][]string{{"user", "category", "date", "amount_cents", "description"}, {"alice", "salary", "2026-03-01", "100", ""}}},
		{"unknown user", "cashflow_movements.csv", [][]string{movementsHeader, {"nobody", "salary", "2026-03-01", "100", "", sid(100)}}},
		{"unknown category", "cashflow_movements.csv", [][]string{movementsHeader, {"alice", "lottery", "2026-03-01", "100", "", sid(100)}}},
		{"impossible day", "cashflow_movements.csv", [][]string{movementsHeader, {"alice", "salary", "2026-02-30", "100", "", sid(100)}}},
		{"unpadded date", "cashflow_movements.csv", [][]string{movementsHeader, {"alice", "salary", "2026-1-5", "100", "", sid(100)}}},
		{"month only", "cashflow_movements.csv", [][]string{movementsHeader, {"alice", "salary", "2026-03", "100", "", sid(100)}}},
		{"zero amount", "cashflow_movements.csv", [][]string{movementsHeader, {"alice", "salary", "2026-03-01", "0", "", sid(100)}}},
		{"decimal amount", "cashflow_movements.csv", [][]string{movementsHeader, {"alice", "salary", "2026-03-01", "10.50", "", sid(100)}}},
		{"negative amount", "cashflow_movements.csv", [][]string{movementsHeader, {"alice", "salary", "2026-03-01", "-100", "", sid(100)}}},
		{"balance for an unknown user", "cashflow_balances.csv", [][]string{balancesHeader, {"nobody", "2026-03", "1000"}}},
		{"balance with a bad month", "cashflow_balances.csv", [][]string{balancesHeader, {"alice", "2026-13", "1000"}}},
		{"balance with a day", "cashflow_balances.csv", [][]string{balancesHeader, {"alice", "2026-03-31", "1000"}}},
		{"decimal balance", "cashflow_balances.csv", [][]string{balancesHeader, {"alice", "2026-03", "10.50"}}},
		{"empty balance", "cashflow_balances.csv", [][]string{balancesHeader, {"alice", "2026-03", ""}}},
		{"only a minus sign", "cashflow_balances.csv", [][]string{balancesHeader, {"alice", "2026-03", "-"}}},
		{"two balances for one month", "cashflow_balances.csv", [][]string{balancesHeader, {"alice", "2026-03", "100"}, {"alice", "2026-03", "200"}}},
		{"empty amount", "cashflow_movements.csv", [][]string{movementsHeader, {"alice", "salary", "2026-03-01", "", "", sid(100)}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := testutil.NewDB(t)
			runner := newRunner()
			dir := t.TempDir()
			writeUsers(t, dir, "alice")
			writeCSV(t, filepath.Join(dir, "cashflow_categories.csv"), [][]string{categoriesHeader, {"salary", "income", "briefcase", "#16A34A", "", ""}})
			writeCSV(t, filepath.Join(dir, tt.file), tt.rows)
			if err := runner.Import(db, dir); err == nil {
				t.Fatal("Import() error = nil, want an error")
			}
		})
	}
}

func TestCashflowBudgetGroupsKeepTheOrderOfTheFileAndMapCategories(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	runner := newRunner()
	dir := t.TempDir()
	writeSample(t, dir)
	// Fun first: the order of the file is the order the groups keep.
	writeCSV(t, filepath.Join(dir, "cashflow_budget_groups.csv"), [][]string{
		groupsHeader, {"Fun", "30"}, {"Essential", "70"},
	})
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if got := queryString(t, db, "SELECT group_concat(name || ':' || share_pct || ':' || position, ' ') FROM (SELECT * FROM cashflow_budget_groups ORDER BY position)"); got != "Fun:30:1 Essential:70:2" {
		t.Errorf("groups = %q, want Fun:30:1 Essential:70:2", got)
	}
	mapping := `SELECT group_concat(c.name || '>' || COALESCE(g.name, '-'), ' ')
	            FROM (SELECT * FROM cashflow_categories ORDER BY name) c
	            LEFT JOIN cashflow_budget_groups g ON g.id = c.budget_group_id`
	if got := queryString(t, db, mapping); got != "groceries>Essential refunds>- rent>Essential salary>- travel>Fun" {
		t.Errorf("category groups = %q", got)
	}

	exportDir := t.TempDir()
	if err := runner.Export(db, exportDir); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	got, err := csvdata.Read(filepath.Join(exportDir, "cashflow_budget_groups.csv"), groupsHeader)
	if err != nil {
		t.Fatalf("read exported groups: %v", err)
	}
	if want := [][]string{{"Fun", "30"}, {"Essential", "70"}}; !reflect.DeepEqual(got, want) {
		t.Errorf("exported groups = %v, want %v", got, want)
	}
}

func TestCashflowBudgetGroupsCanChangeAndLeave(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	runner := newRunner()
	dir := t.TempDir()
	writeSample(t, dir)
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	// A new share, and a group that leaves the file together with the
	// category mapping to it.
	writeCSV(t, filepath.Join(dir, "cashflow_budget_groups.csv"), [][]string{groupsHeader, {"Essential", "100"}})
	moved := append([][]string{}, sampleCategories...)
	moved[5] = []string{"travel", "expense", "car-front", "#0284C7", "Essential", ""}
	writeCSV(t, filepath.Join(dir, "cashflow_categories.csv"), moved)
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if got := queryString(t, db, "SELECT group_concat(name || ':' || share_pct) FROM cashflow_budget_groups"); got != "Essential:100" {
		t.Errorf("groups = %q, want Essential:100", got)
	}
	if got := queryString(t, db, "SELECT g.name FROM cashflow_categories c JOIN cashflow_budget_groups g ON g.id = c.budget_group_id WHERE c.name = 'travel'"); got != "Essential" {
		t.Errorf("travel group = %q, want Essential", got)
	}

	// An empty groups file turns the feature off: no groups, and categories
	// need none.
	writeCSV(t, filepath.Join(dir, "cashflow_budget_groups.csv"), [][]string{groupsHeader})
	ungrouped := [][]string{categoriesHeader}
	for _, row := range sampleCategories[1:] {
		ungrouped = append(ungrouped, []string{row[0], row[1], row[2], row[3], "", ""})
	}
	writeCSV(t, filepath.Join(dir, "cashflow_categories.csv"), ungrouped)
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("Import() without groups error = %v", err)
	}
	if got := queryString(t, db, "SELECT COUNT(*) FROM cashflow_budget_groups"); got != "0" {
		t.Errorf("groups = %s, want none", got)
	}
}

func TestCashflowImportRejectsBadBudgetGroups(t *testing.T) {
	t.Parallel()

	grouped := func(group string) [][]string {
		return [][]string{categoriesHeader, {"food", "expense", "utensils", "#E76F51", group, ""}}
	}
	tests := []struct {
		name       string
		groups     [][]string
		categories [][]string
	}{
		{"shares below 100", [][]string{groupsHeader, {"Essential", "60"}, {"Fun", "30"}}, grouped("Essential")},
		{"shares above 100", [][]string{groupsHeader, {"Essential", "80"}, {"Fun", "30"}}, grouped("Essential")},
		{"zero share", [][]string{groupsHeader, {"Essential", "100"}, {"Fun", "0"}}, grouped("Essential")},
		{"share above 100", [][]string{groupsHeader, {"Essential", "101"}}, grouped("Essential")},
		{"decimal share", [][]string{groupsHeader, {"Essential", "99.5"}, {"Fun", "0.5"}}, grouped("Essential")},
		{"text share", [][]string{groupsHeader, {"Essential", "all"}}, grouped("Essential")},
		{"empty group name", [][]string{groupsHeader, {"", "100"}}, grouped("")},
		{"duplicate group", [][]string{groupsHeader, {"Essential", "50"}, {"Essential", "50"}}, grouped("Essential")},
		{"unknown group on a category", [][]string{groupsHeader, {"Essential", "100"}}, grouped("Fun")},
		{"group names are case sensitive", [][]string{groupsHeader, {"Essential", "100"}}, grouped("essential")},
		{"expense category without a group", [][]string{groupsHeader, {"Essential", "100"}}, grouped("")},
		{"income category with a group", [][]string{groupsHeader, {"Essential", "100"}}, [][]string{
			categoriesHeader, {"food", "expense", "utensils", "#E76F51", "Essential", ""}, {"salary", "income", "briefcase", "#16A34A", "Essential", ""},
		}},
		{"group named while no groups exist", [][]string{groupsHeader}, grouped("Essential")},
		{"old four-column categories file", [][]string{groupsHeader, {"Essential", "100"}}, [][]string{
			{"name", "direction", "icon", "color"}, {"food", "expense", "utensils", "#E76F51"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := testutil.NewDB(t)
			runner := newRunner()
			dir := t.TempDir()
			writeUsers(t, dir, "alice")
			writeCSV(t, filepath.Join(dir, "cashflow_budget_groups.csv"), tt.groups)
			writeCSV(t, filepath.Join(dir, "cashflow_categories.csv"), tt.categories)
			if err := runner.Import(db, dir); err == nil {
				t.Fatal("Import() error = nil, want an error")
			}
			if got := queryString(t, db, "SELECT COUNT(*) FROM cashflow_budget_groups"); got != "0" {
				t.Errorf("groups after a rejected import = %s, want none (the import is one transaction)", got)
			}
		})
	}
}

func TestCashflowCategoryRolesImportAndExport(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	dir := t.TempDir()
	cats := [][]string{
		categoriesHeader,
		{"goal pot", "expense", "target", "#7C3AED", "", "goals"},
		{"invest", "expense", "trending-up", "#0D9488", "", "investments"},
		{"mortgage", "expense", "house", "#8D6E63", "", "credits"},
		{"salary", "income", "briefcase", "#16A34A", "", ""},
		{"saving", "expense", "piggy-bank", "#0E5A78", "", "savings"},
		{"travel", "expense", "car-front", "#0284C7", "", ""},
	}
	writeUsers(t, dir, "alice")
	writeCSV(t, filepath.Join(dir, "cashflow_budget_groups.csv"), [][]string{groupsHeader})
	writeCSV(t, filepath.Join(dir, "cashflow_categories.csv"), cats)
	writeCSV(t, filepath.Join(dir, "cashflow_movements.csv"), [][]string{movementsHeader})
	if err := newRunner().Import(db, dir); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	for role, want := range map[string]string{"savings": "saving", "investments": "invest", "goals": "goal pot", "credits": "mortgage"} {
		got := queryString(t, db, "SELECT GROUP_CONCAT(name) FROM cashflow_categories WHERE role = '"+role+"'")
		if got != want {
			t.Errorf("categories with role %s = %q, want %q", role, got, want)
		}
	}

	out := t.TempDir()
	if err := newRunner().Export(db, out); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	got, err := csvdata.Read(filepath.Join(out, "cashflow_categories.csv"), categoriesHeader)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	if !reflect.DeepEqual(got, cats[1:]) {
		t.Errorf("exported categories = %#v, want %#v", got, cats[1:])
	}
}

func TestCashflowImportRejectsBadRoles(t *testing.T) {
	t.Parallel()

	for name, row := range map[string][]string{
		"income category":   {"salary", "income", "briefcase", "#16A34A", "", "savings"},
		"not a role":        {"travel", "expense", "car-front", "#0284C7", "", "true"},
		"the old flag":      {"travel", "expense", "car-front", "#0284C7", "", "false"},
		"unknown role name": {"travel", "expense", "car-front", "#0284C7", "", "holiday"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			writeUsers(t, dir, "alice")
			writeCSV(t, filepath.Join(dir, "cashflow_budget_groups.csv"), [][]string{groupsHeader})
			writeCSV(t, filepath.Join(dir, "cashflow_categories.csv"), [][]string{categoriesHeader, row})
			writeCSV(t, filepath.Join(dir, "cashflow_movements.csv"), [][]string{movementsHeader})
			if err := newRunner().Import(testutil.NewDB(t), dir); err == nil {
				t.Fatalf("Import() with %v error = nil, want a rejection", row)
			}
		})
	}
}

func TestCashflowBalancesRoundTripChangeAndRemoval(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	runner := newRunner()
	dir := t.TempDir()
	writeUsers(t, dir, "alice", "bob")
	writeCSV(t, filepath.Join(dir, "cashflow_budget_groups.csv"), sampleGroups)
	writeCSV(t, filepath.Join(dir, "cashflow_categories.csv"), sampleCategories)
	// A balance can be negative. The rows are in the order Export writes them.
	first := [][]string{
		balancesHeader,
		{"alice", "2026-03", "250000"},
		{"alice", "2026-05", "-1200"},
		{"bob", "2026-03", "0"},
	}
	writeCSV(t, filepath.Join(dir, "cashflow_balances.csv"), first)
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if got := queryString(t, db, "SELECT COUNT(*) FROM cashflow_balances"); got != "3" {
		t.Errorf("balances = %s, want 3", got)
	}

	exportDir := t.TempDir()
	if err := runner.Export(db, exportDir); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	got, err := csvdata.Read(filepath.Join(exportDir, "cashflow_balances.csv"), balancesHeader)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	if want := first[1:]; !reflect.DeepEqual(got, want) {
		t.Errorf("exported balances = %#v, want %#v", got, want)
	}

	// A changed balance is updated, a balance that left the file is removed.
	writeCSV(t, filepath.Join(dir, "cashflow_balances.csv"), [][]string{
		balancesHeader,
		{"alice", "2026-03", "260000"},
		{"bob", "2026-03", "0"},
	})
	if err := runner.Import(db, dir); err != nil {
		t.Fatalf("second Import() error = %v", err)
	}
	if got := queryString(t, db, "SELECT group_concat(balance_cents) FROM (SELECT balance_cents FROM cashflow_balances ORDER BY user_id, month)"); got != "260000,0" {
		t.Errorf("balances after the change = %s, want 260000,0", got)
	}
}

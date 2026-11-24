package goals_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

func TestGoalImportRejectsInvalidRows(t *testing.T) {
	t.Parallel()

	accountsHeader := []string{"name", "monthly_deposit_cents"}
	goalsHeader := []string{"account", "name", "target_cents", "start_month", "end_month", "status"}
	movementsHeader := []string{"account", "type", "goal", "month", "amount_cents", "description"}

	tests := []struct {
		name string
		file string
		rows [][]string
		want string
	}{
		{"duplicate account name", "goal_accounts.csv",
			[][]string{accountsHeader, {"Household", "36600"}, {"Household", "1"}}, "more than once"},
		{"negative deposit", "goal_accounts.csv",
			[][]string{accountsHeader, {"Household", "-1"}}, "monthly_deposit_cents"},
		{"missing deposit", "goal_accounts.csv",
			[][]string{accountsHeader, {"Household", ""}}, "monthly_deposit_cents"},
		{"bad month", "goal_account_goals.csv",
			[][]string{goalsHeader, {"Household", "Trip", "100", "2026-13", "2026-14", "active"}}, "start_month"},
		{"end before start", "goal_account_goals.csv",
			[][]string{goalsHeader, {"Household", "Trip", "100", "2026-05", "2026-01", "active"}}, "end_month"},
		{"unknown status", "goal_account_goals.csv",
			[][]string{goalsHeader, {"Household", "Trip", "100", "2026-01", "2026-02", "paused"}}, "status"},
		{"deposit with goal", "goal_account_movements.csv",
			[][]string{movementsHeader, {"Household", "deposit", "Trip", "2026-01", "100", ""}}, "deposits must not"},
		{"allocation without goal", "goal_account_movements.csv",
			[][]string{movementsHeader, {"Household", "allocation", "", "2026-01", "100", ""}}, "requires a goal"},
		{"bad movement month", "goal_account_movements.csv",
			[][]string{movementsHeader, {"Household", "deposit", "", "2026-00", "100", ""}}, "month"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := testutil.NewDB(t)
			dir := t.TempDir()
			writeTestCSV(t, filepath.Join(dir, "goal_accounts.csv"), [][]string{
				accountsHeader, {"Household", "36600"},
			})
			writeTestCSV(t, filepath.Join(dir, "goal_account_goals.csv"), [][]string{
				goalsHeader, {"Household", "Trip", "100", "2026-01", "2026-02", "active"},
			})
			writeTestCSV(t, filepath.Join(dir, tt.file), tt.rows)

			err := newRunner().Import(db, dir)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("newRunner().Import() error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

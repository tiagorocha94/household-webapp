package service

import (
	"testing"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/goals"
	"github.com/tiagorocha94/household-webapp/internal/goals/store"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

func TestSetElapsedMonthsFollowsTheLocalMonth(t *testing.T) {
	t.Parallel()

	// 00:30 on 1 March in UTC+1 is still February in UTC, so the old UTC-based
	// view counted two months here instead of three.
	lisbon := time.FixedZone("WEST", 3600)
	tests := []struct {
		name string
		now  time.Time
		want int
	}{
		{"before start", time.Date(2025, 12, 31, 23, 59, 0, 0, lisbon), 0},
		{"first month", time.Date(2026, 1, 1, 0, 0, 0, 0, lisbon), 1},
		{"just after a local month change", time.Date(2026, 3, 1, 0, 30, 0, 0, lisbon), 3},
		{"last month", time.Date(2026, 12, 31, 23, 59, 0, 0, lisbon), 12},
		{"after end", time.Date(2027, 6, 1, 0, 0, 0, 0, lisbon), 12},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := &Service{now: func() time.Time { return tt.now }}
			goal := &goals.GoalSummary{GoalID: 1, StartMonth: "2026-01", EndMonth: "2026-12"}
			if err := svc.setElapsedMonths(goal); err != nil {
				t.Fatalf("setElapsedMonths() error = %v", err)
			}
			if goal.ElapsedMonths != tt.want {
				t.Errorf("ElapsedMonths = %d, want %d", goal.ElapsedMonths, tt.want)
			}
		})
	}

	svc := &Service{now: time.Now}
	if err := svc.setElapsedMonths(&goals.GoalSummary{GoalID: 2, StartMonth: "bad", EndMonth: "2026-12"}); err == nil {
		t.Error("setElapsedMonths(bad start) succeeded, want error")
	}
}

func TestGoalSummariesCarryElapsedMonths(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	accountID := createTestAccount(t, db)
	goalID := createGoal(t, db, accountID, "Trip", 12000, "2026-01", "2026-12", goals.StatusActive)

	svc := New(store.New(db))
	svc.now = func() time.Time { return time.Date(2026, 3, 1, 0, 30, 0, 0, time.FixedZone("WEST", 3600)) }

	goal, err := svc.GetGoalSummary(goalID)
	if err != nil || goal.ElapsedMonths != 3 {
		t.Fatalf("GetGoalSummary() = %+v, %v, want 3 elapsed months", goal, err)
	}
	list, err := svc.ListGoalSummaries(accountID)
	if err != nil || len(list) != 1 || list[0].ElapsedMonths != 3 {
		t.Fatalf("ListGoalSummaries() = %+v, %v, want one goal with 3 elapsed months", list, err)
	}
}

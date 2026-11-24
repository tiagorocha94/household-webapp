package service

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/tiagorocha94/household-webapp/internal/goals"
	"github.com/tiagorocha94/household-webapp/internal/goals/store"
	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/month"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

// generateProjection sets the account's monthly deposit and builds its projection.
func generateProjection(db *sql.DB, accountID int64, deposit money.Cents) ([]goals.ProjectionMonth, error) {
	if _, err := db.Exec("UPDATE goal_accounts SET monthly_deposit_cents = ? WHERE id = ?", deposit, accountID); err != nil {
		return nil, err
	}
	return New(store.New(db)).GetProjection(accountID)
}

// TestProjectionNoActivityProjectsRecurringDeposit verifies the
// default 12-months-back to 6-months-forward window, and that months are
// correctly split into actual (zero, no data) vs. projected (recurring
// deposit only, since there are no goals).
func TestProjectionNoActivityProjectsRecurringDeposit(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	accountID := createTestAccount(t, db)

	projection, err := generateProjection(db, accountID, 30000)
	if err != nil {
		t.Fatalf("GetProjection returned error: %v", err)
	}

	currentMonth := monthOffset(0)

	const wantLen = 19 // -12 .. +6 inclusive
	if len(projection) != wantLen {
		t.Fatalf("expected %d months, got %d", wantLen, len(projection))
	}

	var runningBalance money.Cents
	for _, pm := range projection {
		wantProjection := pm.Month > currentMonth
		if pm.IsProjection != wantProjection {
			t.Errorf("month %s: IsProjection = %v, want %v", pm.Month, pm.IsProjection, wantProjection)
		}

		var wantDeposit money.Cents
		if wantProjection {
			wantDeposit = 30000
		}
		if pm.DepositsCents != wantDeposit {
			t.Errorf("month %s: DepositsCents = %v, want %v", pm.Month, pm.DepositsCents, wantDeposit)
		}
		if pm.AllocationsCents != 0 {
			t.Errorf("month %s: AllocationsCents = %v, want 0 (no goals)", pm.Month, pm.AllocationsCents)
		}

		runningBalance += wantDeposit
		if pm.BalanceCents != runningBalance {
			t.Errorf("month %s: BalanceCents = %v, want %v", pm.Month, pm.BalanceCents, runningBalance)
		}
		if pm.UnallocatedCents != runningBalance {
			t.Errorf("month %s: UnallocatedCents = %v, want %v (no allocations)", pm.Month, pm.UnallocatedCents, runningBalance)
		}
	}
}

// TestProjectionPastActualsCarryIntoProjectedBalance verifies that
// real deposit/allocation/expense movements accumulate correctly into
// BalanceCents and UnallocatedCents, and that an active goal's planned allocation
// continues into the projected months after its last recorded allocation.
func TestProjectionPastActualsCarryIntoProjectedBalance(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	accountID := createTestAccount(t, db)

	twoMonthsAgo := monthOffset(-2)
	oneMonthAgo := monthOffset(-1)
	currentMonth := monthOffset(0)
	goalEnd := monthOffset(2)

	insertMovement(t, db, accountID, nil, goals.MovementDeposit, twoMonthsAgo, 50000)
	insertMovement(t, db, accountID, nil, goals.MovementDeposit, oneMonthAgo, 50000)
	insertMovement(t, db, accountID, nil, goals.MovementDeposit, currentMonth, 50000)

	// 4 months (oneMonthAgo..goalEnd inclusive) at 30000 total -> 7500/month.
	goalID := createGoal(t, db, accountID, "Fund", 30000, oneMonthAgo, goalEnd, goals.StatusActive)
	insertMovement(t, db, accountID, &goalID, goals.MovementAllocation, oneMonthAgo, 7500)
	insertMovement(t, db, accountID, &goalID, goals.MovementExpense, currentMonth, 2000)

	projection, err := generateProjection(db, accountID, 20000)
	if err != nil {
		t.Fatalf("GetProjection returned error: %v", err)
	}

	wantDeposits := map[string]money.Cents{twoMonthsAgo: 50000, oneMonthAgo: 50000, currentMonth: 50000}
	wantAllocations := map[string]money.Cents{oneMonthAgo: 7500}
	wantExpenses := map[string]money.Cents{currentMonth: 2000}

	var totalDeposits, totalAllocations, totalExpenses money.Cents
	for _, pm := range projection {
		if pm.Month <= currentMonth {
			totalDeposits += wantDeposits[pm.Month]
			totalAllocations += wantAllocations[pm.Month]
			totalExpenses += wantExpenses[pm.Month]
		} else {
			totalDeposits += 20000
			if pm.Month <= goalEnd {
				totalAllocations += 7500
			}
		}

		wantBalance := totalDeposits - totalExpenses
		wantUnallocated := totalDeposits - totalAllocations
		if pm.BalanceCents != wantBalance {
			t.Errorf("month %s: BalanceCents = %v, want %v", pm.Month, pm.BalanceCents, wantBalance)
		}
		if pm.UnallocatedCents != wantUnallocated {
			t.Errorf("month %s: UnallocatedCents = %v, want %v", pm.Month, pm.UnallocatedCents, wantUnallocated)
		}
	}
}

// TestProjectionWindowExtendsToLastActiveGoalEndMonth verifies the
// projection window reaches as far as the furthest active goal's end month,
// and ignores non-active goals even if their end month is farther out.
func TestProjectionWindowExtendsToLastActiveGoalEndMonth(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	accountID := createTestAccount(t, db)

	activeEnd := monthOffset(20) // farther than the default +6 month window
	cancelledEnd := monthOffset(30)

	createGoal(t, db, accountID, "Long-term", 120000, monthOffset(0), activeEnd, goals.StatusActive)
	createGoal(t, db, accountID, "Abandoned", 60000, monthOffset(0), cancelledEnd, goals.StatusCancelled)

	projection, err := generateProjection(db, accountID, 10000)
	if err != nil {
		t.Fatalf("GetProjection returned error: %v", err)
	}

	last := projection[len(projection)-1]
	if last.Month != activeEnd {
		t.Fatalf("expected projection to extend to the active goal's end month %s, got %s", activeEnd, last.Month)
	}
}

// TestProjectionReleaseIncreasesUnallocated verifies a release
// movement adds back to UnallocatedCents without affecting BalanceCents, since a
// release moves money between a goal and the unallocated pool rather than
// in or out of the account.
func TestProjectionReleaseIncreasesUnallocated(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	accountID := createTestAccount(t, db)

	month := monthOffset(-1)
	insertMovement(t, db, accountID, nil, goals.MovementDeposit, month, 10000)

	goalID := createGoal(t, db, accountID, "Small", 5000, month, month, goals.StatusCompleted)
	insertMovement(t, db, accountID, &goalID, goals.MovementAllocation, month, 5000)
	insertMovement(t, db, accountID, &goalID, goals.MovementRelease, month, 5000)

	projection, err := generateProjection(db, accountID, 0)
	if err != nil {
		t.Fatalf("GetProjection returned error: %v", err)
	}

	pm := findMonth(t, projection, month)

	const wantUnallocated = money.Cents(10000 - 5000 + 5000) // deposits - allocations + releases
	if pm.UnallocatedCents != wantUnallocated {
		t.Errorf("UnallocatedCents = %v, want %v", pm.UnallocatedCents, wantUnallocated)
	}

	const wantBalance = money.Cents(10000) // deposits - expenses(0); release doesn't touch balance
	if pm.BalanceCents != wantBalance {
		t.Errorf("BalanceCents = %v, want %v", pm.BalanceCents, wantBalance)
	}
}

// TestProjectionCoverReducesUnallocated verifies a cover movement takes money
// from the unallocated pool, again without touching the account balance.
func TestProjectionCoverReducesUnallocated(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	accountID := createTestAccount(t, db)

	month := monthOffset(-1)
	insertMovement(t, db, accountID, nil, goals.MovementDeposit, month, 10000)

	goalID := createGoal(t, db, accountID, "Small", 5000, month, month, goals.StatusCompleted)
	insertMovement(t, db, accountID, &goalID, goals.MovementAllocation, month, 5000)
	insertMovement(t, db, accountID, &goalID, goals.MovementExpense, month, 6000)
	insertMovement(t, db, accountID, &goalID, goals.MovementCover, month, 1000)

	projection, err := generateProjection(db, accountID, 0)
	if err != nil {
		t.Fatalf("GetProjection returned error: %v", err)
	}

	pm := findMonth(t, projection, month)

	const wantUnallocated = money.Cents(10000 - 5000 - 1000) // deposits - allocations - covers
	if pm.UnallocatedCents != wantUnallocated {
		t.Errorf("UnallocatedCents = %v, want %v", pm.UnallocatedCents, wantUnallocated)
	}
	const wantBalance = money.Cents(10000 - 6000) // deposits - expenses
	if pm.BalanceCents != wantBalance {
		t.Errorf("BalanceCents = %v, want %v", pm.BalanceCents, wantBalance)
	}
}

func TestGetPlannedAllocationsSpreadsRemainderCents(t *testing.T) {
	t.Parallel()

	schedules := []goalSchedule{{
		startMonth: "2026-01",
		endMonth:   "2026-03",
		schedule:   mustSchedule(t, "2026-01", "2026-03", 10000),
	}}

	want := map[string]money.Cents{"2025-12": 0, "2026-01": 3334, "2026-02": 3333, "2026-03": 3333, "2026-04": 0}
	for month, w := range want {
		if got := getPlannedAllocations(schedules, month); got != w {
			t.Errorf("month %s: planned = %v, want %v", month, got, w)
		}
	}
}

func TestBuildSchedulesFinalMonthAbsorbsDrift(t *testing.T) {
	t.Parallel()

	// Split(10000, 3) is 3334/3333/3333; the first month recorded 3333.
	schedules, err := buildSchedules([]goals.GoalPlan{{
		StartMonth: "2026-01", EndMonth: "2026-03", TargetCents: 10000,
		AllocationCount: 1, AllocatedCents: 3333,
	}})
	if err != nil {
		t.Fatalf("buildSchedules() error = %v", err)
	}
	if got := getPlannedAllocations(schedules, "2026-02"); got != 3333 {
		t.Errorf("february = %v, want 3333", got)
	}
	if got := getPlannedAllocations(schedules, "2026-03"); got != 3334 {
		t.Errorf("march = %v, want 3334", got)
	}
}

// TestSyncAndProjectionAgreeOnTheFinalMonth ties the two callers together. Sync
// allocates the last month as the target minus everything in the other months.
// The projection plans it from the schedule plus drift. For goals whose earlier
// months are recorded, as sync leaves them, both must give the same amount.
func TestSyncAndProjectionAgreeOnTheFinalMonth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		target   money.Cents
		recorded []money.Cents // the first months, one allocation each
		months   int
	}{
		{"on schedule", 10000, []money.Cents{3334, 3333}, 3},
		{"rounded down", 10000, []money.Cents{3333, 3333}, 3},
		{"edited up", 10000, []money.Cents{4000, 3333}, 3},
		{"edited down", 10000, []money.Cents{100, 100, 100}, 4},
		{"over-allocated", 10000, []money.Cents{6000, 6000}, 3},
		{"single month", 777, nil, 1},
		{"long goal", 1250000, []money.Cents{21552, 21552, 21552, 21552}, 58},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Each case records every month but the last, so the earlier months
			// are all in place before the last one is allocated.
			if len(tt.recorded) > tt.months-1 {
				t.Fatalf("case records %d months of %d", len(tt.recorded), tt.months)
			}
			start := "2026-01"
			end, err := month.Add(start, tt.months-1)
			if err != nil {
				t.Fatalf("month.Add() error = %v", err)
			}

			var others money.Cents
			for _, r := range tt.recorded {
				others += r
			}
			// Sync fills any months it has not recorded yet with their share, in
			// order, before it reaches the last one.
			schedule := mustSchedule(t, start, end, tt.target)
			for i := len(tt.recorded); i < tt.months-1; i++ {
				others += schedule.Amount(i, 0)
			}
			syncAmount := schedule.FinalAmount(others)
			if want := tt.target - others; syncAmount != want {
				t.Fatalf("sync amount = %d, want target - others = %d", syncAmount, want)
			}

			// The projection sees the same goal as recorded months and a total.
			var recordedTotal money.Cents
			for _, r := range tt.recorded {
				recordedTotal += r
			}
			schedules, err := buildSchedules([]goals.GoalPlan{{
				StartMonth: start, EndMonth: end, TargetCents: tt.target,
				AllocationCount: len(tt.recorded), AllocatedCents: recordedTotal,
			}})
			if err != nil {
				t.Fatalf("buildSchedules() error = %v", err)
			}
			if got, want := getPlannedAllocations(schedules, end), max(syncAmount, 0); got != want {
				t.Errorf("projection amount = %d, want %d (sync amount %d, never below zero)", got, want, syncAmount)
			}
		})
	}
}

func mustSchedule(t *testing.T, start, end string, target money.Cents) goals.Schedule {
	t.Helper()
	s, err := goals.NewSchedule(start, end, target)
	if err != nil {
		t.Fatalf("NewSchedule(%q, %q, %d) error = %v", start, end, target, err)
	}
	return s
}

package goals

import (
	"time"

	"github.com/tiagorocha94/household-webapp/internal/money"
)

// Goal status values.
const (
	StatusActive    = "active"
	StatusCompleted = "completed"
	StatusCancelled = "cancelled"
)

// GoalSummary holds a goal with its computed fields from
// the goal_account_goal_summary_view.
type GoalSummary struct {
	GoalID                 int64
	AccountID              int64
	Name                   string
	TargetCents            money.Cents
	StartMonth             string
	EndMonth               string
	Status                 string
	TotalMonths            int
	MonthlyAllocationCents money.Cents
	ElapsedMonths          int // set by the service from the current local month, not by the view
	TotalAllocatedCents    money.Cents
	TotalExpensesCents     money.Cents
	TotalReleasedCents     money.Cents
	TotalCoveredCents      money.Cents
	GoalBalanceCents       money.Cents
	// ClosingBalanceCents is the balance the goal had when it was closed: the
	// leftover when positive, the overspend when negative. Nil while open.
	ClosingBalanceCents *money.Cents
	ProgressPct         int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

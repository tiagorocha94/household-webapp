package goals

import (
	"time"

	"github.com/tiagorocha94/household-webapp/internal/money"
)

// Movement type values.
const (
	MovementDeposit    = "deposit"
	MovementAllocation = "allocation"
	MovementExpense    = "expense"
	MovementRelease    = "release"
	MovementCover      = "cover"
)

// Movement represents a single money event in a goal account ledger.
type Movement struct {
	ID          int64
	AccountID   int64
	GoalID      *int64
	GoalName    string
	Type        string
	AmountCents money.Cents
	Month       string
	Description string
	CreatedAt   time.Time
}

// IsInflow returns true if the movement adds money to the account.
// This is the account (bank-statement) perspective: deposits and releases
// increase the unallocated money, allocations and covers reduce it.
func (m Movement) IsInflow() bool {
	return m.Type == MovementDeposit || m.Type == MovementRelease
}

// IsInflowForGoal returns true if the movement adds money to the goal.
// This is the goal's own perspective, which is the inverse of the account
// perspective for allocations, releases and covers: an allocation reserves
// money into the goal and a cover tops up an overspent goal on close (both
// inflows), while an expense spends it and a release returns the leftover out
// of the goal on close (both outflows). Deposits are not goal-scoped, so they
// are not goal inflows.
func (m Movement) IsInflowForGoal() bool {
	return m.Type == MovementAllocation || m.Type == MovementCover
}

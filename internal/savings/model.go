// Package savings records money kept aside and within reach, such as an
// emergency fund: accounts, the deposits and withdrawals of each, and value
// snapshots. It is separate from investments, so savings never add to
// "Invested" or its gains (ADR 014). Every value is typed into CSV files.
package savings

import (
	"time"

	"github.com/tiagorocha94/household-webapp/internal/money"
)

// Movement types.
const (
	TypeDeposit    = "deposit"
	TypeWithdrawal = "withdrawal"
)

// Account is a place where money is saved.
type Account struct {
	ID   int64
	Name string
	// Username is the owner, or empty when the account belongs to the household.
	Username string
	// TargetCents is the amount to reach, or 0 when the account has no target.
	TargetCents money.Cents
	Notes       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// AccountSummary is an account with the figures derived from its movements and
// snapshots.
type AccountSummary struct {
	Account
	MovementCount  int
	SnapshotCount  int
	DepositedCents money.Cents
	WithdrawnCents money.Cents
	// BalanceCents is the latest snapshot, or the deposits less the
	// withdrawals while there is no snapshot.
	BalanceCents money.Cents
	// InterestCents is the balance less deposits plus withdrawals. It is 0
	// without a snapshot, as the balance is then only the movements.
	InterestCents money.Cents
	// ProgressPct is the balance as a whole percent of the target. It is 0
	// without a target, never negative and can be above 100.
	ProgressPct       int
	LastSnapshotMonth string
}

// Movement is money put into or taken out of an account in a month.
type Movement struct {
	ID          int64
	AccountID   int64
	Month       string
	Type        string
	AmountCents money.Cents
	Description string
	CreatedAt   time.Time
}

// Snapshot is the balance seen in a month.
type Snapshot struct {
	ID          int64
	AccountID   int64
	Month       string
	ValueCents  money.Cents
	Description string
	CreatedAt   time.Time
}

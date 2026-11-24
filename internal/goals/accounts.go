package goals

import (
	"time"

	"github.com/tiagorocha94/household-webapp/internal/money"
)

// Account represents a goal account. Every logged-in user sees every account.
// MonthlyDepositCents is the deposit its projection assumes for each future
// month.
type Account struct {
	ID                  int64
	Name                string
	MonthlyDepositCents money.Cents
	CreatedAt           time.Time
}

// AccountBalance holds the computed balance for an account, derived from
// the goal_account_balance_view.
type AccountBalance struct {
	AccountID           int64
	TotalDepositsCents  money.Cents
	TotalAllocatedCents money.Cents
	TotalExpensesCents  money.Cents
	TotalReleasedCents  money.Cents
	TotalCoveredCents   money.Cents
	BalanceCents        money.Cents
	UnallocatedCents    money.Cents
}

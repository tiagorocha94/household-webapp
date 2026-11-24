package goals

import "github.com/tiagorocha94/household-webapp/internal/money"

// ProjectionMonth represents one month in the account projection.
type ProjectionMonth struct {
	Month            string
	DepositsCents    money.Cents
	AllocationsCents money.Cents
	ExpensesCents    money.Cents
	ReleasedCents    money.Cents
	CoveredCents     money.Cents
	BalanceCents     money.Cents
	UnallocatedCents money.Cents
	IsProjection     bool
}

// MonthActual holds the recorded movement totals of one month.
type MonthActual struct {
	DepositsCents    money.Cents
	AllocationsCents money.Cents
	ExpensesCents    money.Cents
	ReleasedCents    money.Cents
	CoveredCents     money.Cents
}

// GoalPlan is the stored data needed to plan one active goal.
type GoalPlan struct {
	StartMonth      string
	EndMonth        string
	TargetCents     money.Cents
	AllocationCount int
	AllocatedCents  money.Cents
}

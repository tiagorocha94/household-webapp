// Package cashflow exposes a household ledger of money coming in and going
// out, per person and per month, through the read-only JSON API. Every
// movement is typed into the CSV files; nothing here imports bank statements.
package cashflow

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/month"
)

// Category directions.
const (
	DirectionIncome  = "income"
	DirectionExpense = "expense"
)

// syncIDPattern is a lowercase UUID, the form devices make sync IDs in.
var syncIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Errors the movement writes report, so callers can tell them apart with
// errors.Is.
var (
	// ErrUnknownCategory means the category name is not in the category set.
	ErrUnknownCategory = errors.New("unknown category")
	// ErrSyncIDTaken means the sync ID already belongs to another person's
	// movement, which a write must never touch.
	ErrSyncIDTaken = errors.New("sync id belongs to another person")
)

// Category is one entry of the fixed set of movement categories. Its direction
// says whether movements in it count as income or as expenses.
type Category struct {
	ID        int64
	Name      string
	Direction string
	// Icon is a Lucide icon name in kebab-case, such as shopping-basket.
	Icon string
	// Color is a #RRGGBB colour the category is drawn in.
	Color string
	// Role is empty, or where an expense category's money goes: RoleSavings,
	// RoleInvestments, RoleGoals or RoleCredits. Its movements stay expenses.
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// MonthSummary holds the figures of one month that has movements, derived from
// its movements and never stored.
type MonthSummary struct {
	// Month is the calendar month as YYYY-MM.
	Month         string
	IncomeCents   money.Cents
	ExpenseCents  money.Cents
	NetCents      money.Cents
	MovementCount int
	// SavingsRatePct is net divided by income as a percentage scaled by 100:
	// 2500 means 25.00%. It can be negative and is 0 when income is 0.
	SavingsRatePct int
}

// MonthDetail is a month's summary together with where the money went, by
// category, and how it splits between people.
type MonthDetail struct {
	MonthSummary
	ByCategory []CategoryTotal
	// SavedCents is the part of the expenses in savings categories.
	SavedCents money.Cents
	ByUser     []UserTotal
	// PeopleHistory is the income and the money put aside of each person in the
	// month and the five before it, oldest month first, for people with
	// movements in those months.
	PeopleHistory []UserMonthTotal
	// ByGroup is the budget by group, in the order of the groups file. It is
	// empty when no budget groups are configured.
	ByGroup []GroupTotal
}

// Budget is what a phone shows of a month's budget by group: the month's
// totals, the groups, and the months that have movements.
type Budget struct {
	Month        string
	IncomeCents  money.Cents
	ExpenseCents money.Cents
	NetCents     money.Cents
	// ByGroup is empty when no budget groups are configured.
	ByGroup []GroupTotal
	// Months lists the months that have movements, newest first.
	Months []string
}

// BudgetGroup is a named share of the monthly income, from
// cashflow_budget_groups.csv.
type BudgetGroup struct {
	ID       int64
	Name     string
	SharePct int
}

// GroupTotal is one budget group's figures for a month.
type GroupTotal struct {
	Name     string
	SharePct int
	// BudgetCents is the group's share of the month's income. The budgets of
	// all groups add up to the income exactly.
	BudgetCents money.Cents
	// SpentCents is what the group's categories spent in the month.
	SpentCents money.Cents
	// LeftCents is the budget minus what was spent; negative when over.
	LeftCents money.Cents
	// Categories are the group's expense categories with spending in the
	// month, the largest first.
	Categories []CategoryTotal
}

// CategoryTotal is the total of one category in a month.
type CategoryTotal struct {
	CategoryID    int64
	Name          string
	Direction     string
	TotalCents    money.Cents
	MovementCount int
	// Role is the role of the category; empty for none.
	Role string
}

// UserTotal is the income, expenses and net of one person in a month.
type UserTotal struct {
	UserID       int64
	Username     string
	IncomeCents  money.Cents
	ExpenseCents money.Cents
	NetCents     money.Cents
	// PutAsideCents is the part of the expenses in the savings, investments and
	// goals categories. What is left of the expenses was spent; credit payments
	// count as spent.
	PutAsideCents money.Cents
	// CashCents is the person's cash at the end of the month, or nil when they
	// have no typed bank balance.
	CashCents *money.Cents
}

// UserMonthTotal is the income and the money put aside of one person in one
// month, for the history of how much each person puts aside.
type UserMonthTotal struct {
	Month         string
	Username      string
	IncomeCents   money.Cents
	PutAsideCents money.Cents
}

// Movement is a single amount of money that came in or went out on a day.
// Amounts are always positive; Direction says which way the money moved.
type Movement struct {
	ID int64
	// Date is the day of the movement as YYYY-MM-DD.
	Date string
	// Username is the person the movement belongs to.
	Username string
	// Category is the name of the movement's category.
	Category string
	// Direction is the direction of the movement's category.
	Direction   string
	AmountCents money.Cents
	Description string
	CreatedAt   time.Time
}

// MovementWrite is a movement as a device sends it: everything about it except
// its database ID. SyncID names it; UserID is the person it belongs to, taken
// from the device's login, never from the device's request.
type MovementWrite struct {
	SyncID      string
	UserID      int64
	Category    string
	Date        string
	AmountCents money.Cents
	Description string
}

// Validate checks everything that does not need the database: a well-formed
// sync ID, a real calendar day and a positive amount. A problem is a
// bad-request error naming the field.
func (m MovementWrite) Validate() error {
	if !ValidSyncID(m.SyncID) {
		return fmt.Errorf("sync_id %q is not a lowercase UUID: %w", m.SyncID, apperr.ErrBadRequest)
	}
	if err := month.ValidateDay(m.Date); err != nil {
		return fmt.Errorf("date: %w: %w", err, apperr.ErrBadRequest)
	}
	if m.AmountCents <= 0 {
		return fmt.Errorf("amount_cents must be greater than zero: %w", apperr.ErrBadRequest)
	}
	return nil
}

// ValidSyncID reports whether id is a lowercase UUID, such as
// 3f2b8c1e-5d4a-4e7b-9a10-6c2d8e4f1a07.
func ValidSyncID(id string) bool { return syncIDPattern.MatchString(id) }

package store

import (
	"database/sql"
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/goals"
	"github.com/tiagorocha94/household-webapp/internal/money"
)

// ListMonthlyActuals returns recorded movement totals per month.
func (s *Store) ListMonthlyActuals(accountID int64) (map[string]*goals.MonthActual, error) {
	rows, err := s.db.Query(
		`SELECT month, type, SUM(amount_cents)
		 FROM goal_account_movements
		 WHERE account_id = ?
		 GROUP BY month, type
		 ORDER BY month ASC`,
		accountID,
	)
	if err != nil {
		return nil, fmt.Errorf("query monthly actuals: %w", err)
	}
	defer rows.Close()

	result := make(map[string]*goals.MonthActual)
	for rows.Next() {
		var month, movType string
		var amount money.Cents
		if err := rows.Scan(&month, &movType, &amount); err != nil {
			return nil, fmt.Errorf("scan monthly actual: %w", err)
		}
		if result[month] == nil {
			result[month] = &goals.MonthActual{}
		}
		switch movType {
		case goals.MovementDeposit:
			result[month].DepositsCents += amount
		case goals.MovementAllocation:
			result[month].AllocationsCents += amount
		case goals.MovementExpense:
			result[month].ExpensesCents += amount
		case goals.MovementRelease:
			result[month].ReleasedCents += amount
		case goals.MovementCover:
			result[month].CoveredCents += amount
		}
	}
	return result, rows.Err()
}

// ListActiveGoalPlans returns the active goals of an account.
func (s *Store) ListActiveGoalPlans(accountID int64) ([]goals.GoalPlan, error) {
	rows, err := s.db.Query(
		`SELECT g.start_month, g.end_month, g.target_cents,
		        (SELECT COUNT(*) FROM goal_account_movements m
		         WHERE m.goal_id = g.id AND m.type = 'allocation'),
		        (SELECT COALESCE(SUM(m.amount_cents), 0) FROM goal_account_movements m
		         WHERE m.goal_id = g.id AND m.type = 'allocation')
		 FROM goal_account_goals g
		 WHERE g.account_id = ? AND g.status = 'active'`,
		accountID,
	)
	if err != nil {
		return nil, fmt.Errorf("query goal plans: %w", err)
	}
	defer rows.Close()

	var plans []goals.GoalPlan
	for rows.Next() {
		var p goals.GoalPlan
		if err := rows.Scan(&p.StartMonth, &p.EndMonth, &p.TargetCents, &p.AllocationCount, &p.AllocatedCents); err != nil {
			return nil, fmt.Errorf("scan goal plan: %w", err)
		}
		plans = append(plans, p)
	}
	return plans, rows.Err()
}

// GetLastActiveGoalEndMonth returns the latest end month of the account's
// active goals, or an empty string when there are none.
func (s *Store) GetLastActiveGoalEndMonth(accountID int64) (string, error) {
	var endMonth sql.NullString
	err := s.db.QueryRow(
		"SELECT MAX(end_month) FROM goal_account_goals WHERE account_id = ? AND status = 'active'",
		accountID,
	).Scan(&endMonth)
	if err != nil {
		return "", fmt.Errorf("query last goal end month: %w", err)
	}
	return endMonth.String, nil
}

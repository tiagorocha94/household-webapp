package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/goals"
)

// ListGoalSummariesByAccountID returns all goals for the given account with
// computed fields from the summary view. ElapsedMonths is not part of the view;
// the service fills it in from the current month.
func (s *Store) ListGoalSummariesByAccountID(accountID int64) ([]goals.GoalSummary, error) {
	rows, err := s.db.Query(
		`SELECT goal_id, account_id, name, target_cents, start_month, end_month, status,
				total_months, monthly_allocation_cents,
				total_allocated_cents, total_expenses_cents, total_released_cents, total_covered_cents, goal_balance_cents, progress_pct, closing_balance_cents,
				created_at, updated_at
		 FROM goal_account_goal_summary_view
		 WHERE account_id = ?
		 ORDER BY end_month ASC`,
		accountID,
	)
	if err != nil {
		return nil, fmt.Errorf("list goal summaries for account %d: %w", accountID, err)
	}
	defer rows.Close()

	var summaries []goals.GoalSummary
	for rows.Next() {
		var g goals.GoalSummary
		if err := rows.Scan(&g.GoalID, &g.AccountID, &g.Name, &g.TargetCents, &g.StartMonth, &g.EndMonth, &g.Status,
			&g.TotalMonths, &g.MonthlyAllocationCents,
			&g.TotalAllocatedCents, &g.TotalExpensesCents, &g.TotalReleasedCents, &g.TotalCoveredCents, &g.GoalBalanceCents, &g.ProgressPct, &g.ClosingBalanceCents,
			&g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan goal summary: %w", err)
		}
		summaries = append(summaries, g)
	}

	return summaries, rows.Err()
}

// GetGoalSummaryByID retrieves a single goal summary by its ID. ElapsedMonths
// is left for the service to fill in.
func (s *Store) GetGoalSummaryByID(id int64) (*goals.GoalSummary, error) {
	row := s.db.QueryRow(
		`SELECT goal_id, account_id, name, target_cents, start_month, end_month, status,
		        total_months, monthly_allocation_cents,
		        total_allocated_cents, total_expenses_cents, total_released_cents, total_covered_cents, goal_balance_cents, progress_pct, closing_balance_cents,
		        created_at, updated_at
		 FROM goal_account_goal_summary_view
		 WHERE goal_id = ?`,
		id,
	)

	g := &goals.GoalSummary{}
	err := row.Scan(&g.GoalID, &g.AccountID, &g.Name, &g.TargetCents, &g.StartMonth, &g.EndMonth, &g.Status,
		&g.TotalMonths, &g.MonthlyAllocationCents,
		&g.TotalAllocatedCents, &g.TotalExpensesCents, &g.TotalReleasedCents, &g.TotalCoveredCents, &g.GoalBalanceCents, &g.ProgressPct, &g.ClosingBalanceCents,
		&g.CreatedAt, &g.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query goal summary %d: %w", id, err)
	}

	return g, nil
}

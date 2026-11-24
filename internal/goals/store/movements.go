package store

import (
	"database/sql"
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/goals"
	"github.com/tiagorocha94/household-webapp/internal/money"
)

// ListMovementsByAccountID returns all movements for the given account.
// Rows are ordered newest month first. Within a month they follow the reverse
// of the logical event sequence (deposit, allocation, expense, closing), so the
// latest event in a month appears on top: closing, expense, allocation, deposit.
func (s *Store) ListMovementsByAccountID(accountID int64) ([]goals.Movement, error) {
	rows, err := s.db.Query(
		`SELECT m.id, m.account_id, m.goal_id, COALESCE(g.name, ''), m.type, m.amount_cents, m.month, COALESCE(m.description, ''), m.created_at
		 FROM goal_account_movements m
		 LEFT JOIN goal_account_goals g ON m.goal_id = g.id
		 WHERE m.account_id = ?
		 ORDER BY m.month DESC,
		          CASE m.type
		            WHEN 'deposit' THEN 1
		            WHEN 'allocation' THEN 2
		            WHEN 'expense' THEN 3
		            WHEN 'release' THEN 4
		            WHEN 'cover' THEN 4
		            ELSE 5
		          END DESC,
		          m.created_at DESC`,
		accountID,
	)
	if err != nil {
		return nil, fmt.Errorf("list movements for account %d: %w", accountID, err)
	}
	defer rows.Close()

	return scanMovements(rows)
}

// ListMovementsByGoalID returns all movements for a specific goal.
// Rows are ordered newest month first. Within a month they follow the reverse
// of the logical event sequence (allocation, expense, closing), so the latest
// event in a month appears on top: a closing movement sits above the expense and
// allocation for the same month.
func (s *Store) ListMovementsByGoalID(goalID int64) ([]goals.Movement, error) {
	rows, err := s.db.Query(
		`SELECT m.id, m.account_id, m.goal_id, COALESCE(g.name, ''), m.type, m.amount_cents, m.month, COALESCE(m.description, ''), m.created_at
		 FROM goal_account_movements m
		 LEFT JOIN goal_account_goals g ON m.goal_id = g.id
		 WHERE m.goal_id = ?
		 ORDER BY m.month DESC,
		          CASE m.type
		            WHEN 'deposit' THEN 1
		            WHEN 'allocation' THEN 2
		            WHEN 'expense' THEN 3
		            WHEN 'release' THEN 4
		            WHEN 'cover' THEN 4
		            ELSE 5
		          END DESC,
		          m.created_at DESC`,
		goalID,
	)
	if err != nil {
		return nil, fmt.Errorf("list movements for goal %d: %w", goalID, err)
	}
	defer rows.Close()

	return scanMovements(rows)
}

func scanMovements(rows *sql.Rows) ([]goals.Movement, error) {
	var movements []goals.Movement
	for rows.Next() {
		var mv goals.Movement
		var goalID sql.NullInt64
		if err := rows.Scan(&mv.ID, &mv.AccountID, &goalID, &mv.GoalName, &mv.Type, &mv.AmountCents, &mv.Month, &mv.Description, &mv.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan movement: %w", err)
		}
		if goalID.Valid {
			mv.GoalID = &goalID.Int64
		}
		movements = append(movements, mv)
	}

	return movements, rows.Err()
}

// DepositsByMonth returns what the deposits of every account add up to in each
// month, keyed by YYYY-MM.
func (s *Store) DepositsByMonth() (map[string]money.Cents, error) {
	rows, err := s.db.Query(
		`SELECT month, SUM(amount_cents) FROM goal_account_movements
		 WHERE type = 'deposit' GROUP BY month`)
	if err != nil {
		return nil, fmt.Errorf("sum goal deposits by month: %w", err)
	}
	defer rows.Close()

	totals := make(map[string]money.Cents)
	for rows.Next() {
		var month string
		var total money.Cents
		if err := rows.Scan(&month, &total); err != nil {
			return nil, fmt.Errorf("scan goal deposit total: %w", err)
		}
		totals[month] = total
	}
	return totals, rows.Err()
}

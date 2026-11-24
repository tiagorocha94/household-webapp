package store

import (
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/money"
)

// ExpensesByRole returns what the expenses of the categories with the given
// role add up to in each month, keyed by YYYY-MM.
func (s *Store) ExpensesByRole(role string) (map[string]money.Cents, error) {
	rows, err := s.db.Query(
		`SELECT substr(m.date, 1, 7), SUM(m.amount_cents)
		 FROM cashflow_movements m
		 JOIN cashflow_categories c ON c.id = m.category_id
		 WHERE c.direction = 'expense' AND c.role = ?
		 GROUP BY substr(m.date, 1, 7)`,
		role,
	)
	if err != nil {
		return nil, fmt.Errorf("sum %s expenses by month: %w", role, err)
	}
	defer rows.Close()

	totals := make(map[string]money.Cents)
	for rows.Next() {
		var month string
		var total money.Cents
		if err := rows.Scan(&month, &total); err != nil {
			return nil, fmt.Errorf("scan %s total: %w", role, err)
		}
		totals[month] = total
	}
	return totals, rows.Err()
}

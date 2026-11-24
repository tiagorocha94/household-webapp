package store

import (
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/cashflow"
)

// ListUserMonthTotals returns each person's income and the money put aside
// (the expenses of the savings, investments and goals categories) for every
// month from first to last, both included, ordered by month and then username.
// A person with no movements in a month has no row for it.
func (s *Store) ListUserMonthTotals(first, last string) ([]cashflow.UserMonthTotal, error) {
	rows, err := s.db.Query(
		`SELECT substr(m.date, 1, 7) AS month, u.username,
		        COALESCE(SUM(CASE WHEN c.direction = 'income' THEN m.amount_cents ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN c.direction = 'expense'
		                           AND c.role IN ('savings', 'investments', 'goals')
		                          THEN m.amount_cents ELSE 0 END), 0)
		 FROM cashflow_movements m
		 JOIN users u ON u.id = m.user_id
		 JOIN cashflow_categories c ON c.id = m.category_id
		 WHERE substr(m.date, 1, 7) BETWEEN ? AND ?
		 GROUP BY month, u.username
		 ORDER BY month ASC, u.username ASC`,
		first, last,
	)
	if err != nil {
		return nil, fmt.Errorf("list income and money put aside by person: %w", err)
	}
	defer rows.Close()

	var totals []cashflow.UserMonthTotal
	for rows.Next() {
		var t cashflow.UserMonthTotal
		if err := rows.Scan(&t.Month, &t.Username, &t.IncomeCents, &t.PutAsideCents); err != nil {
			return nil, fmt.Errorf("scan income and money put aside: %w", err)
		}
		totals = append(totals, t)
	}
	return totals, rows.Err()
}

package store

import (
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/cashflow"
)

// ListMovementsByMonth returns every movement of a month, newest date first.
// The month is filtered as a date range so the index on date is used.
func (s *Store) ListMovementsByMonth(month string) ([]cashflow.Movement, error) {
	rows, err := s.db.Query(
		`SELECT m.id, m.date, u.username, c.name, c.direction, m.amount_cents,
		        COALESCE(m.description, ''), m.created_at
		 FROM cashflow_movements m
		 JOIN users u ON u.id = m.user_id
		 JOIN cashflow_categories c ON c.id = m.category_id
		 WHERE m.date >= ? || '-01' AND m.date <= ? || '-31'
		 ORDER BY m.date DESC, m.id DESC`,
		month, month,
	)
	if err != nil {
		return nil, fmt.Errorf("list cashflow movements for %s: %w", month, err)
	}
	defer rows.Close()

	var movements []cashflow.Movement
	for rows.Next() {
		var m cashflow.Movement
		if err := rows.Scan(
			&m.ID, &m.Date, &m.Username, &m.Category, &m.Direction, &m.AmountCents,
			&m.Description, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan cashflow movement: %w", err)
		}
		movements = append(movements, m)
	}
	return movements, rows.Err()
}

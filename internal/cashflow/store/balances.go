package store

import (
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/cashflow"
	"github.com/tiagorocha94/household-webapp/internal/money"
)

// ListCashAnchors returns every typed bank balance, oldest month first.
func (s *Store) ListCashAnchors() ([]cashflow.CashAnchor, error) {
	rows, err := s.db.Query(
		`SELECT u.username, b.month, b.balance_cents
		 FROM cashflow_balances b JOIN users u ON u.id = b.user_id
		 ORDER BY b.month ASC, u.username ASC`)
	if err != nil {
		return nil, fmt.Errorf("list cash balances: %w", err)
	}
	defer rows.Close()

	var anchors []cashflow.CashAnchor
	for rows.Next() {
		var a cashflow.CashAnchor
		if err := rows.Scan(&a.User, &a.Month, &a.BalanceCents); err != nil {
			return nil, fmt.Errorf("scan cash balance: %w", err)
		}
		anchors = append(anchors, a)
	}
	return anchors, rows.Err()
}

// NetsByUserMonth returns each person's income minus expenses by YYYY-MM
// month, keyed by username.
func (s *Store) NetsByUserMonth() (map[string]map[string]money.Cents, error) {
	rows, err := s.db.Query(
		`SELECT u.username, substr(m.date, 1, 7),
		        SUM(CASE WHEN c.direction = 'income' THEN m.amount_cents ELSE -m.amount_cents END)
		 FROM cashflow_movements m
		 JOIN users u ON u.id = m.user_id
		 JOIN cashflow_categories c ON c.id = m.category_id
		 GROUP BY u.username, substr(m.date, 1, 7)`)
	if err != nil {
		return nil, fmt.Errorf("sum net cashflow by person and month: %w", err)
	}
	defer rows.Close()

	nets := make(map[string]map[string]money.Cents)
	for rows.Next() {
		var user, month string
		var net money.Cents
		if err := rows.Scan(&user, &month, &net); err != nil {
			return nil, fmt.Errorf("scan net cashflow: %w", err)
		}
		if nets[user] == nil {
			nets[user] = make(map[string]money.Cents)
		}
		nets[user][month] = net
	}
	return nets, rows.Err()
}

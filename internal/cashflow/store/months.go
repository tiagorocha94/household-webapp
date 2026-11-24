package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/cashflow"
)

const monthSummaryColumns = `month, income_cents, expense_cents, net_cents,
savings_rate_pct, movement_count`

// ListMonthSummaries returns one summary per month that has movements, newest
// first.
func (s *Store) ListMonthSummaries() ([]cashflow.MonthSummary, error) {
	rows, err := s.db.Query(
		`SELECT ` + monthSummaryColumns + `
		 FROM cashflow_month_summary_view ORDER BY month DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list cashflow month summaries: %w", err)
	}
	defer rows.Close()

	var summaries []cashflow.MonthSummary
	for rows.Next() {
		var summary cashflow.MonthSummary
		if err := scanMonthSummary(rows, &summary); err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	return summaries, rows.Err()
}

// GetMonthSummary returns the summary of a month, or nil when the month has no
// movements.
func (s *Store) GetMonthSummary(month string) (*cashflow.MonthSummary, error) {
	row := s.db.QueryRow(
		`SELECT `+monthSummaryColumns+` FROM cashflow_month_summary_view WHERE month = ?`, month,
	)

	var summary cashflow.MonthSummary
	if err := scanMonthSummary(row, &summary); errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &summary, nil
}

// ListMonthCategoryTotals returns the total of each category with movements in
// a month: expenses before income, the largest first within each.
func (s *Store) ListMonthCategoryTotals(month string) ([]cashflow.CategoryTotal, error) {
	rows, err := s.db.Query(
		`SELECT category_id, category_name, direction, total_cents, movement_count
		 FROM cashflow_month_category_view WHERE month = ?
		 ORDER BY direction ASC, total_cents DESC, category_name ASC`,
		month,
	)
	if err != nil {
		return nil, fmt.Errorf("list cashflow category totals for %s: %w", month, err)
	}
	defer rows.Close()

	var totals []cashflow.CategoryTotal
	for rows.Next() {
		var t cashflow.CategoryTotal
		if err := rows.Scan(&t.CategoryID, &t.Name, &t.Direction, &t.TotalCents, &t.MovementCount); err != nil {
			return nil, fmt.Errorf("scan cashflow category total: %w", err)
		}
		totals = append(totals, t)
	}
	return totals, rows.Err()
}

// ListMonthUserTotals returns the totals of each person with movements in a
// month, ordered by username.
func (s *Store) ListMonthUserTotals(month string) ([]cashflow.UserTotal, error) {
	rows, err := s.db.Query(
		`SELECT user_id, username, income_cents, expense_cents, net_cents
		 FROM cashflow_month_user_view WHERE month = ?
		 ORDER BY username ASC`,
		month,
	)
	if err != nil {
		return nil, fmt.Errorf("list cashflow user totals for %s: %w", month, err)
	}
	defer rows.Close()

	var totals []cashflow.UserTotal
	for rows.Next() {
		var t cashflow.UserTotal
		if err := rows.Scan(&t.UserID, &t.Username, &t.IncomeCents, &t.ExpenseCents, &t.NetCents); err != nil {
			return nil, fmt.Errorf("scan cashflow user total: %w", err)
		}
		totals = append(totals, t)
	}
	return totals, rows.Err()
}

func scanMonthSummary(row rowScanner, summary *cashflow.MonthSummary) error {
	if err := row.Scan(
		&summary.Month, &summary.IncomeCents, &summary.ExpenseCents, &summary.NetCents,
		&summary.SavingsRatePct, &summary.MovementCount,
	); err != nil {
		return fmt.Errorf("scan cashflow month summary: %w", err)
	}
	return nil
}

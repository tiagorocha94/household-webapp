package store

import (
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/investments"
	"github.com/tiagorocha94/household-webapp/internal/money"
)

// ListContributionsByInstrumentID returns every contribution to an
// instrument, newest first.
func (s *Store) ListContributionsByInstrumentID(instrumentID int64) ([]investments.Contribution, error) {
	rows, err := s.db.Query(
		`SELECT id, instrument_id, month, amount_cents, COALESCE(description, ''),
		        created_at, updated_at
		 FROM investment_contributions WHERE instrument_id = ?
		 ORDER BY month DESC, id DESC`,
		instrumentID,
	)
	if err != nil {
		return nil, fmt.Errorf("list contributions for investment instrument %d: %w", instrumentID, err)
	}
	defer rows.Close()

	var contributions []investments.Contribution
	for rows.Next() {
		var c investments.Contribution
		if err := rows.Scan(
			&c.ID, &c.InstrumentID, &c.Month, &c.AmountCents, &c.Description,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan investment contribution: %w", err)
		}
		contributions = append(contributions, c)
	}
	return contributions, rows.Err()
}

// ContributionsByMonth returns what the contributions to every instrument add up
// to in each month, keyed by YYYY-MM.
func (s *Store) ContributionsByMonth() (map[string]money.Cents, error) {
	rows, err := s.db.Query(
		`SELECT month, SUM(amount_cents) FROM investment_contributions GROUP BY month`)
	if err != nil {
		return nil, fmt.Errorf("sum investment contributions by month: %w", err)
	}
	defer rows.Close()

	totals := make(map[string]money.Cents)
	for rows.Next() {
		var month string
		var total money.Cents
		if err := rows.Scan(&month, &total); err != nil {
			return nil, fmt.Errorf("scan investment contribution total: %w", err)
		}
		totals[month] = total
	}
	return totals, rows.Err()
}

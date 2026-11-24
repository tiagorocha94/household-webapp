package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/investments"
)

const entitySummaryColumns = `entity_id, name, kind, COALESCE(username, ''), created_at, updated_at,
instrument_count, allocated_cents, current_value_cents, gain_cents, gain_pct`

// GetEntityByID returns an entity by ID, or nil when it does not exist.
func (s *Store) GetEntityByID(id int64) (*investments.Entity, error) {
	var entity investments.Entity
	err := s.db.QueryRow(
		`SELECT e.id, e.name, e.kind, COALESCE(u.username, ''), e.created_at, e.updated_at
		 FROM investment_entities e LEFT JOIN users u ON u.id = e.user_id WHERE e.id = ?`, id,
	).Scan(&entity.ID, &entity.Name, &entity.Kind, &entity.Username, &entity.CreatedAt, &entity.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get investment entity %d: %w", id, err)
	}
	return &entity, nil
}

// ListEntitySummaries returns all entities, the largest allocation first.
func (s *Store) ListEntitySummaries() ([]investments.EntitySummary, error) {
	rows, err := s.db.Query(
		`SELECT ` + entitySummaryColumns + `
		 FROM investment_entity_summary_view
		 ORDER BY allocated_cents DESC, name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list investment entities: %w", err)
	}
	defer rows.Close()

	var summaries []investments.EntitySummary
	for rows.Next() {
		var summary investments.EntitySummary
		if err := scanEntitySummary(rows, &summary); err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	return summaries, rows.Err()
}

// GetEntitySummaryByID returns an entity summary by ID, or nil when absent.
func (s *Store) GetEntitySummaryByID(id int64) (*investments.EntitySummary, error) {
	row := s.db.QueryRow(
		`SELECT `+entitySummaryColumns+` FROM investment_entity_summary_view WHERE entity_id = ?`, id,
	)

	var summary investments.EntitySummary
	if err := scanEntitySummary(row, &summary); errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &summary, nil
}

func scanEntitySummary(row rowScanner, summary *investments.EntitySummary) error {
	err := row.Scan(
		&summary.ID, &summary.Name, &summary.Kind, &summary.Username, &summary.CreatedAt, &summary.UpdatedAt,
		&summary.InstrumentCount, &summary.AllocatedCents, &summary.CurrentValueCents,
		&summary.GainCents, &summary.GainPct,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err != nil {
		return fmt.Errorf("scan investment entity summary: %w", err)
	}
	return nil
}

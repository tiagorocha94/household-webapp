package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/investments"
)

const instrumentSummaryColumns = `instrument_id, entity_id, entity_name, name, type,
COALESCE(notes, ''), COALESCE(username, ''), created_at, updated_at, contribution_count, snapshot_count,
allocated_cents, current_value_cents, gain_cents, gain_pct,
COALESCE(last_snapshot_month, '')`

// GetInstrumentByID returns an instrument by ID, or nil when it does not exist.
func (s *Store) GetInstrumentByID(id int64) (*investments.Instrument, error) {
	var instrument investments.Instrument
	err := s.db.QueryRow(
		`SELECT i.id, i.entity_id, i.name, i.type, COALESCE(i.notes, ''),
		        COALESCE(u.username, ''), i.created_at, i.updated_at
		 FROM investment_instruments i
		 JOIN investment_entities e ON e.id = i.entity_id
		 LEFT JOIN users u ON u.id = e.user_id
		 WHERE i.id = ?`, id,
	).Scan(
		&instrument.ID, &instrument.EntityID, &instrument.Name, &instrument.Type,
		&instrument.Notes, &instrument.Username, &instrument.CreatedAt, &instrument.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get investment instrument %d: %w", id, err)
	}
	return &instrument, nil
}

// ListInstrumentSummariesByEntityID returns the instruments of an entity, the
// most valuable first.
func (s *Store) ListInstrumentSummariesByEntityID(entityID int64) ([]investments.InstrumentSummary, error) {
	rows, err := s.db.Query(
		`SELECT `+instrumentSummaryColumns+`
		 FROM investment_instrument_summary_view WHERE entity_id = ?
		 ORDER BY current_value_cents DESC, name ASC`,
		entityID,
	)
	if err != nil {
		return nil, fmt.Errorf("list instruments for investment entity %d: %w", entityID, err)
	}
	defer rows.Close()

	var summaries []investments.InstrumentSummary
	for rows.Next() {
		var summary investments.InstrumentSummary
		if err := scanInstrumentSummary(rows, &summary); err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	return summaries, rows.Err()
}

// GetInstrumentSummaryByID returns an instrument summary by ID, or nil when absent.
func (s *Store) GetInstrumentSummaryByID(id int64) (*investments.InstrumentSummary, error) {
	row := s.db.QueryRow(
		`SELECT `+instrumentSummaryColumns+` FROM investment_instrument_summary_view WHERE instrument_id = ?`, id,
	)

	var summary investments.InstrumentSummary
	if err := scanInstrumentSummary(row, &summary); errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &summary, nil
}

func scanInstrumentSummary(row rowScanner, summary *investments.InstrumentSummary) error {
	err := row.Scan(
		&summary.ID, &summary.EntityID, &summary.EntityName, &summary.Name, &summary.Type,
		&summary.Notes, &summary.Username, &summary.CreatedAt, &summary.UpdatedAt,
		&summary.ContributionCount, &summary.SnapshotCount,
		&summary.AllocatedCents, &summary.CurrentValueCents, &summary.GainCents,
		&summary.GainPct, &summary.LastSnapshotMonth,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err != nil {
		return fmt.Errorf("scan investment instrument summary: %w", err)
	}
	return nil
}

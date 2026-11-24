package store

import (
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/investments"
)

// ListSnapshotsByInstrumentID returns every value snapshot of an instrument,
// newest first.
func (s *Store) ListSnapshotsByInstrumentID(instrumentID int64) ([]investments.Snapshot, error) {
	rows, err := s.db.Query(
		`SELECT id, instrument_id, month, value_cents, COALESCE(description, ''),
		        created_at, updated_at
		 FROM investment_snapshots WHERE instrument_id = ?
		 ORDER BY month DESC, id DESC`,
		instrumentID,
	)
	if err != nil {
		return nil, fmt.Errorf("list snapshots for investment instrument %d: %w", instrumentID, err)
	}
	defer rows.Close()

	var snapshots []investments.Snapshot
	for rows.Next() {
		var sn investments.Snapshot
		if err := rows.Scan(
			&sn.ID, &sn.InstrumentID, &sn.Month, &sn.ValueCents, &sn.Description,
			&sn.CreatedAt, &sn.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan investment snapshot: %w", err)
		}
		snapshots = append(snapshots, sn)
	}
	return snapshots, rows.Err()
}

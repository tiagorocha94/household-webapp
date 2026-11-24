package service

import (
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/investments"
)

// GetInstrumentSummary returns the summary of one instrument.
func (s *Service) GetInstrumentSummary(instrumentID int64) (*investments.InstrumentSummary, error) {
	summary, err := s.store.GetInstrumentSummaryByID(instrumentID)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return nil, fmt.Errorf("get investment instrument %d: %w", instrumentID, apperr.ErrNotFound)
	}
	return summary, nil
}

// ListContributions returns an instrument's contribution history.
func (s *Service) ListContributions(instrumentID int64) ([]investments.Contribution, error) {
	if _, err := s.getInstrument(instrumentID); err != nil {
		return nil, err
	}
	return s.store.ListContributionsByInstrumentID(instrumentID)
}

// ListSnapshots returns an instrument's value history.
func (s *Service) ListSnapshots(instrumentID int64) ([]investments.Snapshot, error) {
	if _, err := s.getInstrument(instrumentID); err != nil {
		return nil, err
	}
	return s.store.ListSnapshotsByInstrumentID(instrumentID)
}

package service

import (
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/investments"
)

// ListEntities returns summaries of all entities.
func (s *Service) ListEntities() ([]investments.EntitySummary, error) {
	return s.store.ListEntitySummaries()
}

// GetEntitySummary returns the summary of one entity.
func (s *Service) GetEntitySummary(entityID int64) (*investments.EntitySummary, error) {
	summary, err := s.store.GetEntitySummaryByID(entityID)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return nil, fmt.Errorf("get investment entity %d: %w", entityID, apperr.ErrNotFound)
	}
	return summary, nil
}

// ListInstrumentsByEntity returns summaries of an entity's instruments.
func (s *Service) ListInstrumentsByEntity(entityID int64) ([]investments.InstrumentSummary, error) {
	if _, err := s.getEntity(entityID); err != nil {
		return nil, err
	}
	return s.store.ListInstrumentSummariesByEntityID(entityID)
}

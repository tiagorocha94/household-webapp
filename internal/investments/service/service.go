package service

import (
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/investments"
	"github.com/tiagorocha94/household-webapp/internal/investments/store"
	"github.com/tiagorocha94/household-webapp/internal/money"
)

// Service handles investment reads.
type Service struct {
	store *store.Store
}

// New creates a Service backed by the given Store.
func New(store *store.Store) *Service {
	return &Service{store: store}
}

// ContributionsByMonth returns what the contributions to every instrument add up
// to in each month, keyed by YYYY-MM.
func (s *Service) ContributionsByMonth() (map[string]money.Cents, error) {
	return s.store.ContributionsByMonth()
}

func (s *Service) getEntity(id int64) (*investments.Entity, error) {
	entity, err := s.store.GetEntityByID(id)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, fmt.Errorf("get investment entity %d: %w", id, apperr.ErrNotFound)
	}
	return entity, nil
}

func (s *Service) getInstrument(id int64) (*investments.Instrument, error) {
	instrument, err := s.store.GetInstrumentByID(id)
	if err != nil {
		return nil, err
	}
	if instrument == nil {
		return nil, fmt.Errorf("get investment instrument %d: %w", id, apperr.ErrNotFound)
	}
	return instrument, nil
}

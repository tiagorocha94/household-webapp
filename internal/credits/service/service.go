package service

import (
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/credits"
	"github.com/tiagorocha94/household-webapp/internal/credits/store"
)

// Service handles credit contract reads.
type Service struct {
	store *store.Store
}

// New creates a Service backed by the given Store.
func New(store *store.Store) *Service {
	return &Service{store: store}
}

func (s *Service) getContract(id int64) (*credits.Contract, error) {
	contract, err := s.store.GetContractByID(id)
	if err != nil {
		return nil, err
	}
	if contract == nil {
		return nil, fmt.Errorf("get credit contract %d: %w", id, apperr.ErrNotFound)
	}
	return contract, nil
}

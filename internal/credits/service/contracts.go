package service

import (
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/credits"
)

// ListContracts returns summaries of all contracts.
func (s *Service) ListContracts() ([]credits.ContractSummary, error) {
	return s.store.ListContractSummaries()
}

// GetContractSummary returns the summary of one contract.
func (s *Service) GetContractSummary(contractID int64) (*credits.ContractSummary, error) {
	summary, err := s.store.GetContractSummaryByID(contractID)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return nil, fmt.Errorf("get credit contract %d: %w", contractID, apperr.ErrNotFound)
	}
	return summary, nil
}

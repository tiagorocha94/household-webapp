package service

import (
	"github.com/tiagorocha94/household-webapp/internal/credits"
)

// ListPayments returns a contract's payment history.
func (s *Service) ListPayments(contractID int64) ([]credits.Payment, error) {
	if _, err := s.getContract(contractID); err != nil {
		return nil, err
	}
	return s.store.ListPaymentsByContractID(contractID)
}

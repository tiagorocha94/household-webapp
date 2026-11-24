package service

import (
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/goals"
)

// ListAccounts returns all accounts.
func (s *Service) ListAccounts() ([]goals.Account, error) {
	return s.store.ListAccounts()
}

// GetAccount retrieves an account by ID.
func (s *Service) GetAccount(accountID int64) (*goals.Account, error) {
	account, err := s.store.GetAccountByID(accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, fmt.Errorf("get account %d: %w", accountID, apperr.ErrNotFound)
	}
	return account, nil
}

// GetBalance retrieves the computed account balance.
func (s *Service) GetBalance(accountID int64) (*goals.AccountBalance, error) {
	if _, err := s.GetAccount(accountID); err != nil {
		return nil, err
	}
	return s.store.GetAccountBalance(accountID)
}

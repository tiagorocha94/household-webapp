// Package service holds the savings read logic.
package service

import (
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/savings"
	"github.com/tiagorocha94/household-webapp/internal/savings/store"
)

// Service handles savings reads.
type Service struct {
	store *store.Store
}

// New creates a Service backed by the given Store.
func New(store *store.Store) *Service {
	return &Service{store: store}
}

// ListAccounts returns every account with its figures, ordered by name.
func (s *Service) ListAccounts() ([]savings.AccountSummary, error) {
	accounts, err := s.store.ListAccounts()
	if err != nil {
		return nil, err
	}
	out := make([]savings.AccountSummary, 0, len(accounts))
	for _, a := range accounts {
		summary, err := s.summarize(a)
		if err != nil {
			return nil, err
		}
		out = append(out, summary)
	}
	return out, nil
}

// DepositsByMonth returns what the deposits of every account add up to in each
// month, keyed by YYYY-MM.
func (s *Service) DepositsByMonth() (map[string]money.Cents, error) {
	return s.store.DepositsByMonth()
}

// GetAccount returns one account with its figures.
func (s *Service) GetAccount(id int64) (*savings.AccountSummary, error) {
	account, err := s.getAccount(id)
	if err != nil {
		return nil, err
	}
	summary, err := s.summarize(*account)
	if err != nil {
		return nil, err
	}
	return &summary, nil
}

// ListMovements returns the movements of an account, newest month first.
func (s *Service) ListMovements(accountID int64) ([]savings.Movement, error) {
	if _, err := s.getAccount(accountID); err != nil {
		return nil, err
	}
	return s.store.ListMovements(accountID)
}

// ListSnapshots returns the snapshots of an account, newest month first.
func (s *Service) ListSnapshots(accountID int64) ([]savings.Snapshot, error) {
	if _, err := s.getAccount(accountID); err != nil {
		return nil, err
	}
	return s.store.ListSnapshots(accountID)
}

func (s *Service) getAccount(id int64) (*savings.Account, error) {
	account, err := s.store.GetAccountByID(id)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, fmt.Errorf("get savings account %d: %w", id, apperr.ErrNotFound)
	}
	return account, nil
}

func (s *Service) summarize(a savings.Account) (savings.AccountSummary, error) {
	movements, err := s.store.ListMovements(a.ID)
	if err != nil {
		return savings.AccountSummary{}, err
	}
	snapshots, err := s.store.ListSnapshots(a.ID)
	if err != nil {
		return savings.AccountSummary{}, err
	}
	return savings.Summarize(a, movements, snapshots), nil
}

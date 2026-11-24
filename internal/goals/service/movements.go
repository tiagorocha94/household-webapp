package service

import (
	"github.com/tiagorocha94/household-webapp/internal/goals"
	"github.com/tiagorocha94/household-webapp/internal/money"
)

// ListMovements returns all movements of an account.
func (s *Service) ListMovements(accountID int64) ([]goals.Movement, error) {
	if _, err := s.GetAccount(accountID); err != nil {
		return nil, err
	}
	return s.store.ListMovementsByAccountID(accountID)
}

// ListGoalMovements returns all movements of a goal.
func (s *Service) ListGoalMovements(goalID int64) ([]goals.Movement, error) {
	if _, err := s.GetGoalSummary(goalID); err != nil {
		return nil, err
	}
	return s.store.ListMovementsByGoalID(goalID)
}

// DepositsByMonth returns what the deposits of every account add up to in each
// month, keyed by YYYY-MM.
func (s *Service) DepositsByMonth() (map[string]money.Cents, error) {
	return s.store.DepositsByMonth()
}

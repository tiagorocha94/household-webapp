package service

import (
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/goals"
	"github.com/tiagorocha94/household-webapp/internal/month"
)

// ListGoalSummaries returns all goals of an account with computed fields.
func (s *Service) ListGoalSummaries(accountID int64) ([]goals.GoalSummary, error) {
	if _, err := s.GetAccount(accountID); err != nil {
		return nil, err
	}
	goals, err := s.store.ListGoalSummariesByAccountID(accountID)
	if err != nil {
		return nil, err
	}
	for i := range goals {
		if err := s.setElapsedMonths(&goals[i]); err != nil {
			return nil, err
		}
	}
	return goals, nil
}

// GetGoalSummary retrieves a single goal summary by ID.
func (s *Service) GetGoalSummary(id int64) (*goals.GoalSummary, error) {
	goal, err := s.store.GetGoalSummaryByID(id)
	if err != nil {
		return nil, err
	}
	if goal == nil {
		return nil, fmt.Errorf("get goal %d: %w", id, apperr.ErrNotFound)
	}
	if err := s.setElapsedMonths(goal); err != nil {
		return nil, err
	}
	return goal, nil
}

// setElapsedMonths fills in how many months of the goal's schedule have begun,
// counting the current month in the local time zone, like sync and the
// projection do.
func (s *Service) setElapsedMonths(g *goals.GoalSummary) error {
	elapsed, err := month.Elapsed(g.StartMonth, g.EndMonth, month.Current(s.now()))
	if err != nil {
		return fmt.Errorf("count elapsed months of goal %d: %w", g.GoalID, err)
	}
	g.ElapsedMonths = elapsed
	return nil
}

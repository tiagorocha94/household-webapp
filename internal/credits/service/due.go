package service

import (
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/credits"
)

// ListDue returns the credits that are not paid off, have started by the month,
// expect their payment on a day, and have no regular payment in the month yet,
// with the amount of the last regular payment as the expected instalment.
func (s *Service) ListDue(month string) ([]credits.DueCredit, error) {
	contracts, err := s.store.ListContractSummaries()
	if err != nil {
		return nil, err
	}
	due := []credits.DueCredit{}
	for _, c := range contracts {
		if c.Status == "paid_off" || c.ExpectedDay == 0 || month < c.StartMonth {
			continue
		}
		payments, err := s.store.ListPaymentsByContractID(c.ID)
		if err != nil {
			return nil, fmt.Errorf("list the payments of credit %d: %w", c.ID, err)
		}
		d := credits.DueCredit{ContractID: c.ID, Name: c.Name, Month: month, ExpectedDay: c.ExpectedDay}
		paid, last := false, ""
		for _, p := range payments {
			if p.Type != credits.PaymentRegular {
				continue
			}
			if p.Month == month {
				paid = true
				break
			}
			if p.Month > last {
				last = p.Month
				d.PrincipalCents, d.InterestCents = p.PrincipalCents, p.InterestCents
			}
		}
		if !paid {
			due = append(due, d)
		}
	}
	return due, nil
}

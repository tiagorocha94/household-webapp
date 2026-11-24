// Package credits exposes credit contracts and payment history
// through the read-only JSON API.
package credits

import (
	"time"

	"github.com/tiagorocha94/household-webapp/internal/money"
)

// Credit categories.
const (
	CategoryMortgage = "mortgage"
	CategoryCar      = "car"
	CategoryPersonal = "personal"
	CategoryOther    = "other"
)

// Credit payment types.
const (
	PaymentRegular       = "regular"
	PaymentExtraordinary = "extraordinary"
)

// Contract represents a credit agreement.
type Contract struct {
	ID                     int64
	Name                   string
	Category               string
	ContractNumber         string
	Lender                 string
	OriginalPrincipalCents money.Cents
	StartMonth             string
	ExpectedEndMonth       string
	// ExpectedDay is the day of the month by which a payment should be
	// recorded; 0 when none is expected.
	ExpectedDay int
	Notes       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ContractSummary contains a contract and totals derived from its payments.
type ContractSummary struct {
	Contract
	Status                    string
	PaymentCount              int
	TotalPrincipalPaidCents   money.Cents
	TotalInterestPaidCents    money.Cents
	TotalPaidCents            money.Cents
	OutstandingPrincipalCents money.Cents
	ProgressPct               int
	LastPaymentMonth          string
}

// DueCredit is a credit whose regular payment of a month has an expected day
// and is not recorded yet. The instalment is the amount of its last regular
// payment, and zero when there is none.
type DueCredit struct {
	ContractID     int64
	Name           string
	Month          string
	ExpectedDay    int
	PrincipalCents money.Cents
	InterestCents  money.Cents
}

// Payment represents an actual credit payment split into principal and interest.
type Payment struct {
	ID             int64
	ContractID     int64
	Month          string
	Type           string
	PrincipalCents money.Cents
	InterestCents  money.Cents
	Description    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// TotalCents returns the complete amount paid for this payment.
func (p Payment) TotalCents() money.Cents {
	return p.PrincipalCents + p.InterestCents
}

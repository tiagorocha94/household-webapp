package api

import (
	"net/http"
	"time"

	httpapi "github.com/tiagorocha94/household-webapp/internal/api"
	"github.com/tiagorocha94/household-webapp/internal/credits"
	"github.com/tiagorocha94/household-webapp/internal/credits/service"
	"github.com/tiagorocha94/household-webapp/internal/money"
)

// Handler serves the read-only credits JSON API. It parses requests and calls
// the service layer; HTTP status mapping is done by the server adapter.
type Handler struct {
	service *service.Service
}

type apiContract struct {
	ID                        int64       `json:"id"`
	Name                      string      `json:"name"`
	Category                  string      `json:"category"`
	ContractNumber            string      `json:"contractNumber"`
	Lender                    string      `json:"lender"`
	OriginalPrincipalCents    money.Cents `json:"originalPrincipalCents"`
	StartMonth                string      `json:"startMonth"`
	ExpectedEndMonth          *string     `json:"expectedEndMonth"`
	Status                    string      `json:"status"`
	Notes                     string      `json:"notes"`
	PaymentCount              int         `json:"paymentCount"`
	PrincipalPaidCents        money.Cents `json:"principalPaidCents"`
	InterestPaidCents         money.Cents `json:"interestPaidCents"`
	TotalPaidCents            money.Cents `json:"totalPaidCents"`
	OutstandingPrincipalCents money.Cents `json:"outstandingPrincipalCents"`
	ProgressPct               int         `json:"progressPct"`
	LastPaymentMonth          *string     `json:"lastPaymentMonth"`
	ExpectedDay               *int        `json:"expectedDay"`
}

type apiDueCredit struct {
	ID             int64       `json:"id"`
	Name           string      `json:"name"`
	Month          string      `json:"month"`
	ExpectedDay    int         `json:"expectedDay"`
	PrincipalCents money.Cents `json:"principalCents"`
	InterestCents  money.Cents `json:"interestCents"`
	TotalCents     money.Cents `json:"totalCents"`
}

type apiPayment struct {
	ID             int64       `json:"id"`
	Month          string      `json:"month"`
	Type           string      `json:"type"`
	PrincipalCents money.Cents `json:"principalCents"`
	InterestCents  money.Cents `json:"interestCents"`
	TotalCents     money.Cents `json:"totalCents"`
	Description    string      `json:"description"`
	CreatedAt      time.Time   `json:"createdAt"`
}

// New creates a credit Handler.
func New(service *service.Service) *Handler {
	return &Handler{service: service}
}

// ListContracts returns every contract.
func (h *Handler) ListContracts(_ *http.Request) (any, error) {
	contracts, err := h.service.ListContracts()
	if err != nil {
		return nil, err
	}
	out := make([]apiContract, 0, len(contracts))
	for _, c := range contracts {
		out = append(out, toAPIContract(c))
	}
	return out, nil
}

// GetContract returns one contract with its summary.
func (h *Handler) GetContract(r *http.Request) (any, error) {
	id, err := httpapi.PathID(r, "credit contract id")
	if err != nil {
		return nil, err
	}
	contract, err := h.service.GetContractSummary(id)
	if err != nil {
		return nil, err
	}
	return toAPIContract(*contract), nil
}

// ListDue returns the credits whose regular payment of the current month is
// expected and not recorded yet.
func (h *Handler) ListDue(_ *http.Request) (any, error) {
	due, err := h.service.ListDue(time.Now().Format("2006-01"))
	if err != nil {
		return nil, err
	}
	out := make([]apiDueCredit, 0, len(due))
	for _, d := range due {
		out = append(out, apiDueCredit{
			ID: d.ContractID, Name: d.Name, Month: d.Month, ExpectedDay: d.ExpectedDay,
			PrincipalCents: d.PrincipalCents, InterestCents: d.InterestCents,
			TotalCents: d.PrincipalCents + d.InterestCents,
		})
	}
	return out, nil
}

// ListPayments returns a contract's payment history, newest first.
func (h *Handler) ListPayments(r *http.Request) (any, error) {
	id, err := httpapi.PathID(r, "credit contract id")
	if err != nil {
		return nil, err
	}
	payments, err := h.service.ListPayments(id)
	if err != nil {
		return nil, err
	}
	out := make([]apiPayment, 0, len(payments))
	for _, p := range payments {
		out = append(out, apiPayment{
			ID: p.ID, Month: p.Month, Type: p.Type, PrincipalCents: p.PrincipalCents,
			InterestCents: p.InterestCents, TotalCents: p.TotalCents(), Description: p.Description,
			CreatedAt: p.CreatedAt,
		})
	}
	return out, nil
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toAPIContract(c credits.ContractSummary) apiContract {
	var day *int
	if c.ExpectedDay > 0 {
		day = &c.ExpectedDay
	}
	return apiContract{
		ExpectedDay: day,
		ID:          c.ID, Name: c.Name,
		Category: c.Category, ContractNumber: c.ContractNumber, Lender: c.Lender,
		OriginalPrincipalCents: c.OriginalPrincipalCents, StartMonth: c.StartMonth,
		ExpectedEndMonth: nullable(c.ExpectedEndMonth), Status: c.Status, Notes: c.Notes,
		PaymentCount: c.PaymentCount, PrincipalPaidCents: c.TotalPrincipalPaidCents,
		InterestPaidCents: c.TotalInterestPaidCents, TotalPaidCents: c.TotalPaidCents,
		OutstandingPrincipalCents: c.OutstandingPrincipalCents, ProgressPct: c.ProgressPct,
		LastPaymentMonth: nullable(c.LastPaymentMonth),
	}
}

// Package api serves the read-only savings JSON API. It calls the savings
// service and maps its result to the wire format; HTTP status mapping is done
// by the server adapter.
package api

import (
	"net/http"
	"time"

	httpapi "github.com/tiagorocha94/household-webapp/internal/api"
	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/savings"
	"github.com/tiagorocha94/household-webapp/internal/savings/service"
)

// Handler serves the savings API.
type Handler struct {
	service *service.Service
}

type apiAccount struct {
	ID                int64       `json:"id"`
	Name              string      `json:"name"`
	Owner             *string     `json:"owner"`
	TargetCents       *int64      `json:"targetCents"`
	Notes             string      `json:"notes"`
	BalanceCents      money.Cents `json:"balanceCents"`
	DepositedCents    money.Cents `json:"depositedCents"`
	WithdrawnCents    money.Cents `json:"withdrawnCents"`
	InterestCents     money.Cents `json:"interestCents"`
	ProgressPct       int         `json:"progressPct"`
	MovementCount     int         `json:"movementCount"`
	SnapshotCount     int         `json:"snapshotCount"`
	LastSnapshotMonth *string     `json:"lastSnapshotMonth"`
}

type apiMovement struct {
	ID          int64       `json:"id"`
	Month       string      `json:"month"`
	Type        string      `json:"type"`
	AmountCents money.Cents `json:"amountCents"`
	Description string      `json:"description"`
	CreatedAt   time.Time   `json:"createdAt"`
}

type apiSnapshot struct {
	ID          int64       `json:"id"`
	Month       string      `json:"month"`
	ValueCents  money.Cents `json:"valueCents"`
	Description string      `json:"description"`
	CreatedAt   time.Time   `json:"createdAt"`
}

// New creates a savings Handler.
func New(service *service.Service) *Handler {
	return &Handler{service: service}
}

// ListAccounts returns every savings account with its figures.
func (h *Handler) ListAccounts(_ *http.Request) (any, error) {
	accounts, err := h.service.ListAccounts()
	if err != nil {
		return nil, err
	}
	out := make([]apiAccount, 0, len(accounts))
	for _, a := range accounts {
		out = append(out, toAPIAccount(a))
	}
	return out, nil
}

// GetAccount returns one savings account with its figures.
func (h *Handler) GetAccount(r *http.Request) (any, error) {
	id, err := httpapi.PathID(r, "savings account id")
	if err != nil {
		return nil, err
	}
	account, err := h.service.GetAccount(id)
	if err != nil {
		return nil, err
	}
	return toAPIAccount(*account), nil
}

// ListMovements returns the movements of an account, newest month first.
func (h *Handler) ListMovements(r *http.Request) (any, error) {
	id, err := httpapi.PathID(r, "savings account id")
	if err != nil {
		return nil, err
	}
	movements, err := h.service.ListMovements(id)
	if err != nil {
		return nil, err
	}
	out := make([]apiMovement, 0, len(movements))
	for _, m := range movements {
		out = append(out, apiMovement{
			ID: m.ID, Month: m.Month, Type: m.Type, AmountCents: m.AmountCents,
			Description: m.Description, CreatedAt: m.CreatedAt,
		})
	}
	return out, nil
}

// ListSnapshots returns the snapshots of an account, newest month first.
func (h *Handler) ListSnapshots(r *http.Request) (any, error) {
	id, err := httpapi.PathID(r, "savings account id")
	if err != nil {
		return nil, err
	}
	snapshots, err := h.service.ListSnapshots(id)
	if err != nil {
		return nil, err
	}
	out := make([]apiSnapshot, 0, len(snapshots))
	for _, s := range snapshots {
		out = append(out, apiSnapshot{
			ID: s.ID, Month: s.Month, ValueCents: s.ValueCents,
			Description: s.Description, CreatedAt: s.CreatedAt,
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

func toAPIAccount(a savings.AccountSummary) apiAccount {
	var target *int64
	if a.TargetCents > 0 {
		t := int64(a.TargetCents)
		target = &t
	}
	return apiAccount{
		ID: a.ID, Name: a.Name, Owner: nullable(a.Username), TargetCents: target, Notes: a.Notes,
		BalanceCents: a.BalanceCents, DepositedCents: a.DepositedCents, WithdrawnCents: a.WithdrawnCents,
		InterestCents: a.InterestCents, ProgressPct: a.ProgressPct,
		MovementCount: a.MovementCount, SnapshotCount: a.SnapshotCount,
		LastSnapshotMonth: nullable(a.LastSnapshotMonth),
	}
}

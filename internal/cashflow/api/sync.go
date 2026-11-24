package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/auth"
	"github.com/tiagorocha94/household-webapp/internal/cashflow/service"
	"github.com/tiagorocha94/household-webapp/internal/details"
	"github.com/tiagorocha94/household-webapp/internal/money"
)

// Statuses of one operation in a sync answer.
const (
	syncOK       = "ok"
	syncRejected = "rejected"
)

type apiSyncRequest struct {
	// Operations is a pointer so that a body without the key can be told from
	// an empty batch.
	Operations *[]apiSyncOperation `json:"operations"`
}

type apiSyncOperation struct {
	Op          string      `json:"op"`
	SyncID      string      `json:"syncId"`
	Category    string      `json:"category"`
	Date        string      `json:"date"`
	AmountCents money.Cents `json:"amountCents"`
	Description string      `json:"description"`
	// Detail is a pointer so that a movement without the key leaves its detail
	// alone, and an empty object removes it.
	Detail *apiSyncDetail `json:"detail"`
}

type apiSyncDetail struct {
	Savings     *apiSyncSavings     `json:"savings"`
	Investments *apiSyncInvestments `json:"investments"`
	Credit      *apiSyncCredit      `json:"credit"`
}

type apiSyncCredit struct {
	CreditID       int64       `json:"creditId"`
	PrincipalCents money.Cents `json:"principalCents"`
	InterestCents  money.Cents `json:"interestCents"`
	Extraordinary  bool        `json:"extraordinary"`
}

type apiSyncInvestments struct {
	EntityID int64                   `json:"entityId"`
	Lines    []apiSyncInvestmentLine `json:"lines"`
}

type apiSyncInvestmentLine struct {
	InstrumentID int64        `json:"instrumentId"`
	AmountCents  money.Cents  `json:"amountCents"`
	ValueCents   *money.Cents `json:"valueCents"`
}

type apiSyncSavings struct {
	AccountID  int64        `json:"accountId"`
	ValueCents *money.Cents `json:"valueCents"`
}

type apiSyncResponse struct {
	Results []apiSyncResult `json:"results"`
}

type apiSyncResult struct {
	SyncID string `json:"syncId"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// toDetail turns the wire detail into the one the service takes; a missing
// detail stays missing.
func (d *apiSyncDetail) toDetail() *details.Detail {
	if d == nil {
		return nil
	}
	out := &details.Detail{}
	if d.Savings != nil {
		out.Savings = &details.Savings{AccountID: d.Savings.AccountID, ValueCents: d.Savings.ValueCents}
	}
	if d.Investments != nil {
		inv := &details.Investments{EntityID: d.Investments.EntityID}
		for _, l := range d.Investments.Lines {
			inv.Lines = append(inv.Lines, details.InvestmentLine{
				InstrumentID: l.InstrumentID, AmountCents: l.AmountCents, ValueCents: l.ValueCents,
			})
		}
		out.Investments = inv
	}
	if d.Credit != nil {
		out.Credit = &details.Credit{
			CreditID: d.Credit.CreditID, PrincipalCents: d.Credit.PrincipalCents,
			InterestCents: d.Credit.InterestCents, Extraordinary: d.Credit.Extraordinary,
		}
	}
	return out
}

// Sync applies a batch of movement operations for the person whose device made
// the request. The person comes from the device token (it must run behind
// RequireDevice), never from the body. The answer has one result per
// operation, in the same order. A body that is not the expected JSON, or that
// holds more operations than allowed, is a bad request and nothing is applied.
func (h *Handler) Sync(r *http.Request) (any, error) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		return nil, fmt.Errorf("sync without an authenticated device")
	}

	var req apiSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, fmt.Errorf("decode sync request: %w: %w", apperr.ErrBadRequest, err)
	}
	if req.Operations == nil {
		return nil, fmt.Errorf("sync request has no operations: %w", apperr.ErrBadRequest)
	}

	ops := make([]service.SyncOperation, 0, len(*req.Operations))
	for _, o := range *req.Operations {
		ops = append(ops, service.SyncOperation{
			Op: o.Op, SyncID: o.SyncID, Category: o.Category, Date: o.Date,
			AmountCents: o.AmountCents, Description: o.Description,
			Detail: o.Detail.toDetail(),
		})
	}
	results, err := h.service.ApplySync(user.ID, ops)
	if err != nil {
		return nil, err
	}

	out := apiSyncResponse{Results: make([]apiSyncResult, 0, len(results))}
	for _, res := range results {
		item := apiSyncResult{SyncID: res.SyncID, Status: syncOK}
		if res.Rejected != "" {
			item.Status, item.Reason = syncRejected, res.Rejected
		}
		out.Results = append(out.Results, item)
	}
	return out, nil
}

// Package api serves the read-only overview JSON API. It calls the overview
// service and maps its result to the wire format; HTTP status mapping is done
// by the server adapter.
package api

import (
	"net/http"

	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/overview"
)

// Handler serves the overview.
type Handler struct {
	service *overview.Service
	history *overview.History
}

// apiOverview leaves out the section of a module that does not run.
type apiOverview struct {
	Month       string           `json:"month"`
	Attention   []apiAttention   `json:"attention"`
	Goals       *apiGoals        `json:"goals,omitempty"`
	Credit      *apiCredit       `json:"credit,omitempty"`
	Investments *apiInvestments  `json:"investments,omitempty"`
	Savings     *apiSavings      `json:"savings,omitempty"`
	NetWorth    apiNetWorth      `json:"netWorth"`
	Cashflow    *apiCashflow     `json:"cashflow,omitempty"`
	ActiveGoals []apiGoalRow     `json:"activeGoals"`
	Contracts   []apiContractRow `json:"contracts"`
	Holdings    []apiHoldingRow  `json:"holdings"`
}

type apiAttention struct {
	Kind        string      `json:"kind"`
	Name        string      `json:"name"`
	Month       string      `json:"month"`
	AmountCents money.Cents `json:"amountCents"`
	Pct         int         `json:"pct"`
}

type apiGoals struct {
	SavedCents  money.Cents `json:"savedCents"`
	TargetCents money.Cents `json:"targetCents"`
}

type apiCredit struct {
	OutstandingCents       money.Cents `json:"outstandingCents"`
	MonthlyInstalmentCents money.Cents `json:"monthlyInstalmentCents"`
	ActiveContracts        int         `json:"activeContracts"`
}

type apiInvestments struct {
	CurrentValueCents money.Cents `json:"currentValueCents"`
}

type apiNetWorth struct {
	AssetsCents money.Cents   `json:"assetsCents"`
	CashCents   money.Cents   `json:"cashCents"`
	CashByUser  []apiUserCash `json:"cashByUser"`
	CreditCents money.Cents   `json:"creditCents"`
	NetCents    money.Cents   `json:"netCents"`
}

type apiNetWorthMonth struct {
	Month            string      `json:"month"`
	SavingsCents     money.Cents `json:"savingsCents"`
	InvestmentsCents money.Cents `json:"investmentsCents"`
	GoalsCents       money.Cents `json:"goalsCents"`
	CashCents        money.Cents `json:"cashCents"`
	AssetsCents      money.Cents `json:"assetsCents"`
	CreditCents      money.Cents `json:"creditCents"`
	NetCents         money.Cents `json:"netCents"`
}

type apiUserCash struct {
	User         string      `json:"user"`
	BalanceCents money.Cents `json:"balanceCents"`
}

type apiSavings struct {
	BalanceCents  money.Cents     `json:"balanceCents"`
	TargetCents   money.Cents     `json:"targetCents"`
	Accounts      int             `json:"accounts"`
	InterestCents money.Cents     `json:"interestCents"`
	Items         []apiSavingsRow `json:"items"`
}

type apiSavingsRow struct {
	ID            int64        `json:"id"`
	Name          string       `json:"name"`
	Owner         *string      `json:"owner"`
	BalanceCents  money.Cents  `json:"balanceCents"`
	TargetCents   *money.Cents `json:"targetCents"`
	ProgressPct   int          `json:"progressPct"`
	InterestCents money.Cents  `json:"interestCents"`
}

type apiCashflow struct {
	Month                string        `json:"month"`
	IncomeCents          money.Cents   `json:"incomeCents"`
	ExpenseCents         money.Cents   `json:"expenseCents"`
	SavedCents           money.Cents   `json:"savedCents"`
	NetCents             money.Cents   `json:"netCents"`
	SavingsRatePct       int           `json:"savingsRatePct"`
	PreviousExpenseCents money.Cents   `json:"previousExpenseCents"`
	Groups               []apiGroupRow `json:"groups"`
}

type apiGroupRow struct {
	Name        string      `json:"name"`
	BudgetCents money.Cents `json:"budgetCents"`
	SpentCents  money.Cents `json:"spentCents"`
	UsedPct     int         `json:"usedPct"`
}

type apiGoalRow struct {
	ID          int64       `json:"id"`
	Name        string      `json:"name"`
	EndMonth    string      `json:"endMonth"`
	TargetCents money.Cents `json:"targetCents"`
	SavedCents  money.Cents `json:"savedCents"`
	ProgressPct int         `json:"progressPct"`
	Behind      bool        `json:"behind"`
}

type apiContractRow struct {
	ID                  int64       `json:"id"`
	Name                string      `json:"name"`
	OutstandingCents    money.Cents `json:"outstandingCents"`
	InstalmentCents     money.Cents `json:"instalmentCents"`
	NextInstalmentMonth *string     `json:"nextInstalmentMonth"`
	EndMonth            *string     `json:"endMonth"`
}

type apiHoldingRow struct {
	ID         int64       `json:"id"`
	Name       string      `json:"name"`
	ValueCents money.Cents `json:"valueCents"`
}

// New creates an overview Handler.
func New(service *overview.Service, history *overview.History) *Handler {
	return &Handler{service: service, history: history}
}

// GetNetWorth returns net worth month by month, oldest first, for every month
// that has cashflow.
func (h *Handler) GetNetWorth(_ *http.Request) (any, error) {
	months, err := h.history.NetWorth()
	if err != nil {
		return nil, err
	}
	out := make([]apiNetWorthMonth, 0, len(months))
	for _, m := range months {
		out = append(out, apiNetWorthMonth(m))
	}
	return out, nil
}

// Get returns the overview of every module.
func (h *Handler) Get(_ *http.Request) (any, error) {
	o, err := h.service.Get()
	if err != nil {
		return nil, err
	}
	return toAPI(*o), nil
}

func toAPINetWorth(n overview.NetWorthFigures) apiNetWorth {
	cash := make([]apiUserCash, 0, len(n.CashByUser))
	for _, c := range n.CashByUser {
		cash = append(cash, apiUserCash(c))
	}
	return apiNetWorth{
		AssetsCents: n.AssetsCents,
		CashCents:   n.CashCents,
		CashByUser:  cash,
		CreditCents: n.CreditCents,
		NetCents:    n.NetCents,
	}
}

func toAPISavings(s overview.SavingsFigures) apiSavings {
	out := apiSavings{
		BalanceCents: s.BalanceCents, TargetCents: s.TargetCents, Accounts: s.Accounts,
		InterestCents: s.InterestCents,
		Items:         make([]apiSavingsRow, 0, len(s.Items)),
	}
	for _, it := range s.Items {
		row := apiSavingsRow{
			ID: it.ID, Name: it.Name, BalanceCents: it.BalanceCents,
			ProgressPct: it.ProgressPct, InterestCents: it.InterestCents,
		}
		if it.TargetCents > 0 {
			target := it.TargetCents
			row.TargetCents = &target
		}
		if it.Owner != "" {
			owner := it.Owner
			row.Owner = &owner
		}
		out.Items = append(out.Items, row)
	}
	return out
}

func toAPI(o overview.Overview) apiOverview {
	out := apiOverview{
		Month:       o.Month,
		Attention:   make([]apiAttention, 0, len(o.Attention)),
		NetWorth:    toAPINetWorth(o.NetWorth),
		ActiveGoals: make([]apiGoalRow, 0, len(o.ActiveGoals)),
		Contracts:   make([]apiContractRow, 0, len(o.Contracts)),
		Holdings:    make([]apiHoldingRow, 0, len(o.Holdings)),
	}
	if g := o.Goals; g != nil {
		out.Goals = &apiGoals{SavedCents: g.SavedCents, TargetCents: g.TargetCents}
	}
	if c := o.Credit; c != nil {
		credit := apiCredit(*c)
		out.Credit = &credit
	}
	if i := o.Investments; i != nil {
		investments := apiInvestments(*i)
		out.Investments = &investments
	}
	if sv := o.Savings; sv != nil {
		savings := toAPISavings(*sv)
		out.Savings = &savings
	}
	if cf := o.Cashflow; cf != nil {
		out.Cashflow = &apiCashflow{
			Month: cf.Month, IncomeCents: cf.IncomeCents,
			ExpenseCents: cf.ExpenseCents, SavedCents: cf.SavedCents,
			NetCents:             cf.NetCents,
			SavingsRatePct:       cf.SavingsRatePct,
			PreviousExpenseCents: cf.PreviousExpenseCents,
			Groups:               make([]apiGroupRow, 0, len(cf.Groups)),
		}
		for _, g := range cf.Groups {
			out.Cashflow.Groups = append(out.Cashflow.Groups, apiGroupRow(g))
		}
	}
	for _, a := range o.Attention {
		out.Attention = append(out.Attention, apiAttention(a))
	}
	for _, g := range o.ActiveGoals {
		out.ActiveGoals = append(out.ActiveGoals, apiGoalRow(g))
	}
	for _, c := range o.Contracts {
		out.Contracts = append(out.Contracts, apiContractRow{
			ID: c.ID, Name: c.Name, OutstandingCents: c.OutstandingCents,
			InstalmentCents:     c.InstalmentCents,
			NextInstalmentMonth: nullable(c.NextInstalmentMonth),
			EndMonth:            nullable(c.EndMonth),
		})
	}
	for _, h := range o.Holdings {
		out.Holdings = append(out.Holdings, apiHoldingRow(h))
	}
	return out
}

// nullable maps an empty string to JSON null.
func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

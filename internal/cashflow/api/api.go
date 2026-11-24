package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/auth"
	"github.com/tiagorocha94/household-webapp/internal/cashflow"
	"github.com/tiagorocha94/household-webapp/internal/cashflow/service"
	"github.com/tiagorocha94/household-webapp/internal/money"
)

// Handler serves the read-only cashflow JSON API. It parses requests and calls
// the service layer; HTTP status mapping is done by the server adapter.
type Handler struct {
	service *service.Service
}

type apiCategory struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Direction string `json:"direction"`
	Icon      string `json:"icon"`
	Color     string `json:"color"`
	Role      string `json:"role"`
}

// apiOwnBalance is the caller's own cash at the end of a month. BalanceCents is
// null when no balance was ever typed for the person.
type apiOwnBalance struct {
	Month        string       `json:"month"`
	BalanceCents *money.Cents `json:"balanceCents"`
}

// savingsRatePct is a percentage scaled by 100: 2500 means 25.00%.
type apiMonth struct {
	Month          string      `json:"month"`
	IncomeCents    money.Cents `json:"incomeCents"`
	ExpenseCents   money.Cents `json:"expenseCents"`
	NetCents       money.Cents `json:"netCents"`
	SavingsRatePct int         `json:"savingsRatePct"`
	MovementCount  int         `json:"movementCount"`
}

// apiMonthDetail embeds apiMonth so its fields are flattened into the object.
type apiMonthDetail struct {
	apiMonth
	ByCategory []apiCategoryTotal `json:"byCategory"`
	SavedCents money.Cents        `json:"savedCents"`
	ByUser     []apiUserTotal     `json:"byUser"`
	// PeopleHistory is the six months ending at the month, oldest first.
	PeopleHistory []apiUserMonth  `json:"peopleHistory"`
	ByGroup       []apiGroupTotal `json:"byGroup"`
}

// apiBudget is what a phone reads: one month's totals, the budget groups and
// the months that have movements. It carries no people and no movements.
type apiBudget struct {
	Month        string          `json:"month"`
	IncomeCents  money.Cents     `json:"incomeCents"`
	ExpenseCents money.Cents     `json:"expenseCents"`
	NetCents     money.Cents     `json:"netCents"`
	ByGroup      []apiGroupTotal `json:"byGroup"`
	Months       []string        `json:"months"`
}

type apiGroupTotal struct {
	Name        string             `json:"name"`
	Percent     int                `json:"percent"`
	BudgetCents money.Cents        `json:"budgetCents"`
	SpentCents  money.Cents        `json:"spentCents"`
	LeftCents   money.Cents        `json:"leftCents"`
	Categories  []apiCategoryTotal `json:"categories"`
}

type apiCategoryTotal struct {
	CategoryID    int64       `json:"categoryId"`
	Name          string      `json:"name"`
	Direction     string      `json:"direction"`
	TotalCents    money.Cents `json:"totalCents"`
	MovementCount int         `json:"movementCount"`
	Role          string      `json:"role"`
}

type apiUserTotal struct {
	Username     string      `json:"username"`
	IncomeCents  money.Cents `json:"incomeCents"`
	ExpenseCents money.Cents `json:"expenseCents"`
	NetCents     money.Cents `json:"netCents"`
	// SpentCents is ExpenseCents without PutAsideCents.
	SpentCents    money.Cents  `json:"spentCents"`
	PutAsideCents money.Cents  `json:"putAsideCents"`
	CashCents     *money.Cents `json:"cashCents"`
}

type apiUserMonth struct {
	Month         string      `json:"month"`
	Username      string      `json:"username"`
	IncomeCents   money.Cents `json:"incomeCents"`
	PutAsideCents money.Cents `json:"putAsideCents"`
}

type apiMovement struct {
	ID          int64       `json:"id"`
	Date        string      `json:"date"`
	Username    string      `json:"username"`
	Category    string      `json:"category"`
	Direction   string      `json:"direction"`
	AmountCents money.Cents `json:"amountCents"`
	Description string      `json:"description"`
	CreatedAt   time.Time   `json:"createdAt"`
}

// New creates a cashflow Handler.
func New(service *service.Service) *Handler {
	return &Handler{service: service}
}

// ListCategories returns the fixed set of categories.
func (h *Handler) ListCategories(_ *http.Request) (any, error) {
	categories, err := h.service.ListCategories()
	if err != nil {
		return nil, err
	}
	out := make([]apiCategory, 0, len(categories))
	for _, c := range categories {
		out = append(out, apiCategory{ID: c.ID, Name: c.Name, Direction: c.Direction, Icon: c.Icon, Color: c.Color, Role: c.Role})
	}
	return out, nil
}

// ListMonths returns one summary per month that has movements, newest first.
func (h *Handler) ListMonths(_ *http.Request) (any, error) {
	months, err := h.service.ListMonths()
	if err != nil {
		return nil, err
	}
	out := make([]apiMonth, 0, len(months))
	for _, m := range months {
		out = append(out, toAPIMonth(m))
	}
	return out, nil
}

// GetMonth returns one month with its totals by category and by person. The
// month comes from the path; a malformed one is a bad request and a month
// without movements is not found.
func (h *Handler) GetMonth(r *http.Request) (any, error) {
	detail, err := h.service.GetMonth(r.PathValue("month"))
	if err != nil {
		return nil, err
	}
	out := apiMonthDetail{
		apiMonth:      toAPIMonth(detail.MonthSummary),
		SavedCents:    detail.SavedCents,
		ByCategory:    make([]apiCategoryTotal, 0, len(detail.ByCategory)),
		ByUser:        make([]apiUserTotal, 0, len(detail.ByUser)),
		PeopleHistory: make([]apiUserMonth, 0, len(detail.PeopleHistory)),
		ByGroup:       make([]apiGroupTotal, 0, len(detail.ByGroup)),
	}
	for _, c := range detail.ByCategory {
		out.ByCategory = append(out.ByCategory, toAPICategoryTotal(c))
	}
	for _, g := range detail.ByGroup {
		out.ByGroup = append(out.ByGroup, toAPIGroupTotal(g))
	}
	for _, u := range detail.ByUser {
		out.ByUser = append(out.ByUser, apiUserTotal{
			Username: u.Username, IncomeCents: u.IncomeCents,
			ExpenseCents: u.ExpenseCents, NetCents: u.NetCents,
			SpentCents: u.ExpenseCents - u.PutAsideCents, PutAsideCents: u.PutAsideCents,
			CashCents: u.CashCents,
		})
	}
	for _, h := range detail.PeopleHistory {
		out.PeopleHistory = append(out.PeopleHistory, apiUserMonth(h))
	}
	return out, nil
}

// GetBudget returns one month's budget by group, for the phone. A month
// without movements is empty, not missing; a malformed one is a bad request.
func (h *Handler) GetBudget(r *http.Request) (any, error) {
	budget, err := h.service.GetBudget(r.PathValue("month"))
	if err != nil {
		return nil, err
	}
	out := apiBudget{
		Month: budget.Month, IncomeCents: budget.IncomeCents, ExpenseCents: budget.ExpenseCents,
		NetCents: budget.NetCents, Months: budget.Months,
		ByGroup: make([]apiGroupTotal, 0, len(budget.ByGroup)),
	}
	for _, g := range budget.ByGroup {
		out.ByGroup = append(out.ByGroup, toAPIGroupTotal(g))
	}
	return out, nil
}

// GetOwnBalance returns the cash of the person who made the request (session
// user or device owner), never anyone else's.
func (h *Handler) GetOwnBalance(r *http.Request) (any, error) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		return nil, fmt.Errorf("own balance without an authenticated user")
	}
	m := r.PathValue("month")
	cents, err := h.service.OwnCash(user.Username, m)
	if err != nil {
		return nil, err
	}
	return apiOwnBalance{Month: m, BalanceCents: cents}, nil
}

// ListMonthMovements returns every movement of a month, newest date first. A
// month without movements is an empty list.
func (h *Handler) ListMonthMovements(r *http.Request) (any, error) {
	movements, err := h.service.ListMonthMovements(r.PathValue("month"))
	if err != nil {
		return nil, err
	}
	out := make([]apiMovement, 0, len(movements))
	for _, m := range movements {
		out = append(out, apiMovement{
			ID: m.ID, Date: m.Date, Username: m.Username, Category: m.Category,
			Direction: m.Direction, AmountCents: m.AmountCents,
			Description: m.Description, CreatedAt: m.CreatedAt,
		})
	}
	return out, nil
}

func toAPIGroupTotal(g cashflow.GroupTotal) apiGroupTotal {
	group := apiGroupTotal{
		Name: g.Name, Percent: g.SharePct, BudgetCents: g.BudgetCents,
		SpentCents: g.SpentCents, LeftCents: g.LeftCents,
		Categories: make([]apiCategoryTotal, 0, len(g.Categories)),
	}
	for _, c := range g.Categories {
		group.Categories = append(group.Categories, toAPICategoryTotal(c))
	}
	return group
}

func toAPICategoryTotal(c cashflow.CategoryTotal) apiCategoryTotal {
	return apiCategoryTotal{
		CategoryID: c.CategoryID, Name: c.Name, Direction: c.Direction,
		TotalCents: c.TotalCents, MovementCount: c.MovementCount, Role: c.Role,
	}
}

func toAPIMonth(m cashflow.MonthSummary) apiMonth {
	return apiMonth{
		Month: m.Month, IncomeCents: m.IncomeCents, ExpenseCents: m.ExpenseCents,
		NetCents: m.NetCents, SavingsRatePct: m.SavingsRatePct, MovementCount: m.MovementCount,
	}
}

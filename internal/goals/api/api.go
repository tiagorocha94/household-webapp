package api

import (
	"net/http"
	"time"

	httpapi "github.com/tiagorocha94/household-webapp/internal/api"
	"github.com/tiagorocha94/household-webapp/internal/goals"
	"github.com/tiagorocha94/household-webapp/internal/goals/service"
	"github.com/tiagorocha94/household-webapp/internal/money"
)

// Handler serves the read-only goals JSON API. It parses requests and calls
// the service layer; HTTP status mapping is done by the server adapter.
type Handler struct {
	service *service.Service
}

type apiGoal struct {
	ID                     int64       `json:"id"`
	AccountID              int64       `json:"accountId"`
	Name                   string      `json:"name"`
	TargetCents            money.Cents `json:"targetCents"`
	StartMonth             string      `json:"startMonth"`
	EndMonth               string      `json:"endMonth"`
	Status                 string      `json:"status"`
	TotalMonths            int         `json:"totalMonths"`
	ElapsedMonths          int         `json:"elapsedMonths"`
	MonthlyAllocationCents money.Cents `json:"monthlyAllocationCents"`
	AllocatedCents         money.Cents `json:"allocatedCents"`
	ExpensesCents          money.Cents `json:"expensesCents"`
	BalanceCents           money.Cents `json:"balanceCents"`
	// ClosingBalanceCents is the balance the goal had when it was completed;
	// null while the goal is open.
	ClosingBalanceCents *money.Cents `json:"closingBalanceCents"`
	ProgressPct         int          `json:"progressPct"`
}

type apiProjection struct {
	MonthlyDepositCents money.Cents          `json:"monthlyDepositCents"`
	Months              []apiProjectionMonth `json:"months"`
}

type apiProjectionMonth struct {
	Month            string      `json:"month"`
	DepositsCents    money.Cents `json:"depositsCents"`
	AllocationsCents money.Cents `json:"allocationsCents"`
	ExpensesCents    money.Cents `json:"expensesCents"`
	ReleasedCents    money.Cents `json:"releasedCents"`
	CoveredCents     money.Cents `json:"coveredCents"`
	BalanceCents     money.Cents `json:"balanceCents"`
	UnallocatedCents money.Cents `json:"unallocatedCents"`
	IsProjection     bool        `json:"isProjection"`
}

type apiAccountDetail struct {
	Account apiAccount `json:"account"`
	Balance apiBalance `json:"balance"`
}

type apiBalance struct {
	DepositsCents    money.Cents `json:"depositsCents"`
	AllocatedCents   money.Cents `json:"allocatedCents"`
	ExpensesCents    money.Cents `json:"expensesCents"`
	ReleasedCents    money.Cents `json:"releasedCents"`
	CoveredCents     money.Cents `json:"coveredCents"`
	BalanceCents     money.Cents `json:"balanceCents"`
	UnallocatedCents money.Cents `json:"unallocatedCents"`
}

type apiAccount struct {
	ID                  int64       `json:"id"`
	Name                string      `json:"name"`
	MonthlyDepositCents money.Cents `json:"monthlyDepositCents"`
}

type apiGoalDetail struct {
	Goal      apiGoal       `json:"goal"`
	Movements []apiMovement `json:"movements"`
}

type apiMovement struct {
	ID            int64       `json:"id"`
	AccountID     int64       `json:"accountId"`
	GoalID        *int64      `json:"goalId"`
	GoalName      string      `json:"goalName"`
	Type          string      `json:"type"`
	AmountCents   money.Cents `json:"amountCents"`
	Month         string      `json:"month"`
	Description   string      `json:"description"`
	CreatedAt     time.Time   `json:"createdAt"`
	AccountInflow bool        `json:"accountInflow"`
	GoalInflow    bool        `json:"goalInflow"`
}

// New creates a new Handler backed by the given Service.
func New(svc *service.Service) *Handler {
	return &Handler{service: svc}
}

// The api* types are the JSON shapes of the read API. They are separate from
// the domain structs so the wire format can evolve independently. Money is
// integer cents and every money key ends in "Cents".

// GetAccount returns one account with its balance summary.
func (h *Handler) GetAccount(r *http.Request) (any, error) {
	id, err := httpapi.PathID(r, "account id")
	if err != nil {
		return nil, err
	}
	account, err := h.service.GetAccount(id)
	if err != nil {
		return nil, err
	}
	b, err := h.service.GetBalance(id)
	if err != nil {
		return nil, err
	}
	return apiAccountDetail{
		Account: toAPIAccount(*account),
		Balance: apiBalance{
			DepositsCents: b.TotalDepositsCents, AllocatedCents: b.TotalAllocatedCents, ExpensesCents: b.TotalExpensesCents,
			ReleasedCents: b.TotalReleasedCents, CoveredCents: b.TotalCoveredCents, BalanceCents: b.BalanceCents, UnallocatedCents: b.UnallocatedCents,
		},
	}, nil
}

// ListGoals returns every goal of an account with computed totals.
func (h *Handler) ListGoals(r *http.Request) (any, error) {
	id, err := httpapi.PathID(r, "account id")
	if err != nil {
		return nil, err
	}
	summaries, err := h.service.ListGoalSummaries(id)
	if err != nil {
		return nil, err
	}
	out := make([]apiGoal, 0, len(summaries))
	for _, g := range summaries {
		out = append(out, toAPIGoal(g))
	}
	return out, nil
}

// ListAccounts returns every account.
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

// ListMovements returns the full account ledger, newest month first.
func (h *Handler) ListMovements(r *http.Request) (any, error) {
	id, err := httpapi.PathID(r, "account id")
	if err != nil {
		return nil, err
	}
	movements, err := h.service.ListMovements(id)
	if err != nil {
		return nil, err
	}
	return toAPIMovements(movements), nil
}

// GetProjection returns the full projection window for an account.
func (h *Handler) GetProjection(r *http.Request) (any, error) {
	id, err := httpapi.PathID(r, "account id")
	if err != nil {
		return nil, err
	}
	account, err := h.service.GetAccount(id)
	if err != nil {
		return nil, err
	}
	months, err := h.service.GetProjection(id)
	if err != nil {
		return nil, err
	}
	return apiProjection{MonthlyDepositCents: account.MonthlyDepositCents, Months: toAPIProjectionMonths(months)}, nil
}

// GetGoal returns one goal with its movements from the goal's perspective.
func (h *Handler) GetGoal(r *http.Request) (any, error) {
	id, err := httpapi.PathID(r, "goal id")
	if err != nil {
		return nil, err
	}
	goal, err := h.service.GetGoalSummary(id)
	if err != nil {
		return nil, err
	}
	movements, err := h.service.ListGoalMovements(id)
	if err != nil {
		return nil, err
	}
	return apiGoalDetail{Goal: toAPIGoal(*goal), Movements: toAPIMovements(movements)}, nil
}

func toAPIProjectionMonths(months []goals.ProjectionMonth) []apiProjectionMonth {
	out := make([]apiProjectionMonth, 0, len(months))
	for _, m := range months {
		out = append(out, apiProjectionMonth(m)) // same fields; only the tags differ
	}
	return out
}

func toAPIAccount(a goals.Account) apiAccount {
	return apiAccount{ID: a.ID, Name: a.Name, MonthlyDepositCents: a.MonthlyDepositCents}
}

func toAPIGoal(g goals.GoalSummary) apiGoal {
	return apiGoal{
		ID: g.GoalID, AccountID: g.AccountID, Name: g.Name, TargetCents: g.TargetCents,
		StartMonth: g.StartMonth, EndMonth: g.EndMonth, Status: g.Status,
		TotalMonths: g.TotalMonths, ElapsedMonths: g.ElapsedMonths,
		MonthlyAllocationCents: g.MonthlyAllocationCents, AllocatedCents: g.TotalAllocatedCents,
		ExpensesCents: g.TotalExpensesCents,
		BalanceCents:  g.GoalBalanceCents, ClosingBalanceCents: g.ClosingBalanceCents, ProgressPct: g.ProgressPct,
	}
}

func toAPIMovements(movements []goals.Movement) []apiMovement {
	out := make([]apiMovement, 0, len(movements))
	for _, m := range movements {
		out = append(out, apiMovement{
			ID: m.ID, AccountID: m.AccountID, GoalID: m.GoalID, GoalName: m.GoalName,
			Type: m.Type, AmountCents: m.AmountCents, Month: m.Month, Description: m.Description,
			CreatedAt: m.CreatedAt, AccountInflow: m.IsInflow(), GoalInflow: m.IsInflowForGoal(),
		})
	}
	return out
}

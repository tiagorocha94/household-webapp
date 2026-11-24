package api

import (
	"net/http"
	"time"

	httpapi "github.com/tiagorocha94/household-webapp/internal/api"
	"github.com/tiagorocha94/household-webapp/internal/investments"
	"github.com/tiagorocha94/household-webapp/internal/investments/service"
	"github.com/tiagorocha94/household-webapp/internal/money"
)

// Handler serves the read-only investments JSON API. It parses requests and
// calls the service layer; HTTP status mapping is done by the server adapter.
type Handler struct {
	service *service.Service
}

// gainPct fields are percentages scaled by 100: 734 means +7.34%.
type apiEntity struct {
	ID                int64       `json:"id"`
	Name              string      `json:"name"`
	Kind              string      `json:"kind"`
	User              *string     `json:"user"`
	InstrumentCount   int         `json:"instrumentCount"`
	AllocatedCents    money.Cents `json:"allocatedCents"`
	CurrentValueCents money.Cents `json:"currentValueCents"`
	GainCents         money.Cents `json:"gainCents"`
	GainPct           int         `json:"gainPct"`
}

type apiInstrument struct {
	ID                int64       `json:"id"`
	EntityID          int64       `json:"entityId"`
	EntityName        string      `json:"entityName"`
	Name              string      `json:"name"`
	Type              string      `json:"type"`
	Notes             string      `json:"notes"`
	User              *string     `json:"user"`
	ContributionCount int         `json:"contributionCount"`
	SnapshotCount     int         `json:"snapshotCount"`
	AllocatedCents    money.Cents `json:"allocatedCents"`
	CurrentValueCents money.Cents `json:"currentValueCents"`
	GainCents         money.Cents `json:"gainCents"`
	GainPct           int         `json:"gainPct"`
	LastSnapshotMonth *string     `json:"lastSnapshotMonth"`
}

type apiContribution struct {
	ID          int64       `json:"id"`
	Month       string      `json:"month"`
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

// New creates an investments Handler.
func New(service *service.Service) *Handler {
	return &Handler{service: service}
}

// ListEntities returns every entity with its totals.
func (h *Handler) ListEntities(_ *http.Request) (any, error) {
	entities, err := h.service.ListEntities()
	if err != nil {
		return nil, err
	}
	out := make([]apiEntity, 0, len(entities))
	for _, e := range entities {
		out = append(out, toAPIEntity(e))
	}
	return out, nil
}

// GetEntity returns one entity with its totals.
func (h *Handler) GetEntity(r *http.Request) (any, error) {
	id, err := httpapi.PathID(r, "investment entity id")
	if err != nil {
		return nil, err
	}
	entity, err := h.service.GetEntitySummary(id)
	if err != nil {
		return nil, err
	}
	return toAPIEntity(*entity), nil
}

// ListEntityInstruments returns the instruments of one entity.
func (h *Handler) ListEntityInstruments(r *http.Request) (any, error) {
	id, err := httpapi.PathID(r, "investment entity id")
	if err != nil {
		return nil, err
	}
	instruments, err := h.service.ListInstrumentsByEntity(id)
	if err != nil {
		return nil, err
	}
	out := make([]apiInstrument, 0, len(instruments))
	for _, i := range instruments {
		out = append(out, toAPIInstrument(i))
	}
	return out, nil
}

// GetInstrument returns one instrument with its totals.
func (h *Handler) GetInstrument(r *http.Request) (any, error) {
	id, err := httpapi.PathID(r, "investment instrument id")
	if err != nil {
		return nil, err
	}
	instrument, err := h.service.GetInstrumentSummary(id)
	if err != nil {
		return nil, err
	}
	return toAPIInstrument(*instrument), nil
}

// ListContributions returns an instrument's contribution history, newest first.
func (h *Handler) ListContributions(r *http.Request) (any, error) {
	id, err := httpapi.PathID(r, "investment instrument id")
	if err != nil {
		return nil, err
	}
	contributions, err := h.service.ListContributions(id)
	if err != nil {
		return nil, err
	}
	out := make([]apiContribution, 0, len(contributions))
	for _, c := range contributions {
		out = append(out, apiContribution{
			ID: c.ID, Month: c.Month, AmountCents: c.AmountCents,
			Description: c.Description, CreatedAt: c.CreatedAt,
		})
	}
	return out, nil
}

// ListSnapshots returns an instrument's value history, newest first.
func (h *Handler) ListSnapshots(r *http.Request) (any, error) {
	id, err := httpapi.PathID(r, "investment instrument id")
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

func toAPIEntity(e investments.EntitySummary) apiEntity {
	return apiEntity{
		ID: e.ID, Name: e.Name, Kind: e.Kind, User: nullable(e.Username), InstrumentCount: e.InstrumentCount,
		AllocatedCents: e.AllocatedCents, CurrentValueCents: e.CurrentValueCents,
		GainCents: e.GainCents, GainPct: e.GainPct,
	}
}

func toAPIInstrument(i investments.InstrumentSummary) apiInstrument {
	return apiInstrument{
		ID: i.ID, EntityID: i.EntityID, EntityName: i.EntityName, Name: i.Name,
		Type: i.Type, Notes: i.Notes, User: nullable(i.Username), ContributionCount: i.ContributionCount,
		SnapshotCount: i.SnapshotCount, AllocatedCents: i.AllocatedCents,
		CurrentValueCents: i.CurrentValueCents, GainCents: i.GainCents,
		GainPct: i.GainPct, LastSnapshotMonth: nullable(i.LastSnapshotMonth),
	}
}

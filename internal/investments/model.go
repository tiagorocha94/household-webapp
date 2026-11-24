// Package investments exposes a personal portfolio of entities, instruments,
// contributions and value snapshots through the read-only JSON API. There is
// no market sync: every value is typed into the CSV files.
package investments

import (
	"time"

	"github.com/tiagorocha94/household-webapp/internal/money"
)

// Entity kinds.
const (
	KindBank           = "bank"
	KindBroker         = "broker"
	KindInvestmentFirm = "investment_firm"
	KindState          = "state"
	KindOther          = "other"
)

// Instrument types.
const (
	TypeETF                = "etf"
	TypeFund               = "fund"
	TypePPR                = "ppr"
	TypeCertificadosAforro = "certificados_aforro"
	TypeOther              = "other"
)

// Entity is a provider or location of money, such as a bank or a broker.
type Entity struct {
	ID   int64
	Name string
	Kind string
	// Username is the person the account belongs to, or empty when it belongs
	// to the household.
	Username  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// EntitySummary contains an entity and figures derived from its instruments.
type EntitySummary struct {
	Entity
	InstrumentCount   int
	AllocatedCents    money.Cents
	CurrentValueCents money.Cents
	GainCents         money.Cents
	// GainPct is the gain as a percentage scaled by 100: 734 means +7.34%.
	// It can be negative and is 0 when nothing is allocated.
	GainPct int
}

// Instrument is a single holding at an entity.
type Instrument struct {
	ID       int64
	EntityID int64
	Name     string
	Type     string
	Notes    string
	// Username is the person its account belongs to, or empty when that
	// belongs to the household.
	Username  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// InstrumentSummary contains an instrument and figures derived from its
// contributions and snapshots.
type InstrumentSummary struct {
	Instrument
	EntityName        string
	ContributionCount int
	SnapshotCount     int
	AllocatedCents    money.Cents
	CurrentValueCents money.Cents
	GainCents         money.Cents
	// GainPct is the gain as a percentage scaled by 100: 734 means +7.34%.
	// It can be negative and is 0 when nothing is allocated.
	GainPct           int
	LastSnapshotMonth string
}

// Contribution is money added to an instrument in a month.
type Contribution struct {
	ID           int64
	InstrumentID int64
	Month        string
	AmountCents  money.Cents
	Description  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Snapshot is the observed value of an instrument in a month.
type Snapshot struct {
	ID           int64
	InstrumentID int64
	Month        string
	ValueCents   money.Cents
	Description  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

package goals

import (
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/month"
)

// Schedule is the cent-exact plan for allocating a goal's target over its
// months. Every month takes its share of money.Split, except the last, which
// also absorbs drift: the difference between what the schedule says the earlier
// months should have received and what was actually recorded for them. That
// keeps a goal's allocations adding up to its target after rounding differences
// or hand-edited rows. Sync and the projection both allocate through it.
type Schedule struct {
	amountsCents []money.Cents
}

// NewSchedule splits target over the months from startMonth to endMonth
// inclusive. It returns an error when either month is not a valid YYYY-MM.
func NewSchedule(startMonth, endMonth string, target money.Cents) (Schedule, error) {
	months, err := month.Count(startMonth, endMonth)
	if err != nil {
		return Schedule{}, fmt.Errorf("count schedule months: %w", err)
	}
	return Schedule{amountsCents: money.Split(target, months)}, nil
}

// Months returns the number of months in the schedule.
func (s Schedule) Months() int {
	return len(s.amountsCents)
}

// Amount returns the amount to allocate in the i-th month of the schedule,
// counting from zero. The last month also takes drift, which may be negative.
// It returns zero when i is outside the schedule.
func (s Schedule) Amount(i int, drift money.Cents) money.Cents {
	switch {
	case i < 0 || i >= len(s.amountsCents):
		return 0
	case i == len(s.amountsCents)-1:
		return s.amountsCents[i] + drift
	default:
		return s.amountsCents[i]
	}
}

// Drift returns what the first recordedMonths months were scheduled to receive
// minus the recorded cents actually allocated for them, so a positive drift
// means the goal is behind its schedule.
func (s Schedule) Drift(recordedMonths int, recorded money.Cents) money.Cents {
	var scheduled money.Cents
	for _, a := range s.amountsCents[:min(max(recordedMonths, 0), len(s.amountsCents))] {
		scheduled += a
	}
	return scheduled - recorded
}

// FinalAmount returns the amount for the last month when allocatedElsewhere
// cents are already allocated in all the other months: the target minus them.
// It returns zero for an empty schedule.
func (s Schedule) FinalAmount(allocatedElsewhere money.Cents) money.Cents {
	last := len(s.amountsCents) - 1
	return s.Amount(last, s.Drift(last, allocatedElsewhere))
}

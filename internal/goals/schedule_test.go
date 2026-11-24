package goals

import (
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/money"
)

func mustSchedule(t *testing.T, start, end string, target money.Cents) Schedule {
	t.Helper()
	s, err := NewSchedule(start, end, target)
	if err != nil {
		t.Fatalf("NewSchedule(%q, %q, %d) error = %v", start, end, target, err)
	}
	return s
}

func TestScheduleAmountsAddUpToTheTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		start, end string
		target     money.Cents
		months     int
	}{
		{"2026-01", "2026-03", 10000, 3},
		{"2025-11", "2027-06", 420000, 20},
		{"2026-09", "2031-06", 1250000, 58},
		{"2026-02", "2027-06", 130000, 17},
		{"2026-05", "2026-05", 777, 1},
		{"2026-11", "2027-05", 5, 7},
	}
	for _, tt := range tests {
		s := mustSchedule(t, tt.start, tt.end, tt.target)
		if s.Months() != tt.months {
			t.Errorf("%s..%s: Months() = %d, want %d", tt.start, tt.end, s.Months(), tt.months)
		}
		var sum money.Cents
		for i := range s.Months() {
			sum += s.Amount(i, 0)
		}
		if sum != tt.target {
			t.Errorf("%s..%s: amounts sum to %d, want %d", tt.start, tt.end, sum, tt.target)
		}
	}
}

func TestScheduleSpreadsRemainderCentsOverTheFirstMonths(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t, "2026-01", "2026-03", 10000)
	for i, want := range []money.Cents{3334, 3333, 3333} {
		if got := s.Amount(i, 0); got != want {
			t.Errorf("Amount(%d) = %d, want %d", i, got, want)
		}
	}
}

func TestScheduleOnlyTheLastMonthTakesDrift(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t, "2026-01", "2026-03", 10000)
	for i, want := range []money.Cents{3334, 3333, 3333 + 7} {
		if got := s.Amount(i, 7); got != want {
			t.Errorf("Amount(%d, 7) = %d, want %d", i, got, want)
		}
	}
	if got := s.Amount(2, -3333); got != 0 {
		t.Errorf("Amount(2, -3333) = %d, want 0", got)
	}
}

func TestScheduleAmountOutsideTheScheduleIsZero(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t, "2026-01", "2026-03", 10000)
	for _, i := range []int{-1, 3, 100} {
		if got := s.Amount(i, 50); got != 0 {
			t.Errorf("Amount(%d) = %d, want 0", i, got)
		}
	}
}

func TestScheduleDrift(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t, "2026-01", "2026-03", 10000) // 3334, 3333, 3333
	tests := []struct {
		name     string
		recorded int
		cents    money.Cents
		want     money.Cents
	}{
		{"nothing recorded", 0, 0, 0},
		{"on schedule", 2, 6667, 0},
		{"behind by a cent", 1, 3333, 1},
		{"ahead", 2, 7000, -333},
		{"more months than the schedule", 5, 9000, 1000},
		{"negative count is treated as zero", -2, 0, 0},
	}
	for _, tt := range tests {
		if got := s.Drift(tt.recorded, tt.cents); got != tt.want {
			t.Errorf("%s: Drift(%d, %d) = %d, want %d", tt.name, tt.recorded, tt.cents, got, tt.want)
		}
	}
}

func TestScheduleFinalAmountIsTheTargetMinusTheOtherMonths(t *testing.T) {
	t.Parallel()

	s := mustSchedule(t, "2026-01", "2026-03", 10000)
	for _, others := range []money.Cents{0, 3333, 6666, 6667, 9999, 10000, 12000} {
		if got, want := s.FinalAmount(others), 10000-others; got != want {
			t.Errorf("FinalAmount(%d) = %d, want %d", others, got, want)
		}
	}

	empty := mustSchedule(t, "2026-05", "2026-04", 10000)
	if empty.Months() != 0 || empty.FinalAmount(100) != 0 {
		t.Errorf("empty schedule: Months() = %d, FinalAmount(100) = %d, want 0 and 0", empty.Months(), empty.FinalAmount(100))
	}
}

func TestNewScheduleRejectsInvalidMonths(t *testing.T) {
	t.Parallel()

	for _, months := range [][2]string{{"2026-13", "2026-12"}, {"2026-01", "soon"}, {"", ""}} {
		if _, err := NewSchedule(months[0], months[1], 100); err == nil {
			t.Errorf("NewSchedule(%q, %q) succeeded, want error", months[0], months[1])
		}
	}
}

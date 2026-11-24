// Package month provides parsing, counting and arithmetic for calendar months
// written as YYYY-MM strings, the form used in the database, the CSV files and
// the API. Canonical values sort chronologically as text.
package month

import (
	"fmt"
	"iter"
	"time"
)

const (
	// layout is the YYYY-MM layout in Go's reference time.
	layout = "2006-01"
	// dayLayout is the YYYY-MM-DD layout in Go's reference time.
	dayLayout = "2006-01-02"
)

// Validate returns an error unless m is a valid zero-padded YYYY-MM month.
func Validate(m string) error {
	if _, err := time.Parse(layout, m); err != nil {
		return fmt.Errorf("expected YYYY-MM, got %q", m)
	}
	return nil
}

// ValidateDay returns an error unless d is a real calendar day written as a
// zero-padded YYYY-MM-DD. It rejects unpadded values such as 2026-1-5 and
// impossible days such as 2026-02-30.
func ValidateDay(d string) error {
	if _, err := time.Parse(dayLayout, d); err != nil {
		return fmt.Errorf("expected a real day as YYYY-MM-DD, got %q", d)
	}
	return nil
}

// Current returns the month that now falls in, in now's own time zone.
func Current(now time.Time) string {
	return now.Format(layout)
}

// Offset returns the month n months away from the month that now falls in.
// It works from the first day of the month, so it never overflows on days such
// as 31 October plus six months, which time.AddDate would push into May.
func Offset(now time.Time, n int) string {
	return time.Date(now.Year(), now.Month()+time.Month(n), 1, 0, 0, 0, 0, now.Location()).Format(layout)
}

// Add returns the month n months after m (before m when n is negative).
func Add(m string, n int) (string, error) {
	t, err := time.Parse(layout, m)
	if err != nil {
		return "", fmt.Errorf("invalid month %q: %w", m, err)
	}
	return Offset(t, n), nil
}

// Count returns the number of months from start to end, counting both ends, so
// a month to itself is 1. It is zero or negative when end is before start.
func Count(start, end string) (int, error) {
	s, err := time.Parse(layout, start)
	if err != nil {
		return 0, fmt.Errorf("invalid start month %q: %w", start, err)
	}
	e, err := time.Parse(layout, end)
	if err != nil {
		return 0, fmt.Errorf("invalid end month %q: %w", end, err)
	}
	years := e.Year() - s.Year()
	months := int(e.Month()) - int(s.Month())
	return years*12 + months + 1, nil
}

// Elapsed returns how many months of the range start..end have begun by the
// current month, from 0 (not started) up to the length of the range.
func Elapsed(start, end, current string) (int, error) {
	total, err := Count(start, end)
	if err != nil {
		return 0, err
	}
	sofar, err := Count(start, current)
	if err != nil {
		return 0, fmt.Errorf("count elapsed months: %w", err)
	}
	return min(max(sofar, 0), max(total, 0)), nil
}

// Range returns the months from start to end inclusive, with each month's
// zero-based position. It yields nothing when end is before start, and returns
// an error when either month is invalid.
func Range(start, end string) (iter.Seq2[int, string], error) {
	count, err := Count(start, end)
	if err != nil {
		return nil, err
	}
	first, err := time.Parse(layout, start)
	if err != nil {
		return nil, fmt.Errorf("invalid start month %q: %w", start, err)
	}
	return func(yield func(int, string) bool) {
		for i := range count {
			if !yield(i, Offset(first, i)) {
				return
			}
		}
	}, nil
}

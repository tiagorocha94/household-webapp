package month

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	for _, ok := range []string{"2026-01", "2026-12", "1999-06"} {
		if err := Validate(ok); err != nil {
			t.Errorf("Validate(%q) error = %v, want nil", ok, err)
		}
	}
	for _, bad := range []string{"", "2026", "2026-00", "2026-13", "2026-1", "26-01", "2026/01", "2026-01-01", " 2026-01"} {
		err := Validate(bad)
		if err == nil || !strings.Contains(err.Error(), "YYYY-MM") {
			t.Errorf("Validate(%q) error = %v, want a YYYY-MM error", bad, err)
		}
	}
}

func TestValidateDay(t *testing.T) {
	t.Parallel()

	for _, ok := range []string{"2026-01-01", "2026-12-31", "2028-02-29", "1999-06-15"} {
		if err := ValidateDay(ok); err != nil {
			t.Errorf("ValidateDay(%q) error = %v, want nil", ok, err)
		}
	}
	bad := []string{
		"", "2026", "2026-01", "2026-1-5", "2026-01-5", "26-01-01", "2026/01/01",
		"2026-02-30", "2027-02-29", "2026-04-31", "2026-00-10", "2026-13-01",
		"2026-01-00", "2026-01-32", " 2026-01-01", "2026-01-01 ", "2026-01-01T00:00:00Z",
	}
	for _, d := range bad {
		err := ValidateDay(d)
		if err == nil || !strings.Contains(err.Error(), "YYYY-MM-DD") {
			t.Errorf("ValidateDay(%q) error = %v, want a YYYY-MM-DD error", d, err)
		}
	}
}

func TestCurrentUsesTheTimeZoneOfNow(t *testing.T) {
	t.Parallel()

	// 00:30 on 1 March in UTC+1 is still February in UTC.
	lisbon := time.Date(2026, 3, 1, 0, 30, 0, 0, time.FixedZone("WEST", 3600))
	if got := Current(lisbon); got != "2026-03" {
		t.Errorf("Current(lisbon) = %s, want 2026-03", got)
	}
	if got := Current(lisbon.UTC()); got != "2026-02" {
		t.Errorf("Current(utc) = %s, want 2026-02", got)
	}
}

func TestOffset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		from time.Time
		n    int
		want string
	}{
		{"end of month forward", time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC), 6, "2027-04"},
		{"leap day back a year", time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC), -12, "2027-02"},
		{"across year boundary", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), -2, "2025-11"},
		{"zero", time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC), 0, "2026-05"},
		{"several years", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), 58, "2031-08"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := Offset(tt.from, tt.n); got != tt.want {
				t.Fatalf("Offset() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestAdd(t *testing.T) {
	t.Parallel()

	tests := []struct {
		month string
		n     int
		want  string
	}{
		{"2026-10", 6, "2027-04"},
		{"2026-01", -1, "2025-12"},
		{"2026-12", 1, "2027-01"},
		{"2026-03", 0, "2026-03"},
		{"2026-03", -15, "2024-12"},
	}
	for _, tt := range tests {
		got, err := Add(tt.month, tt.n)
		if err != nil || got != tt.want {
			t.Errorf("Add(%q, %d) = %q, %v, want %q", tt.month, tt.n, got, err, tt.want)
		}
	}
	if _, err := Add("2026-13", 1); err == nil {
		t.Error("Add(2026-13, 1) succeeded, want error")
	}
}

func TestCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		start, end string
		want       int
	}{
		{"2026-01", "2026-01", 1},
		{"2026-01", "2026-12", 12},
		{"2025-11", "2027-06", 20},
		{"2026-09", "2031-06", 58},
		{"2025-12", "2026-01", 2},
		{"2026-05", "2026-04", 0},
		{"2026-05", "2026-01", -3},
	}
	for _, tt := range tests {
		got, err := Count(tt.start, tt.end)
		if err != nil || got != tt.want {
			t.Errorf("Count(%q, %q) = %d, %v, want %d", tt.start, tt.end, got, err, tt.want)
		}
	}

	if _, err := Count("2026-13", "2026-12"); err == nil || !strings.Contains(err.Error(), "start month") {
		t.Errorf("Count(bad start) error = %v, want a start month error", err)
	}
	if _, err := Count("2026-01", "nope"); err == nil || !strings.Contains(err.Error(), "end month") {
		t.Errorf("Count(bad end) error = %v, want an end month error", err)
	}
}

func TestElapsed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		start, end, current string
		want                int
	}{
		{"before start", "2026-09", "2031-06", "2026-08", 0},
		{"first month", "2026-09", "2031-06", "2026-09", 1},
		{"middle", "2025-11", "2027-06", "2026-10", 12},
		{"last month", "2026-01", "2026-03", "2026-03", 3},
		{"after end", "2026-01", "2026-03", "2027-01", 3},
		{"long before start", "2099-01", "2099-12", "2026-10", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Elapsed(tt.start, tt.end, tt.current)
			if err != nil || got != tt.want {
				t.Fatalf("Elapsed(%q, %q, %q) = %d, %v, want %d", tt.start, tt.end, tt.current, got, err, tt.want)
			}
		})
	}

	if _, err := Elapsed("2026-01", "2026-03", "bad"); err == nil {
		t.Error("Elapsed(bad current) succeeded, want error")
	}
}

func collect(t *testing.T, start, end string) []string {
	t.Helper()
	seq, err := Range(start, end)
	if err != nil {
		t.Fatalf("Range(%q, %q) error = %v", start, end, err)
	}
	var months []string
	for i, m := range seq {
		if i != len(months) {
			t.Fatalf("Range(%q, %q) position = %d, want %d", start, end, i, len(months))
		}
		months = append(months, m)
	}
	return months
}

func TestRange(t *testing.T) {
	t.Parallel()

	if got, want := collect(t, "2025-11", "2026-02"), []string{"2025-11", "2025-12", "2026-01", "2026-02"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Range across a year = %v, want %v", got, want)
	}
	if got, want := collect(t, "2026-05", "2026-05"), []string{"2026-05"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Range of one month = %v, want %v", got, want)
	}
	if got := collect(t, "2026-05", "2026-04"); len(got) != 0 {
		t.Errorf("Range with end before start = %v, want nothing", got)
	}
	if got := collect(t, "2026-09", "2031-06"); len(got) != 58 || got[0] != "2026-09" || got[57] != "2031-06" {
		t.Errorf("long Range has %d months, want 58 from 2026-09 to 2031-06", len(got))
	}

	if _, err := Range("2026-13", "2026-12"); err == nil {
		t.Error("Range(bad start) succeeded, want error")
	}
	if _, err := Range("2026-01", "bad"); err == nil {
		t.Error("Range(bad end) succeeded, want error")
	}
}

func TestRangeStopsWhenTheLoopBreaks(t *testing.T) {
	t.Parallel()

	seq, err := Range("2026-01", "2026-12")
	if err != nil {
		t.Fatalf("Range() error = %v", err)
	}
	var seen int
	for range seq {
		seen++
		if seen == 3 {
			break
		}
	}
	if seen != 3 {
		t.Errorf("seen = %d, want 3", seen)
	}
}

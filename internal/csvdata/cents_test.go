package csvdata

import "testing"

func TestParseNonNegativeCents(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"12.34", "12,34", "-1", "+1", "", "one", "9223372036854775808"} {
		t.Run(value, func(t *testing.T) {
			t.Parallel()

			if _, err := ParseNonNegativeCents(value, "amount_cents"); err == nil {
				t.Fatalf("ParseNonNegativeCents(%q) succeeded, want error", value)
			}
		})
	}

	got, err := ParseNonNegativeCents(" 12345 ", "amount_cents")
	if err != nil || got != 12345 {
		t.Fatalf("ParseNonNegativeCents() = (%d, %v), want (12345, nil)", got, err)
	}
}

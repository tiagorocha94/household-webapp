package money

import (
	"reflect"
	"testing"
)

func TestSplitSumsExactly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		total Cents
		n     int
	}{{1250000, 58}, {116300, 14}, {10000, 3}, {1, 5}, {0, 4}}

	for _, tt := range tests {
		parts := Split(tt.total, tt.n)
		if len(parts) != tt.n {
			t.Fatalf("Split(%d, %d) returned %d parts", tt.total, tt.n, len(parts))
		}
		var sum Cents
		for _, p := range parts {
			sum += p
		}
		if sum != tt.total {
			t.Errorf("Split(%d, %d) sums to %d", tt.total, tt.n, sum)
		}
		if parts[0]-parts[len(parts)-1] > 1 {
			t.Errorf("Split(%d, %d) parts differ by more than one cent: %v", tt.total, tt.n, parts)
		}
	}
}

func TestSplitGivesExtraCentsToFirstParts(t *testing.T) {
	t.Parallel()

	want := []Cents{3334, 3333, 3333}
	if got := Split(10000, 3); !reflect.DeepEqual(got, want) {
		t.Fatalf("Split(10000, 3) = %v, want %v", got, want)
	}
}

func TestSplitRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	if Split(100, 0) != nil || Split(-1, 3) != nil {
		t.Fatal("Split should return nil for n <= 0 or a negative total")
	}
}

package savings

import (
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/money"
)

func TestSummarizeWithoutASnapshotUsesTheMovements(t *testing.T) {
	t.Parallel()

	got := Summarize(Account{TargetCents: 10000}, []Movement{
		{Type: TypeDeposit, AmountCents: 5000},
		{Type: TypeDeposit, AmountCents: 3000},
		{Type: TypeWithdrawal, AmountCents: 1000},
		{Type: TypeWithdrawal, AmountCents: 500},
	}, nil)

	if got.DepositedCents != 8000 || got.WithdrawnCents != 1500 {
		t.Errorf("totals = %+v, want 8000 deposited, 1500 withdrawn", got)
	}
	if got.BalanceCents != 6500 || got.InterestCents != 0 || got.ProgressPct != 65 {
		t.Errorf("balance %d, interest %d, progress %d%%, want 6500, 0, 65%%", got.BalanceCents, got.InterestCents, got.ProgressPct)
	}
	if got.MovementCount != 4 || got.SnapshotCount != 0 || got.LastSnapshotMonth != "" {
		t.Errorf("counts = %+v", got)
	}
}

func TestSummarizeUsesTheLatestSnapshotAndDerivesInterest(t *testing.T) {
	t.Parallel()

	got := Summarize(Account{TargetCents: 5000}, []Movement{
		{Type: TypeDeposit, AmountCents: 3500},
		{Type: TypeWithdrawal, AmountCents: 500},
	}, []Snapshot{
		{ID: 3, Month: "2026-09", ValueCents: 3100},
		{ID: 1, Month: "2026-11", ValueCents: 3140},
		{ID: 2, Month: "2026-11", ValueCents: 3150}, // same month, higher id: newer
	})

	if got.BalanceCents != 3150 || got.InterestCents != 150 || got.LastSnapshotMonth != "2026-11" {
		t.Errorf("balance %d, interest %d, last %q, want 3150, 150, 2026-11", got.BalanceCents, got.InterestCents, got.LastSnapshotMonth)
	}
	if got.ProgressPct != 63 {
		t.Errorf("progress = %d%%, want 63%%", got.ProgressPct)
	}
}

func TestSummarizeProgressEdges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target int64
		value  int64
		want   int
	}{
		{"no target", 0, 5000, 0},
		{"over the target", 1000, 1500, 150},
		{"empty", 1000, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Summarize(Account{TargetCents: money.Cents(tt.target)}, nil, []Snapshot{{Month: "2026-11", ValueCents: money.Cents(tt.value)}})
			if got.ProgressPct != tt.want {
				t.Errorf("progress = %d, want %d", got.ProgressPct, tt.want)
			}
		})
	}
}

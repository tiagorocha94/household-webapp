package goals

import "testing"

func TestMovementDirectionPerspectives(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		movementType  string
		accountInflow bool
		goalInflow    bool
	}{
		{name: "deposit", movementType: MovementDeposit, accountInflow: true, goalInflow: false},
		{name: "allocation", movementType: MovementAllocation, accountInflow: false, goalInflow: true},
		{name: "expense", movementType: MovementExpense, accountInflow: false, goalInflow: false},
		{name: "release", movementType: MovementRelease, accountInflow: true, goalInflow: false},
		{name: "cover", movementType: MovementCover, accountInflow: false, goalInflow: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			movement := Movement{Type: tt.movementType}
			if got := movement.IsInflow(); got != tt.accountInflow {
				t.Errorf("IsInflow() = %v, want %v", got, tt.accountInflow)
			}
			if got := movement.IsInflowForGoal(); got != tt.goalInflow {
				t.Errorf("IsInflowForGoal() = %v, want %v", got, tt.goalInflow)
			}
		})
	}
}

package csvdata

import (
	"fmt"
	"strings"
)

// MovementIDLength is the length of a cashflow movement's sync_id, a UUID.
const MovementIDLength = 36

// ParseMovementID reads the optional movement column of a savings, investment
// or credit row: the sync_id of the cashflow movement the row was added with.
// An empty value means the row is not linked to a movement.
func ParseMovementID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value != "" && len(value) != MovementIDLength {
		return "", fmt.Errorf("invalid movement %q: expected a sync_id of %d characters or empty", value, MovementIDLength)
	}
	return value, nil
}

package csvdata_test

import (
	"strings"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/csvdata"
)

func TestParseMovementID(t *testing.T) {
	t.Parallel()

	id := "123e4567-e89b-12d3-a456-426614174000"
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"empty is not linked", "", "", false},
		{"blank is not linked", "  ", "", false},
		{"a sync id", " " + id + " ", id, false},
		{"too short", "abc", "", true},
		{"too long", id + "0", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := csvdata.ParseMovementID(tt.in)
			if (err != nil) != tt.wantErr || got != tt.want {
				t.Errorf("ParseMovementID(%q) = %q, %v; want %q, error %v", tt.in, got, err, tt.want, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), "movement") {
				t.Errorf("the message %q does not name the column", err)
			}
		})
	}
}

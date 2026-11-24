package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

func TestErrorStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err  error
		want int
	}{
		{fmt.Errorf("x: %w", apperr.ErrBadRequest), http.StatusBadRequest},
		{fmt.Errorf("x: %w", apperr.ErrNotFound), http.StatusNotFound},
		{errors.New("database exploded"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		got, msg := ErrorStatus(testutil.Logger(), "test", tt.err)
		if got != tt.want {
			t.Errorf("ErrorStatus(%v) = %d, want %d", tt.err, got, tt.want)
		}
		if got == http.StatusInternalServerError && msg != "Internal Server Error" {
			t.Errorf("500 leaked details: %q", msg)
		}
	}
}

func TestPathID(t *testing.T) {
	t.Parallel()

	for value, ok := range map[string]bool{"1": true, "42": true, "0": false, "-3": false, "abc": false, "": false} {
		r := httptest.NewRequest(http.MethodGet, "/x", nil)
		r.SetPathValue("id", value)
		id, err := PathID(r, "thing id")
		if ok && (err != nil || id <= 0) {
			t.Errorf("PathID(%q) = %d, %v, want a valid ID", value, id, err)
		}
		if !ok && !errors.Is(err, apperr.ErrBadRequest) {
			t.Errorf("PathID(%q) error = %v, want bad request", value, err)
		}
	}
}

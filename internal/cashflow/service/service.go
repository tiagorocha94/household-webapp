package service

import (
	"fmt"
	"log/slog"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/cashflow/store"
	"github.com/tiagorocha94/household-webapp/internal/month"
)

// Service handles cashflow reads and the writes of enrolled devices.
type Service struct {
	store *store.Store
	// mirror and log are set by EnableMirror.
	mirror Mirror
	log    *slog.Logger
	// detailMirrors are set by EnableDetailMirrors.
	detailMirrors []Mirror
}

// New creates a Service backed by the given Store.
func New(store *store.Store) *Service {
	return &Service{store: store}
}

// validateMonth returns a bad-request error unless m is a YYYY-MM month.
func validateMonth(m string) error {
	if err := month.Validate(m); err != nil {
		return fmt.Errorf("cashflow month: %w: %w", apperr.ErrBadRequest, err)
	}
	return nil
}

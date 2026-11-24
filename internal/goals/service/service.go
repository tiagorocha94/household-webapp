package service

import (
	"time"

	"github.com/tiagorocha94/household-webapp/internal/goals/store"
)

// Service handles business logic for goal accounts, goals, and movements.
type Service struct {
	store *store.Store
	now   func() time.Time
}

// New creates a new Service backed by the given Store.
func New(store *store.Store) *Service {
	return &Service{store: store, now: time.Now}
}

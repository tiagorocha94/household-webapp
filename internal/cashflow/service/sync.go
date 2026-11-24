package service

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/cashflow"
	"github.com/tiagorocha94/household-webapp/internal/cashflow/store"
	"github.com/tiagorocha94/household-webapp/internal/details"
	"github.com/tiagorocha94/household-webapp/internal/money"
)

// MaxSyncOperations is the most operations one batch may hold.
const MaxSyncOperations = 200

// Kinds of sync operation.
const (
	OpUpsert = "upsert"
	OpDelete = "delete"
)

// Mirror brings the movements CSV file up to date after a change. It is
// *cashflow.CSVMirror in the server.
type Mirror interface {
	Flush() error
}

// SyncOperation is one change a device asks for. A delete only needs SyncID.
type SyncOperation struct {
	Op          string
	SyncID      string
	Category    string
	Date        string
	AmountCents money.Cents
	Description string
	// Detail is where the money went. Nil leaves the detail as it is; an empty
	// Detail removes it.
	Detail *details.Detail
}

// SyncResult is what happened to one operation: Rejected is empty when it was
// applied (or had nothing to change) and otherwise says why it was refused.
type SyncResult struct {
	SyncID   string
	Rejected string
}

// EnableMirror makes every batch that changes something rewrite the movements
// file before it returns. A file that cannot be written is logged and does not
// fail the batch: the changes are in the database, the file stays marked as
// behind, and the next batch or start-up writes it.
func (s *Service) EnableMirror(m Mirror, log *slog.Logger) {
	s.mirror, s.log = m, log
}

// EnableDetailMirrors makes every batch that changes the detail of a movement
// rewrite the CSV files of the modules the detail belongs to, in the same way
// as EnableMirror does for the movements file.
func (s *Service) EnableDetailMirrors(ms ...Mirror) {
	s.detailMirrors = ms
}

// ApplySync applies a device's operations for one person, in order, and
// returns one result per operation. Operations are independent: one that is
// refused (a bad value, an unknown category, another person's sync ID) is
// reported and the rest still apply. Both kinds are idempotent, so a batch that
// is sent again changes nothing. The batch is one transaction, so if the
// database itself fails nothing is kept and the device sends it again.
//
// It fails with a bad-request error for a batch of more than MaxSyncOperations.
func (s *Service) ApplySync(userID int64, ops []SyncOperation) ([]SyncResult, error) {
	if len(ops) > MaxSyncOperations {
		return nil, fmt.Errorf("a batch holds at most %d operations, got %d: %w", MaxSyncOperations, len(ops), apperr.ErrBadRequest)
	}

	results := make([]SyncResult, 0, len(ops))
	err := s.store.Write(func(w *store.Writer) error {
		results = results[:0]
		for _, op := range ops {
			reason, err := applyOperation(w, userID, op)
			if err != nil {
				return err
			}
			results = append(results, SyncResult{SyncID: op.SyncID, Rejected: reason})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if s.mirror != nil {
		if err := s.mirror.Flush(); err != nil {
			s.log.Error("write the cashflow movements file", "error", err)
		}
	}
	for _, m := range s.detailMirrors {
		if err := m.Flush(); err != nil {
			s.log.Error("write the csv files of a movement detail", "error", err)
		}
	}
	return results, nil
}

// applyOperation does one operation. It returns a reason when the operation is
// refused, and an error only when the database fails.
func applyOperation(w *store.Writer, userID int64, op SyncOperation) (string, error) {
	switch op.Op {
	case OpUpsert:
		m := cashflow.MovementWrite{
			SyncID:      op.SyncID,
			UserID:      userID,
			Category:    op.Category,
			Date:        op.Date,
			AmountCents: op.AmountCents,
			Description: strings.TrimSpace(op.Description),
		}
		if err := m.Validate(); err != nil {
			return reasonOf(err), nil
		}
		in := details.Input{
			UserID: userID, SyncID: op.SyncID, Category: m.Category,
			Month: m.Date[:7], AmountCents: m.AmountCents, Detail: op.Detail,
		}
		// The detail is checked first so that a movement whose detail is
		// refused is not written at all.
		reason, err := details.Check(w.Tx(), in)
		if err != nil || reason != "" {
			return reason, err
		}
		if _, err := w.UpsertMovement(m); err != nil {
			return refusal(err)
		}
		return "", details.Apply(w.Tx(), in)
	case OpDelete:
		if !cashflow.ValidSyncID(op.SyncID) {
			return fmt.Sprintf("sync_id %q is not a lowercase UUID", op.SyncID), nil
		}
		if _, err := w.DeleteMovement(userID, op.SyncID); err != nil {
			return refusal(err)
		}
		return "", details.Remove(w.Tx(), op.SyncID)
	default:
		return fmt.Sprintf("unknown operation %q, want %q or %q", op.Op, OpUpsert, OpDelete), nil
	}
}

// refusal turns the errors that mean "this operation is refused" into a reason
// and passes any other error on.
func refusal(err error) (string, error) {
	switch {
	case err == nil:
		return "", nil
	case errors.Is(err, cashflow.ErrUnknownCategory):
		return reasonOf(err), nil
	case errors.Is(err, cashflow.ErrSyncIDTaken):
		return "sync_id belongs to another person", nil
	default:
		return "", err
	}
}

// reasonOf is the message of a refusal without the bad-request marker.
func reasonOf(err error) string {
	return strings.TrimSuffix(err.Error(), ": "+apperr.ErrBadRequest.Error())
}

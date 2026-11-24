// Package store provides savings persistence.
package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/savings"
)

const accountColumns = `a.id, a.name, COALESCE(u.username, ''), COALESCE(a.target_cents, 0),
COALESCE(a.notes, ''), a.created_at, a.updated_at`

// Store provides savings persistence.
type Store struct {
	db *sql.DB
}

type rowScanner interface {
	Scan(dest ...any) error
}

// New creates a Store backed by the given database connection.
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// ListAccounts returns every account, ordered by name.
func (s *Store) ListAccounts() ([]savings.Account, error) {
	rows, err := s.db.Query(
		`SELECT ` + accountColumns + `
		 FROM savings_accounts a LEFT JOIN users u ON u.id = a.user_id
		 ORDER BY a.name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list savings accounts: %w", err)
	}
	defer rows.Close()

	var accounts []savings.Account
	for rows.Next() {
		var a savings.Account
		if err := scanAccount(rows, &a); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

// GetAccountByID returns an account, or nil when it does not exist.
func (s *Store) GetAccountByID(id int64) (*savings.Account, error) {
	row := s.db.QueryRow(
		`SELECT `+accountColumns+`
		 FROM savings_accounts a LEFT JOIN users u ON u.id = a.user_id
		 WHERE a.id = ?`, id,
	)
	var a savings.Account
	if err := scanAccount(row, &a); errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &a, nil
}

func scanAccount(row rowScanner, a *savings.Account) error {
	err := row.Scan(&a.ID, &a.Name, &a.Username, &a.TargetCents, &a.Notes, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err != nil {
		return fmt.Errorf("scan savings account: %w", err)
	}
	return nil
}

// ListMovements returns the movements of an account, newest month first.
func (s *Store) ListMovements(accountID int64) ([]savings.Movement, error) {
	rows, err := s.db.Query(
		`SELECT id, account_id, month, type, amount_cents, COALESCE(description, ''), created_at
		 FROM savings_movements WHERE account_id = ?
		 ORDER BY month DESC, id DESC`, accountID,
	)
	if err != nil {
		return nil, fmt.Errorf("list movements of savings account %d: %w", accountID, err)
	}
	defer rows.Close()

	var movements []savings.Movement
	for rows.Next() {
		var m savings.Movement
		if err := rows.Scan(&m.ID, &m.AccountID, &m.Month, &m.Type, &m.AmountCents, &m.Description, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan savings movement: %w", err)
		}
		movements = append(movements, m)
	}
	return movements, rows.Err()
}

// ListSnapshots returns the snapshots of an account, newest month first.
func (s *Store) ListSnapshots(accountID int64) ([]savings.Snapshot, error) {
	rows, err := s.db.Query(
		`SELECT id, account_id, month, value_cents, COALESCE(description, ''), created_at
		 FROM savings_snapshots WHERE account_id = ?
		 ORDER BY month DESC, id DESC`, accountID,
	)
	if err != nil {
		return nil, fmt.Errorf("list snapshots of savings account %d: %w", accountID, err)
	}
	defer rows.Close()

	var snapshots []savings.Snapshot
	for rows.Next() {
		var sn savings.Snapshot
		if err := rows.Scan(&sn.ID, &sn.AccountID, &sn.Month, &sn.ValueCents, &sn.Description, &sn.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan savings snapshot: %w", err)
		}
		snapshots = append(snapshots, sn)
	}
	return snapshots, rows.Err()
}

// DepositsByMonth returns what the deposits of every account add up to in each
// month, keyed by YYYY-MM.
func (s *Store) DepositsByMonth() (map[string]money.Cents, error) {
	rows, err := s.db.Query(
		`SELECT month, SUM(amount_cents) FROM savings_movements
		 WHERE type = 'deposit' GROUP BY month`)
	if err != nil {
		return nil, fmt.Errorf("sum savings deposits by month: %w", err)
	}
	defer rows.Close()

	totals := make(map[string]money.Cents)
	for rows.Next() {
		var month string
		var total money.Cents
		if err := rows.Scan(&month, &total); err != nil {
			return nil, fmt.Errorf("scan savings deposit total: %w", err)
		}
		totals[month] = total
	}
	return totals, rows.Err()
}

package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/goals"
)

// ListAccounts returns all accounts, ordered by name.
func (s *Store) ListAccounts() ([]goals.Account, error) {
	rows, err := s.db.Query(
		`SELECT id, name, monthly_deposit_cents, created_at
		 FROM goal_accounts
		 ORDER BY name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []goals.Account
	for rows.Next() {
		var a goals.Account
		if err := rows.Scan(&a.ID, &a.Name, &a.MonthlyDepositCents, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		accounts = append(accounts, a)
	}

	return accounts, rows.Err()
}

// GetAccountByID retrieves an account by its ID. Returns nil if not found.
func (s *Store) GetAccountByID(id int64) (*goals.Account, error) {
	row := s.db.QueryRow("SELECT id, name, monthly_deposit_cents, created_at FROM goal_accounts WHERE id = ?", id)

	a := &goals.Account{}
	err := row.Scan(&a.ID, &a.Name, &a.MonthlyDepositCents, &a.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query account %d: %w", id, err)
	}

	return a, nil
}

// GetAccountBalance retrieves the computed balance for an account from the view.
func (s *Store) GetAccountBalance(accountID int64) (*goals.AccountBalance, error) {
	row := s.db.QueryRow(
		`SELECT account_id, total_deposits_cents, total_allocated_cents, total_expenses_cents, total_released_cents, total_covered_cents, balance_cents, unallocated_cents
		 FROM goal_account_balance_view
		 WHERE account_id = ?`,
		accountID,
	)

	b := &goals.AccountBalance{}
	err := row.Scan(&b.AccountID, &b.TotalDepositsCents, &b.TotalAllocatedCents, &b.TotalExpensesCents, &b.TotalReleasedCents, &b.TotalCoveredCents, &b.BalanceCents, &b.UnallocatedCents)
	if errors.Is(err, sql.ErrNoRows) {
		return &goals.AccountBalance{AccountID: accountID}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query balance for account %d: %w", accountID, err)
	}

	return b, nil
}

package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/credits"
)

const contractColumns = `id, name, category, contract_number, lender,
original_principal_cents, start_month, COALESCE(expected_end_month, ''), 
COALESCE(notes, ''), created_at, updated_at, COALESCE(expected_day, 0)`

type rowScanner interface {
	Scan(dest ...any) error
}

// GetContractByID returns a contract by ID, or nil when it does not exist.
func (s *Store) GetContractByID(id int64) (*credits.Contract, error) {
	row := s.db.QueryRow("SELECT "+contractColumns+" FROM credit_contracts WHERE id = ?", id)
	return scanContract(row)
}

// ListContractSummaries returns all contracts, active ones first.
func (s *Store) ListContractSummaries() ([]credits.ContractSummary, error) {
	rows, err := s.db.Query(
		`SELECT contract_id, name, category,
		        contract_number, lender, original_principal_cents, start_month,
		        COALESCE(expected_end_month, ''), status, COALESCE(notes, ''),
		        created_at, updated_at, payment_count, total_principal_paid_cents,
		        total_interest_paid_cents, total_paid_cents, outstanding_principal_cents,
		        progress_pct, COALESCE(last_payment_month, ''),
		        COALESCE((SELECT expected_day FROM credit_contracts WHERE id = contract_id), 0)
		 FROM credit_contract_summary_view
		 ORDER BY status ASC, name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list credit contracts: %w", err)
	}
	defer rows.Close()

	var summaries []credits.ContractSummary
	for rows.Next() {
		var summary credits.ContractSummary
		if err := scanContractSummary(rows, &summary); err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	return summaries, rows.Err()
}

// GetContractSummaryByID returns a contract summary by ID, or nil when absent.
func (s *Store) GetContractSummaryByID(id int64) (*credits.ContractSummary, error) {
	row := s.db.QueryRow(
		`SELECT contract_id, name, category, contract_number, lender,
		        original_principal_cents, start_month, COALESCE(expected_end_month, ''),
		        status, COALESCE(notes, ''), created_at, updated_at, payment_count,
		        total_principal_paid_cents, total_interest_paid_cents, total_paid_cents,
		        outstanding_principal_cents, progress_pct, COALESCE(last_payment_month, ''),
		        COALESCE((SELECT expected_day FROM credit_contracts WHERE id = contract_id), 0)
		 FROM credit_contract_summary_view WHERE contract_id = ?`,
		id,
	)

	var summary credits.ContractSummary
	if err := scanContractSummary(row, &summary); errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &summary, nil
}

func scanContract(row rowScanner) (*credits.Contract, error) {
	var contract credits.Contract
	err := row.Scan(
		&contract.ID, &contract.Name, &contract.Category,
		&contract.ContractNumber, &contract.Lender, &contract.OriginalPrincipalCents,
		&contract.StartMonth, &contract.ExpectedEndMonth, &contract.Notes,
		&contract.CreatedAt, &contract.UpdatedAt, &contract.ExpectedDay,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan credit contract: %w", err)
	}
	return &contract, nil
}

func scanContractSummary(row rowScanner, summary *credits.ContractSummary) error {
	err := row.Scan(
		&summary.ID, &summary.Name, &summary.Category,
		&summary.ContractNumber, &summary.Lender, &summary.OriginalPrincipalCents,
		&summary.StartMonth, &summary.ExpectedEndMonth, &summary.Status,
		&summary.Notes, &summary.CreatedAt, &summary.UpdatedAt,
		&summary.PaymentCount, &summary.TotalPrincipalPaidCents,
		&summary.TotalInterestPaidCents, &summary.TotalPaidCents,
		&summary.OutstandingPrincipalCents, &summary.ProgressPct,
		&summary.LastPaymentMonth, &summary.ExpectedDay,
	)
	if err != nil {
		return fmt.Errorf("scan credit contract summary: %w", err)
	}
	return nil
}

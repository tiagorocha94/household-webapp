package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/credits"
)

const paymentColumns = `id, contract_id, month, type, principal_cents,
interest_cents, COALESCE(description, ''), created_at, updated_at`

// ListPaymentsByContractID returns all payments for a contract, newest first.
func (s *Store) ListPaymentsByContractID(contractID int64) ([]credits.Payment, error) {
	rows, err := s.db.Query(
		`SELECT `+paymentColumns+`
		 FROM credit_payments WHERE contract_id = ?
		 ORDER BY month DESC, type ASC, id DESC`,
		contractID,
	)
	if err != nil {
		return nil, fmt.Errorf("list payments for credit contract %d: %w", contractID, err)
	}
	defer rows.Close()

	var payments []credits.Payment
	for rows.Next() {
		payment, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		payments = append(payments, *payment)
	}
	return payments, rows.Err()
}

func scanPayment(row rowScanner) (*credits.Payment, error) {
	var payment credits.Payment
	err := row.Scan(
		&payment.ID, &payment.ContractID, &payment.Month, &payment.Type,
		&payment.PrincipalCents, &payment.InterestCents, &payment.Description,
		&payment.CreatedAt, &payment.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan credit payment: %w", err)
	}
	return &payment, nil
}

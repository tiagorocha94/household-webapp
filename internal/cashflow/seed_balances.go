package cashflow

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/tiagorocha94/household-webapp/internal/csvdata"
	"github.com/tiagorocha94/household-webapp/internal/database"
	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/month"
)

// parseSignedCents parses raw integer cents that may be negative: a balance can
// be below zero. Decimal separators, empty values and overflow are rejected.
func parseSignedCents(value, field string) (money.Cents, error) {
	value = strings.TrimSpace(value)
	digits := strings.TrimPrefix(value, "-")
	if digits == "" {
		return 0, fmt.Errorf("%s must be an integer", field)
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid %s %q: expected an integer number of cents", field, value)
		}
	}
	cents, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", field, value, err)
	}
	return money.Cents(cents), nil
}

// mirrorBalances makes the balances table match the file: each row is added or
// changed, and a balance that left the file is removed. A person has at most one
// balance per month.
func mirrorBalances(q database.Querier, rows [][]string, log *slog.Logger) error {
	userIDs, err := loadIDs(q, "SELECT id, username FROM users")
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}
	keep := make(map[string]bool, len(rows))
	for i, row := range rows {
		if len(row) < 3 {
			return fmt.Errorf("row %d: expected 3 columns", i+2)
		}
		user, m := strings.TrimSpace(row[0]), strings.TrimSpace(row[1])
		userID, ok := userIDs[user]
		if !ok {
			return fmt.Errorf("row %d: user %q not found", i+2, user)
		}
		if err := month.Validate(m); err != nil {
			return fmt.Errorf("row %d: month: %w", i+2, err)
		}
		cents, err := parseSignedCents(row[2], "balance_cents")
		if err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}
		key := user + "|" + m
		if keep[key] {
			return fmt.Errorf("row %d: %s has more than one balance for %s", i+2, user, m)
		}
		keep[key] = true

		// The update only fires when the balance changed.
		if _, err := q.Exec(
			`INSERT INTO cashflow_balances (user_id, month, balance_cents) VALUES (?, ?, ?)
			 ON CONFLICT(user_id, month) DO UPDATE SET
			     balance_cents = excluded.balance_cents, updated_at = CURRENT_TIMESTAMP
			 WHERE cashflow_balances.balance_cents IS NOT excluded.balance_cents`,
			userID, m, cents,
		); err != nil {
			return fmt.Errorf("row %d: save the balance of %s for %s: %w", i+2, user, m, err)
		}
	}
	return csvdata.DeleteUnlisted(log, q, "cashflow_balances",
		`SELECT b.id, u.username || '|' || b.month
		 FROM cashflow_balances b JOIN users u ON u.id = b.user_id`,
		keep, nameScan)
}

func exportBalances(db database.Querier) ([][]string, error) {
	rows, err := db.Query(
		`SELECT u.username, b.month, b.balance_cents
		 FROM cashflow_balances b JOIN users u ON u.id = b.user_id
		 ORDER BY u.username ASC, b.month ASC`)
	if err != nil {
		return nil, fmt.Errorf("query cashflow balances for export: %w", err)
	}
	defer rows.Close()

	result := [][]string{balancesHeader}
	for rows.Next() {
		var user, m string
		var cents money.Cents
		if err := rows.Scan(&user, &m, &cents); err != nil {
			return nil, fmt.Errorf("scan cashflow balance for export: %w", err)
		}
		result = append(result, []string{user, m, csvdata.FormatCents(cents)})
	}
	return result, rows.Err()
}

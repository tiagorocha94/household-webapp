package goals

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/csvdata"
	"github.com/tiagorocha94/household-webapp/internal/database"
	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/month"
)

// SeedModule imports, syncs and exports the goal CSV files and satisfies the
// seed package's Module interface.
type SeedModule struct{}

type movementRow struct {
	accountID   int64
	goalID      int64 // 0 for deposits
	movType     string
	month       string
	amountCents money.Cents
	description string
}

// Name returns the module's name for logs and errors.
func (SeedModule) Name() string { return "goals" }

// Files lists the CSV files the module imports, with their headers.
func (SeedModule) Files() []csvdata.File {
	return []csvdata.File{
		{Name: "goal_accounts.csv", Header: []string{"name", "monthly_deposit_cents"}},
		{Name: "goal_account_goals.csv", Header: []string{"account", "name", "target_cents", "start_month", "end_month", "status"}},
		{Name: "goal_account_movements.csv", Header: []string{"account", "type", "goal", "month", "amount_cents", "description"}},
	}
}

// Import mirrors the goal CSV files that exist into the database.
func (SeedModule) Import(tx *sql.Tx, files map[string][][]string, log *slog.Logger) error {
	return applyGoals(tx, files, log)
}

// Sync inserts the allocations that are due and settles the goals that were
// completed, as of now.
func (SeedModule) Sync(db *sql.DB, now time.Time, log *slog.Logger) error {
	return syncGoals(db, now, log)
}

// Export returns the goal tables in write order.
func (SeedModule) Export(db *sql.DB) ([]csvdata.Table, error) {
	steps := []struct {
		file string
		fn   func(*sql.DB) ([][]string, error)
	}{
		{"goal_accounts.csv", exportAccounts},
		{"goal_account_goals.csv", exportGoals},
		{"goal_account_movements.csv", exportMovements},
	}
	tables := make([]csvdata.Table, 0, len(steps))
	for _, step := range steps {
		rows, err := step.fn(db)
		if err != nil {
			return nil, fmt.Errorf("export %s: %w", step.file, err)
		}
		tables = append(tables, csvdata.Table{Name: step.file, Rows: rows})
	}
	return tables, nil
}

func (m movementRow) key() string {
	return fmt.Sprintf("%s|%d|%s|%d|%s", m.movType, m.goalID, m.month, m.amountCents, m.description)
}

func applyGoals(tx *sql.Tx, files map[string][][]string, log *slog.Logger) error {
	if rows, ok := files["goal_accounts.csv"]; ok {
		if err := mirrorAccounts(tx, rows, log); err != nil {
			return fmt.Errorf("goal_accounts.csv: %w", err)
		}
	}
	if rows, ok := files["goal_account_goals.csv"]; ok {
		if err := mirrorGoals(tx, rows, log); err != nil {
			return fmt.Errorf("goal_account_goals.csv: %w", err)
		}
	}
	if rows, ok := files["goal_account_movements.csv"]; ok {
		if err := mirrorMovements(tx, rows); err != nil {
			return fmt.Errorf("goal_account_movements.csv: %w", err)
		}
	}
	return nil
}

func mirrorAccounts(q database.Querier, rows [][]string, log *slog.Logger) error {
	keep := make(map[string]bool, len(rows))
	for i, row := range rows {
		if len(row) < 2 {
			return fmt.Errorf("row %d: expected 2 columns", i+2)
		}
		name := strings.TrimSpace(row[0])
		if name == "" {
			return fmt.Errorf("row %d: account name is required", i+2)
		}
		if keep[name] {
			return fmt.Errorf("row %d: account %q is listed more than once", i+2, name)
		}
		keep[name] = true
		deposit, err := csvdata.ParseNonNegativeCents(row[1], "monthly_deposit_cents")
		if err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}
		// The update only fires when the deposit changed, so IDs stay untouched
		// when the CSV matches the database.
		if _, err := q.Exec(
			`INSERT INTO goal_accounts (name, monthly_deposit_cents) VALUES (?, ?)
			 ON CONFLICT(name) DO UPDATE SET monthly_deposit_cents = excluded.monthly_deposit_cents
			 WHERE goal_accounts.monthly_deposit_cents IS NOT excluded.monthly_deposit_cents`,
			name, deposit,
		); err != nil {
			return fmt.Errorf("row %d: upsert account %s: %w", i+2, name, err)
		}
	}

	return csvdata.DeleteUnlisted(log, q, "goal_accounts", "SELECT id, name FROM goal_accounts", keep,
		func(r *sql.Rows) (int64, string, error) {
			var id int64
			var name string
			if err := r.Scan(&id, &name); err != nil {
				return 0, "", err
			}
			return id, name, nil
		})
}

func mirrorGoals(q database.Querier, rows [][]string, log *slog.Logger) error {
	keep := make(map[string]bool, len(rows))
	for i, row := range rows {
		if len(row) < 6 {
			return fmt.Errorf("row %d: expected 6 columns", i+2)
		}
		accountName := strings.TrimSpace(row[0])
		name := strings.TrimSpace(row[1])
		targetCents, err := csvdata.ParseNonNegativeCents(row[2], "target_cents")
		if err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}
		if targetCents <= 0 {
			return fmt.Errorf("row %d: target_cents must be positive", i+2)
		}
		startMonth := strings.TrimSpace(row[3])
		endMonth := strings.TrimSpace(row[4])
		status := strings.TrimSpace(row[5])
		if err := validateGoalSeed(name, startMonth, endMonth, status); err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}

		accountID, err := getAccountIDByName(q, accountName)
		if err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}

		// The update only fires when a column changed, so IDs and updated_at
		// stay untouched when the CSV matches the database.
		if _, err := q.Exec(
			`INSERT INTO goal_account_goals (account_id, name, target_cents, start_month, end_month, status, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT(account_id, name) DO UPDATE SET
			   target_cents = excluded.target_cents,
			   start_month = excluded.start_month,
			   end_month = excluded.end_month,
			   status = excluded.status,
			   updated_at = CURRENT_TIMESTAMP
			 WHERE goal_account_goals.target_cents IS NOT excluded.target_cents
			    OR goal_account_goals.start_month IS NOT excluded.start_month
			    OR goal_account_goals.end_month IS NOT excluded.end_month
			    OR goal_account_goals.status IS NOT excluded.status`,
			accountID, name, targetCents, startMonth, endMonth, status, startMonth+"-01",
		); err != nil {
			return fmt.Errorf("row %d: upsert goal %s: %w", i+2, name, err)
		}
		keep[csvdata.ScopedKey(accountID, name)] = true
	}

	return csvdata.DeleteUnlisted(log, q, "goal_account_goals", "SELECT id, account_id, name FROM goal_account_goals", keep,
		func(r *sql.Rows) (int64, string, error) {
			var id, accountID int64
			var name string
			if err := r.Scan(&id, &accountID, &name); err != nil {
				return 0, "", err
			}
			return id, csvdata.ScopedKey(accountID, name), nil
		})
}

func validateGoalSeed(name, startMonth, endMonth, status string) error {
	if name == "" {
		return errors.New("goal name is required")
	}
	if err := month.Validate(startMonth); err != nil {
		return fmt.Errorf("invalid start_month: %w", err)
	}
	if err := month.Validate(endMonth); err != nil {
		return fmt.Errorf("invalid end_month: %w", err)
	}
	if endMonth < startMonth {
		return errors.New("end_month cannot be before start_month")
	}
	switch status {
	case "active", "completed", "cancelled":
	default:
		return fmt.Errorf("invalid status %q", status)
	}
	return nil
}

func validateMovementSeed(m movementRow) error {
	if err := month.Validate(m.month); err != nil {
		return fmt.Errorf("invalid month: %w", err)
	}
	switch m.movType {
	case "deposit":
		if m.goalID != 0 {
			return errors.New("deposits must not name a goal")
		}
	case "allocation", "expense", "release", "cover":
		if m.goalID == 0 {
			return fmt.Errorf("%s requires a goal", m.movType)
		}
	default:
		return fmt.Errorf("invalid movement type %q", m.movType)
	}
	return nil
}

func mirrorMovements(q database.Querier, rows [][]string) error {
	byAccount := make(map[int64][]movementRow)
	for i, row := range rows {
		if len(row) < 5 {
			return fmt.Errorf("row %d: expected at least 5 columns", i+2)
		}
		amount, err := csvdata.ParseNonNegativeCents(row[4], "amount_cents")
		if err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}
		if amount <= 0 {
			return fmt.Errorf("row %d: amount_cents must be positive", i+2)
		}
		accountID, err := getAccountIDByName(q, strings.TrimSpace(row[0]))
		if err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}
		movement := movementRow{
			accountID:   accountID,
			movType:     canonicalMovementType(strings.TrimSpace(row[1])),
			month:       strings.TrimSpace(row[3]),
			amountCents: amount,
		}
		if goalName := strings.TrimSpace(row[2]); goalName != "" {
			if movement.goalID, err = getGoalID(q, accountID, goalName); err != nil {
				return fmt.Errorf("row %d: %w", i+2, err)
			}
		}
		if len(row) >= 6 {
			movement.description = strings.TrimSpace(row[5])
		}
		if err := validateMovementSeed(movement); err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}
		byAccount[accountID] = append(byAccount[accountID], movement)
	}

	// Every account is reconciled, so accounts with no rows in the CSV end up
	// with no movements.
	accountIDs, err := csvdata.QueryInt64s(q, "SELECT id FROM goal_accounts ORDER BY id")
	if err != nil {
		return fmt.Errorf("list accounts: %w", err)
	}
	for _, accountID := range accountIDs {
		desired := byAccount[accountID]
		keys := make([]string, len(desired))
		for i, m := range desired {
			keys[i] = m.key()
		}

		changed, err := csvdata.ClearIfChanged(q, "goal_account_movements", "account_id", accountID,
			`SELECT type, COALESCE(goal_id, 0), month, amount_cents, COALESCE(description, '')
			 FROM goal_account_movements WHERE account_id = ?`,
			func(r *sql.Rows) (string, error) {
				var m movementRow
				if err := r.Scan(&m.movType, &m.goalID, &m.month, &m.amountCents, &m.description); err != nil {
					return "", err
				}
				return m.key(), nil
			}, keys)
		if err != nil {
			return err
		}
		if !changed {
			continue
		}
		for _, m := range desired {
			if _, err := q.Exec(
				`INSERT INTO goal_account_movements (account_id, goal_id, type, amount_cents, month, description)
				 VALUES (?, NULLIF(?, 0), ?, ?, ?, NULLIF(?, ''))`,
				accountID, m.goalID, m.movType, m.amountCents, m.month, m.description,
			); err != nil {
				return fmt.Errorf("insert %s movement: %w", m.movType, err)
			}
		}
	}
	return nil
}

// canonicalMovementType maps the old name of the closing movement, refund, to
// release so CSV files written before the rename still import.
func canonicalMovementType(t string) string {
	if t == "refund" {
		return MovementRelease
	}
	return t
}

func getAccountIDByName(q database.Querier, accountName string) (int64, error) {
	var id int64
	err := q.QueryRow("SELECT id FROM goal_accounts WHERE name = ?", accountName).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("account %q not found: %w", accountName, err)
	}
	return id, nil
}

func getGoalID(q database.Querier, accountID int64, goalName string) (int64, error) {
	var id int64
	err := q.QueryRow("SELECT id FROM goal_account_goals WHERE account_id = ? AND name = ?", accountID, goalName).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("goal %q not found in account %d: %w", goalName, accountID, err)
	}
	return id, nil
}

func exportAccounts(db *sql.DB) ([][]string, error) {
	rows, err := db.Query("SELECT name, monthly_deposit_cents FROM goal_accounts ORDER BY name ASC")
	if err != nil {
		return nil, fmt.Errorf("query accounts: %w", err)
	}
	defer rows.Close()

	result := [][]string{{"name", "monthly_deposit_cents"}}
	for rows.Next() {
		var name string
		var deposit money.Cents
		if err := rows.Scan(&name, &deposit); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		result = append(result, []string{name, csvdata.FormatCents(deposit)})
	}

	return result, rows.Err()
}

func exportGoals(db *sql.DB) ([][]string, error) {
	rows, err := db.Query(
		`SELECT a.name, g.name, g.target_cents, g.start_month, g.end_month, g.status
		 FROM goal_account_goals g
		 JOIN goal_accounts a ON g.account_id = a.id
		 ORDER BY a.name ASC, g.name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query goals: %w", err)
	}
	defer rows.Close()

	result := [][]string{{"account", "name", "target_cents", "start_month", "end_month", "status"}}
	for rows.Next() {
		var account, name, startMonth, endMonth, status string
		var targetAmount money.Cents
		if err := rows.Scan(&account, &name, &targetAmount, &startMonth, &endMonth, &status); err != nil {
			return nil, fmt.Errorf("scan goal: %w", err)
		}
		result = append(result, []string{account, name, csvdata.FormatCents(targetAmount), startMonth, endMonth, status})
	}

	return result, rows.Err()
}

func exportMovements(db *sql.DB) ([][]string, error) {
	rows, err := db.Query(
		`SELECT a.name, m.type, COALESCE(g.name, ''), m.month, m.amount_cents, COALESCE(m.description, '')
		 FROM goal_account_movements m
		 JOIN goal_accounts a ON m.account_id = a.id
		 LEFT JOIN goal_account_goals g ON m.goal_id = g.id
		 ORDER BY a.name ASC, m.month ASC, m.type ASC, COALESCE(g.name, '') ASC, m.id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query movements: %w", err)
	}
	defer rows.Close()

	result := [][]string{{"account", "type", "goal", "month", "amount_cents", "description"}}
	for rows.Next() {
		var account, movType, goal, month, description string
		var amount money.Cents
		if err := rows.Scan(&account, &movType, &goal, &month, &amount, &description); err != nil {
			return nil, fmt.Errorf("scan movement: %w", err)
		}
		result = append(result, []string{account, movType, goal, month, csvdata.FormatCents(amount), description})
	}

	return result, rows.Err()
}

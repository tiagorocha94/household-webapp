package savings

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/csvdata"
	"github.com/tiagorocha94/household-webapp/internal/csvstate"
	"github.com/tiagorocha94/household-webapp/internal/database"
	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/month"
)

const (
	accountsFile  = "savings_accounts.csv"
	movementsFile = "savings_movements.csv"
	snapshotsFile = "savings_snapshots.csv"

	// moduleName names the module in logs, errors and the csv state.
	moduleName = "savings"
)

var (
	accountsHeader  = []string{"name", "owner", "target_cents", "notes"}
	movementsHeader = []string{"account", "month", "type", "amount_cents", "description", "movement"}
	snapshotsHeader = []string{"account", "month", "value_cents", "description"}
)

// SeedModule imports and exports the savings CSV files and satisfies the seed
// package's Module interface.
type SeedModule struct{}

// movementRow is a movement row of the CSV file.
type movementRow struct {
	month       string
	typ         string
	cents       money.Cents
	description string
	movement    string // sync_id of the cashflow movement it belongs to, or empty
}

// snapshotRow is a snapshot row of the CSV file.
type snapshotRow struct {
	month       string
	cents       money.Cents
	description string
}

// Name returns the module's name for logs and errors.
func (SeedModule) Name() string { return moduleName }

// Files lists the CSV files the module imports, with their headers.
func (SeedModule) Files() []csvdata.File {
	return []csvdata.File{
		{Name: accountsFile, Header: accountsHeader},
		{Name: movementsFile, Header: movementsHeader},
		{Name: snapshotsFile, Header: snapshotsHeader},
	}
}

// Import mirrors the savings CSV files that exist into the database.
func (SeedModule) Import(tx *sql.Tx, files map[string][][]string, log *slog.Logger) error {
	if rows, ok := files[accountsFile]; ok {
		if err := mirrorAccounts(tx, rows, log); err != nil {
			return fmt.Errorf("%s: %w", accountsFile, err)
		}
	}
	// Devices write movements and snapshots. When they changed the database
	// after the files were last written, the files are older and importing them
	// would undo that; the export that follows rewrites them.
	behind, err := csvstate.Behind(tx, moduleName)
	if err != nil {
		return err
	}
	if behind {
		log.Warn("not importing savings movements and snapshots: the database has changes the files do not have yet")
		return nil
	}
	if rows, ok := files[movementsFile]; ok {
		if err := mirrorMovements(tx, rows); err != nil {
			return fmt.Errorf("%s: %w", movementsFile, err)
		}
	}
	if rows, ok := files[snapshotsFile]; ok {
		if err := mirrorSnapshots(tx, rows); err != nil {
			return fmt.Errorf("%s: %w", snapshotsFile, err)
		}
	}
	return nil
}

// AfterExport records that the files now hold everything in the database. The
// seed runner calls it once they are written, at start-up, before a device can
// change anything.
func (SeedModule) AfterExport(db *sql.DB) error {
	return csvstate.MarkWritten(db, moduleName)
}

// Sync does nothing: every figure is derived when it is read.
func (SeedModule) Sync(*sql.DB, time.Time, *slog.Logger) error { return nil }

// Export returns the savings tables in write order.
func (SeedModule) Export(db *sql.DB) ([]csvdata.Table, error) {
	steps := []struct {
		file string
		fn   func(*sql.DB) ([][]string, error)
	}{
		{accountsFile, exportAccounts},
		{movementsFile, exportMovements},
		{snapshotsFile, exportSnapshots},
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

func exportAccounts(db *sql.DB) ([][]string, error) {
	rows, err := db.Query(
		`SELECT a.name, COALESCE(u.username, ''), COALESCE(a.target_cents, 0), COALESCE(a.notes, '')
		 FROM savings_accounts a LEFT JOIN users u ON u.id = a.user_id
		 ORDER BY a.name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query savings accounts for export: %w", err)
	}
	defer rows.Close()

	result := [][]string{accountsHeader}
	for rows.Next() {
		var name, owner, notes string
		var target money.Cents
		if err := rows.Scan(&name, &owner, &target, &notes); err != nil {
			return nil, fmt.Errorf("scan savings account for export: %w", err)
		}
		targetText := ""
		if target > 0 {
			targetText = csvdata.FormatCents(target)
		}
		result = append(result, []string{name, owner, targetText, notes})
	}
	return result, rows.Err()
}

func exportMovements(db *sql.DB) ([][]string, error) {
	rows, err := db.Query(
		`SELECT a.name, m.month, m.type, m.amount_cents, COALESCE(m.description, ''), COALESCE(m.movement_sync_id, '')
		 FROM savings_movements m JOIN savings_accounts a ON a.id = m.account_id
		 ORDER BY a.name ASC, m.month ASC, m.id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query savings movements for export: %w", err)
	}
	defer rows.Close()

	result := [][]string{movementsHeader}
	for rows.Next() {
		var account, monthValue, typ, description, movement string
		var cents money.Cents
		if err := rows.Scan(&account, &monthValue, &typ, &cents, &description, &movement); err != nil {
			return nil, fmt.Errorf("scan savings movement for export: %w", err)
		}
		result = append(result, []string{account, monthValue, typ, csvdata.FormatCents(cents), description, movement})
	}
	return result, rows.Err()
}

func exportSnapshots(db *sql.DB) ([][]string, error) {
	rows, err := db.Query(
		`SELECT a.name, s.month, s.value_cents, COALESCE(s.description, '')
		 FROM savings_snapshots s JOIN savings_accounts a ON a.id = s.account_id
		 ORDER BY a.name ASC, s.month ASC, s.id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query savings snapshots for export: %w", err)
	}
	defer rows.Close()

	result := [][]string{snapshotsHeader}
	for rows.Next() {
		var account, monthValue, description string
		var cents money.Cents
		if err := rows.Scan(&account, &monthValue, &cents, &description); err != nil {
			return nil, fmt.Errorf("scan savings snapshot for export: %w", err)
		}
		result = append(result, []string{account, monthValue, csvdata.FormatCents(cents), description})
	}
	return result, rows.Err()
}

func nameScan(r *sql.Rows) (int64, string, error) {
	var id int64
	var name string
	if err := r.Scan(&id, &name); err != nil {
		return 0, "", err
	}
	return id, name, nil
}

func getAccountID(q database.Querier, name string) (int64, error) {
	var id int64
	if err := q.QueryRow("SELECT id FROM savings_accounts WHERE name = ?", name).Scan(&id); err != nil {
		return 0, fmt.Errorf("savings account %q not found: %w", name, err)
	}
	return id, nil
}

// getOptionalUserID resolves a username from users.csv to its ID. An empty
// username means the household owns the account and gives nil (NULL).
func getOptionalUserID(q database.Querier, username string) (any, error) {
	if username == "" {
		return nil, nil
	}
	var id int64
	if err := q.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&id); err != nil {
		return nil, fmt.Errorf("user %q not found: %w", username, err)
	}
	return id, nil
}

func mirrorAccounts(q database.Querier, rows [][]string, log *slog.Logger) error {
	keep := make(map[string]bool, len(rows))
	for i, row := range rows {
		if len(row) < 2 {
			return fmt.Errorf("row %d: expected at least 2 columns", i+2)
		}
		name, owner := strings.TrimSpace(row[0]), strings.TrimSpace(row[1])
		var targetText, notes string
		if len(row) >= 3 {
			targetText = strings.TrimSpace(row[2])
		}
		if len(row) >= 4 {
			notes = strings.TrimSpace(row[3])
		}
		if name == "" {
			return fmt.Errorf("row %d: name is required", i+2)
		}
		if keep[name] {
			return fmt.Errorf("row %d: account %q is listed more than once", i+2, name)
		}
		keep[name] = true

		// An empty target means the account has none.
		var target any
		if targetText != "" {
			cents, err := csvdata.ParseNonNegativeCents(targetText, "target_cents")
			if err != nil {
				return fmt.Errorf("row %d: %w", i+2, err)
			}
			if cents <= 0 {
				return fmt.Errorf("row %d: target_cents must be positive, or empty for no target", i+2)
			}
			target = int64(cents)
		}
		userID, err := getOptionalUserID(q, owner)
		if err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}

		// The update only fires when a column changed.
		if _, err := q.Exec(
			`INSERT INTO savings_accounts (name, user_id, target_cents, notes)
			 VALUES (?, ?, ?, NULLIF(?, ''))
			 ON CONFLICT(name) DO UPDATE SET
			   user_id = excluded.user_id,
			   target_cents = excluded.target_cents,
			   notes = excluded.notes,
			   updated_at = CURRENT_TIMESTAMP
			 WHERE savings_accounts.user_id IS NOT excluded.user_id
			    OR savings_accounts.target_cents IS NOT excluded.target_cents
			    OR savings_accounts.notes IS NOT excluded.notes`,
			name, userID, target, notes,
		); err != nil {
			return fmt.Errorf("upsert savings account %q: %w", name, err)
		}
	}
	return csvdata.DeleteUnlisted(log, q, "savings_accounts",
		"SELECT id, name FROM savings_accounts", keep, nameScan)
}

func movementKey(r movementRow) string {
	return fmt.Sprintf("%s|%s|%d|%s|%s", r.month, r.typ, r.cents, r.description, r.movement)
}

func snapshotKey(r snapshotRow) string {
	return fmt.Sprintf("%s|%d|%s", r.month, r.cents, r.description)
}

// mirrorMovements reconciles the movements of every account, so an account
// with no rows in the file ends up with none.
func mirrorMovements(q database.Querier, rows [][]string) error {
	byAccount := make(map[int64][]movementRow)
	for i, row := range rows {
		if len(row) < 4 {
			return fmt.Errorf("row %d: expected at least 4 columns", i+2)
		}
		accountID, err := getAccountID(q, strings.TrimSpace(row[0]))
		if err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}
		r := movementRow{month: strings.TrimSpace(row[1]), typ: strings.TrimSpace(row[2])}
		if err := month.Validate(r.month); err != nil {
			return fmt.Errorf("row %d: invalid month: %w", i+2, err)
		}
		if r.typ != TypeDeposit && r.typ != TypeWithdrawal {
			return fmt.Errorf("row %d: invalid type %q, want deposit or withdrawal", i+2, r.typ)
		}
		r.cents, err = csvdata.ParseNonNegativeCents(row[3], "amount_cents")
		if err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}
		if r.cents <= 0 {
			return fmt.Errorf("row %d: amount_cents must be positive", i+2)
		}
		if len(row) >= 5 {
			r.description = strings.TrimSpace(row[4])
		}
		if len(row) >= 6 {
			if r.movement, err = csvdata.ParseMovementID(row[5]); err != nil {
				return fmt.Errorf("row %d: %w", i+2, err)
			}
		}
		byAccount[accountID] = append(byAccount[accountID], r)
	}

	accountIDs, err := csvdata.QueryInt64s(q, "SELECT id FROM savings_accounts ORDER BY id")
	if err != nil {
		return fmt.Errorf("list savings accounts: %w", err)
	}
	for _, accountID := range accountIDs {
		desired := byAccount[accountID]
		keys := make([]string, len(desired))
		for i, r := range desired {
			keys[i] = movementKey(r)
		}
		changed, err := csvdata.ClearIfChanged(q, "savings_movements", "account_id", accountID,
			`SELECT month, type, amount_cents, COALESCE(description, ''), COALESCE(movement_sync_id, '')
			 FROM savings_movements WHERE account_id = ?`,
			func(rs *sql.Rows) (string, error) {
				var r movementRow
				if err := rs.Scan(&r.month, &r.typ, &r.cents, &r.description, &r.movement); err != nil {
					return "", err
				}
				return movementKey(r), nil
			}, keys)
		if err != nil {
			return err
		}
		if !changed {
			continue
		}
		for _, r := range desired {
			if _, err := q.Exec(
				`INSERT INTO savings_movements (account_id, month, type, amount_cents, description, movement_sync_id)
				 VALUES (?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''))`,
				accountID, r.month, r.typ, r.cents, r.description, r.movement,
			); err != nil {
				return fmt.Errorf("insert savings movement of account %d month %s: %w", accountID, r.month, err)
			}
		}
	}
	return nil
}

// mirrorSnapshots reconciles the snapshots of every account.
func mirrorSnapshots(q database.Querier, rows [][]string) error {
	byAccount := make(map[int64][]snapshotRow)
	for i, row := range rows {
		if len(row) < 3 {
			return fmt.Errorf("row %d: expected at least 3 columns", i+2)
		}
		accountID, err := getAccountID(q, strings.TrimSpace(row[0]))
		if err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}
		r := snapshotRow{month: strings.TrimSpace(row[1])}
		if err := month.Validate(r.month); err != nil {
			return fmt.Errorf("row %d: invalid month: %w", i+2, err)
		}
		if r.cents, err = csvdata.ParseNonNegativeCents(row[2], "value_cents"); err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}
		if len(row) >= 4 {
			r.description = strings.TrimSpace(row[3])
		}
		byAccount[accountID] = append(byAccount[accountID], r)
	}

	accountIDs, err := csvdata.QueryInt64s(q, "SELECT id FROM savings_accounts ORDER BY id")
	if err != nil {
		return fmt.Errorf("list savings accounts: %w", err)
	}
	for _, accountID := range accountIDs {
		desired := byAccount[accountID]
		keys := make([]string, len(desired))
		for i, r := range desired {
			keys[i] = snapshotKey(r)
		}
		changed, err := csvdata.ClearIfChanged(q, "savings_snapshots", "account_id", accountID,
			`SELECT month, value_cents, COALESCE(description, '')
			 FROM savings_snapshots WHERE account_id = ?`,
			func(rs *sql.Rows) (string, error) {
				var r snapshotRow
				if err := rs.Scan(&r.month, &r.cents, &r.description); err != nil {
					return "", err
				}
				return snapshotKey(r), nil
			}, keys)
		if err != nil {
			return err
		}
		if !changed {
			continue
		}
		for _, r := range desired {
			if _, err := q.Exec(
				`INSERT INTO savings_snapshots (account_id, month, value_cents, description)
				 VALUES (?, ?, ?, NULLIF(?, ''))`,
				accountID, r.month, r.cents, r.description,
			); err != nil {
				return fmt.Errorf("insert savings snapshot of account %d month %s: %w", accountID, r.month, err)
			}
		}
	}
	return nil
}

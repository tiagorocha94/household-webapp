package investments

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
	entitiesFile      = "investment_entities.csv"
	instrumentsFile   = "investment_instruments.csv"
	contributionsFile = "investment_contributions.csv"
	snapshotsFile     = "investment_snapshots.csv"

	// moduleName names the module in logs, errors and the csv state.
	moduleName = "investments"
)

var (
	entitiesHeader      = []string{"name", "kind", "user"}
	instrumentsHeader   = []string{"entity", "name", "type", "notes"}
	contributionsHeader = []string{"entity", "instrument", "month", "amount_cents", "description", "movement"}
	snapshotsHeader     = []string{"entity", "instrument", "month", "value_cents", "description"}
)

var (
	contributionsTable = monthlyTable{
		file: contributionsFile, table: "investment_contributions",
		amountCol: "amount_cents", amountName: "amount_cents", mustBePositive: true, linked: true,
	}
	snapshotsTable = monthlyTable{
		file: snapshotsFile, table: "investment_snapshots",
		amountCol: "value_cents", amountName: "value_cents",
	}
)

// SeedModule imports and exports the investment CSV files and satisfies the
// seed package's Module interface.
type SeedModule struct{}

// monthlyRow is a contribution or snapshot row: an amount in a month.
type monthlyRow struct {
	instrumentID int64
	month        string
	cents        money.Cents
	description  string
	movement     string // sync_id of the cashflow movement it belongs to, or empty
}

// monthlyTable describes the two child tables, which differ only in names.
type monthlyTable struct {
	file           string
	table          string
	amountCol      string
	amountName     string
	mustBePositive bool
	// linked is true when a row can be the detail of a cashflow movement.
	linked bool
}

// movementExpr is the SQL for a row's movement: its column, or empty for a
// table that has none.
func (t monthlyTable) movementExpr(prefix string) string {
	if t.linked {
		return "COALESCE(" + prefix + "movement_sync_id, '')"
	}
	return "''"
}

// Name returns the module's name for logs and errors.
func (SeedModule) Name() string { return moduleName }

// Files lists the CSV files the module imports, with their headers.
func (SeedModule) Files() []csvdata.File {
	return []csvdata.File{
		{Name: entitiesFile, Header: entitiesHeader},
		{Name: instrumentsFile, Header: instrumentsHeader},
		{Name: contributionsFile, Header: contributionsHeader},
		{Name: snapshotsFile, Header: snapshotsHeader},
	}
}

// Import mirrors the investment CSV files that exist into the database.
func (SeedModule) Import(tx *sql.Tx, files map[string][][]string, log *slog.Logger) error {
	if rows, ok := files[entitiesFile]; ok {
		if err := mirrorEntities(tx, rows, log); err != nil {
			return fmt.Errorf("%s: %w", entitiesFile, err)
		}
	}
	if rows, ok := files[instrumentsFile]; ok {
		if err := mirrorInstruments(tx, rows, log); err != nil {
			return fmt.Errorf("%s: %w", instrumentsFile, err)
		}
	}
	// Devices write contributions and snapshots. When they changed the database
	// after the files were last written, the files are older and importing them
	// would undo that; the export that follows rewrites them.
	behind, err := csvstate.Behind(tx, moduleName)
	if err != nil {
		return err
	}
	if behind {
		log.Warn("not importing investment contributions and snapshots: the database has changes the files do not have yet")
		return nil
	}
	if rows, ok := files[contributionsFile]; ok {
		if err := mirrorMonthly(tx, contributionsTable, rows); err != nil {
			return fmt.Errorf("%s: %w", contributionsFile, err)
		}
	}
	if rows, ok := files[snapshotsFile]; ok {
		if err := mirrorMonthly(tx, snapshotsTable, rows); err != nil {
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

// Sync does nothing: every figure is derived by views, not maintained.
func (SeedModule) Sync(*sql.DB, time.Time, *slog.Logger) error { return nil }

// Export returns the investment tables in write order.
func (SeedModule) Export(db *sql.DB) ([]csvdata.Table, error) {
	steps := []struct {
		file string
		fn   func(*sql.DB) ([][]string, error)
	}{
		{entitiesFile, exportEntities},
		{instrumentsFile, exportInstruments},
		{contributionsFile, func(db *sql.DB) ([][]string, error) {
			return exportMonthly(db, contributionsTable, contributionsHeader)
		}},
		{snapshotsFile, func(db *sql.DB) ([][]string, error) {
			return exportMonthly(db, snapshotsTable, snapshotsHeader)
		}},
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

func exportEntities(db *sql.DB) ([][]string, error) {
	rows, err := db.Query(`SELECT e.name, e.kind, COALESCE(u.username, '')
		 FROM investment_entities e LEFT JOIN users u ON u.id = e.user_id
		 ORDER BY e.name ASC`)
	if err != nil {
		return nil, fmt.Errorf("query investment entities for export: %w", err)
	}
	defer rows.Close()

	result := [][]string{entitiesHeader}
	for rows.Next() {
		var name, kind, user string
		if err := rows.Scan(&name, &kind, &user); err != nil {
			return nil, fmt.Errorf("scan investment entity for export: %w", err)
		}
		result = append(result, []string{name, kind, user})
	}
	return result, rows.Err()
}

func exportInstruments(db *sql.DB) ([][]string, error) {
	rows, err := db.Query(
		`SELECT e.name, i.name, i.type, COALESCE(i.notes, '')
		 FROM investment_instruments i
		 JOIN investment_entities e ON e.id = i.entity_id
		 ORDER BY e.name ASC, i.name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query investment instruments for export: %w", err)
	}
	defer rows.Close()

	result := [][]string{instrumentsHeader}
	for rows.Next() {
		var entity, name, instrumentType, notes string
		if err := rows.Scan(&entity, &name, &instrumentType, &notes); err != nil {
			return nil, fmt.Errorf("scan investment instrument for export: %w", err)
		}
		result = append(result, []string{entity, name, instrumentType, notes})
	}
	return result, rows.Err()
}

func exportMonthly(db *sql.DB, t monthlyTable, header []string) ([][]string, error) {
	rows, err := db.Query(
		`SELECT e.name, i.name, t.month, t.` + t.amountCol + `, COALESCE(t.description, ''), ` + t.movementExpr("t.") + `
		 FROM ` + t.table + ` t
		 JOIN investment_instruments i ON i.id = t.instrument_id
		 JOIN investment_entities e ON e.id = i.entity_id
		 ORDER BY e.name ASC, i.name ASC, t.month ASC, t.id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query %s for export: %w", t.table, err)
	}
	defer rows.Close()

	result := [][]string{header}
	for rows.Next() {
		var entity, instrument, monthValue, description, movement string
		var cents money.Cents
		if err := rows.Scan(&entity, &instrument, &monthValue, &cents, &description, &movement); err != nil {
			return nil, fmt.Errorf("scan %s for export: %w", t.table, err)
		}
		line := []string{entity, instrument, monthValue, csvdata.FormatCents(cents), description}
		if t.linked {
			line = append(line, movement)
		}
		result = append(result, line)
	}
	return result, rows.Err()
}

func validKind(kind string) bool {
	switch kind {
	case KindBank, KindBroker, KindInvestmentFirm, KindState, KindOther:
		return true
	}
	return false
}

func validType(instrumentType string) bool {
	switch instrumentType {
	case TypeETF, TypeFund, TypePPR, TypeCertificadosAforro, TypeOther:
		return true
	}
	return false
}

func nameScan(r *sql.Rows) (int64, string, error) {
	var id int64
	var name string
	if err := r.Scan(&id, &name); err != nil {
		return 0, "", err
	}
	return id, name, nil
}

func mirrorEntities(q database.Querier, rows [][]string, log *slog.Logger) error {
	keep := make(map[string]bool, len(rows))
	for i, row := range rows {
		if len(row) < 2 {
			return fmt.Errorf("row %d: expected at least 2 columns", i+2)
		}
		name, kind := strings.TrimSpace(row[0]), strings.TrimSpace(row[1])
		var user string
		if len(row) >= 3 {
			user = strings.TrimSpace(row[2])
		}
		if name == "" {
			return fmt.Errorf("row %d: name is required", i+2)
		}
		if !validKind(kind) {
			return fmt.Errorf("row %d: invalid kind %q", i+2, kind)
		}
		if keep[name] {
			return fmt.Errorf("row %d: entity %q is listed more than once", i+2, name)
		}
		keep[name] = true

		userID, err := getOptionalUserID(q, user)
		if err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}

		// The update only fires when a column changed.
		if _, err := q.Exec(
			`INSERT INTO investment_entities (name, kind, user_id) VALUES (?, ?, ?)
			 ON CONFLICT(name) DO UPDATE SET
			   kind = excluded.kind,
			   user_id = excluded.user_id,
			   updated_at = CURRENT_TIMESTAMP
			 WHERE investment_entities.kind IS NOT excluded.kind
			    OR investment_entities.user_id IS NOT excluded.user_id`,
			name, kind, userID,
		); err != nil {
			return fmt.Errorf("upsert investment entity %q: %w", name, err)
		}
	}
	return csvdata.DeleteUnlisted(log, q, "investment_entities",
		"SELECT id, name FROM investment_entities", keep, nameScan)
}

func instrumentKey(entity, name string) string {
	return entity + "|" + name
}

func getEntityID(q database.Querier, name string) (int64, error) {
	var id int64
	if err := q.QueryRow("SELECT id FROM investment_entities WHERE name = ?", name).Scan(&id); err != nil {
		return 0, fmt.Errorf("investment entity %q not found: %w", name, err)
	}
	return id, nil
}

func getInstrumentID(q database.Querier, entity, name string) (int64, error) {
	var id int64
	err := q.QueryRow(
		`SELECT i.id FROM investment_instruments i
		 JOIN investment_entities e ON e.id = i.entity_id
		 WHERE e.name = ? AND i.name = ?`, entity, name,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("investment instrument %q of entity %q not found: %w", name, entity, err)
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

func mirrorInstruments(q database.Querier, rows [][]string, log *slog.Logger) error {
	keep := make(map[string]bool, len(rows))
	for i, row := range rows {
		if len(row) < 3 {
			return fmt.Errorf("row %d: expected at least 3 columns", i+2)
		}
		entity, name := strings.TrimSpace(row[0]), strings.TrimSpace(row[1])
		instrumentType := strings.TrimSpace(row[2])
		var notes string
		if len(row) >= 4 {
			notes = strings.TrimSpace(row[3])
		}
		if name == "" {
			return fmt.Errorf("row %d: name is required", i+2)
		}
		if !validType(instrumentType) {
			return fmt.Errorf("row %d: invalid type %q", i+2, instrumentType)
		}
		key := instrumentKey(entity, name)
		if keep[key] {
			return fmt.Errorf("row %d: instrument %q of entity %q is listed more than once", i+2, name, entity)
		}
		keep[key] = true

		entityID, err := getEntityID(q, entity)
		if err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}
		if _, err := q.Exec(
			`INSERT INTO investment_instruments (entity_id, name, type, notes)
			 VALUES (?, ?, ?, NULLIF(?, ''))
			 ON CONFLICT(entity_id, name) DO UPDATE SET
			   type = excluded.type,
			   notes = excluded.notes,
			   updated_at = CURRENT_TIMESTAMP
			 WHERE investment_instruments.type IS NOT excluded.type
			    OR investment_instruments.notes IS NOT excluded.notes`,
			entityID, name, instrumentType, notes,
		); err != nil {
			return fmt.Errorf("upsert investment instrument %q of entity %q: %w", name, entity, err)
		}
	}
	return csvdata.DeleteUnlisted(log, q, "investment_instruments",
		`SELECT i.id, e.name || '|' || i.name FROM investment_instruments i
		 JOIN investment_entities e ON e.id = i.entity_id`, keep, nameScan)
}

func monthlyKey(r monthlyRow) string {
	return fmt.Sprintf("%s|%d|%s|%s", r.month, r.cents, r.description, r.movement)
}

// mirrorMonthly reconciles contributions or snapshots. Every instrument is
// reconciled, so an instrument with no rows in the file ends up with none.
func mirrorMonthly(q database.Querier, t monthlyTable, rows [][]string) error {
	byInstrument := make(map[int64][]monthlyRow)
	for i, row := range rows {
		if len(row) < 4 {
			return fmt.Errorf("row %d: expected at least 4 columns", i+2)
		}
		entity, instrument := strings.TrimSpace(row[0]), strings.TrimSpace(row[1])
		instrumentID, err := getInstrumentID(q, entity, instrument)
		if err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}
		r := monthlyRow{instrumentID: instrumentID, month: strings.TrimSpace(row[2])}
		if err := month.Validate(r.month); err != nil {
			return fmt.Errorf("row %d: invalid month: %w", i+2, err)
		}
		r.cents, err = csvdata.ParseNonNegativeCents(row[3], t.amountName)
		if err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}
		if t.mustBePositive && r.cents <= 0 {
			return fmt.Errorf("row %d: %s must be positive", i+2, t.amountName)
		}
		if len(row) >= 5 {
			r.description = strings.TrimSpace(row[4])
		}
		if t.linked && len(row) >= 6 {
			if r.movement, err = csvdata.ParseMovementID(row[5]); err != nil {
				return fmt.Errorf("row %d: %w", i+2, err)
			}
		}
		byInstrument[instrumentID] = append(byInstrument[instrumentID], r)
	}

	instrumentIDs, err := csvdata.QueryInt64s(q, "SELECT id FROM investment_instruments ORDER BY id")
	if err != nil {
		return fmt.Errorf("list investment instruments: %w", err)
	}
	for _, instrumentID := range instrumentIDs {
		desired := byInstrument[instrumentID]
		keys := make([]string, len(desired))
		for i, r := range desired {
			keys[i] = monthlyKey(r)
		}

		changed, err := csvdata.ClearIfChanged(q, t.table, "instrument_id", instrumentID,
			`SELECT month, `+t.amountCol+`, COALESCE(description, ''), `+t.movementExpr("")+`
			 FROM `+t.table+` WHERE instrument_id = ?`,
			func(r *sql.Rows) (string, error) {
				var row monthlyRow
				if err := r.Scan(&row.month, &row.cents, &row.description, &row.movement); err != nil {
					return "", err
				}
				return monthlyKey(row), nil
			}, keys)
		if err != nil {
			return err
		}
		if !changed {
			continue
		}
		insert := `INSERT INTO ` + t.table + ` (instrument_id, month, ` + t.amountCol + `, description)
			 VALUES (?, ?, ?, NULLIF(?, ''))`
		if t.linked {
			insert = `INSERT INTO ` + t.table + ` (instrument_id, month, ` + t.amountCol + `, description, movement_sync_id)
			 VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''))`
		}
		for _, r := range desired {
			args := []any{r.instrumentID, r.month, r.cents, r.description}
			if t.linked {
				args = append(args, r.movement)
			}
			if _, err := q.Exec(insert, args...); err != nil {
				return fmt.Errorf("insert into %s for investment instrument %d month %s: %w", t.table, r.instrumentID, r.month, err)
			}
		}
	}
	return nil
}

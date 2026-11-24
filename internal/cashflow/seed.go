package cashflow

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/csvdata"
	"github.com/tiagorocha94/household-webapp/internal/database"
	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/month"
)

const (
	groupsFile     = "cashflow_budget_groups.csv"
	categoriesFile = "cashflow_categories.csv"
	movementsFile  = "cashflow_movements.csv"
	balancesFile   = "cashflow_balances.csv"
)

var (
	// iconPattern is a Lucide icon name: lowercase words joined by hyphens.
	iconPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	// colorPattern is a #RRGGBB colour.
	colorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

	groupsHeader     = []string{"name", "share_pct"}
	categoriesHeader = []string{"name", "direction", "icon", "color", "budget_group", "role"}
	movementsHeader  = []string{"user", "category", "date", "amount_cents", "description", "sync_id"}
	balancesHeader   = []string{"user", "month", "balance_cents"}
)

// SeedModule imports and exports the cashflow CSV files and satisfies the
// seed package's Module interface. It must run after the auth module, because
// every movement names a user from users.csv.
type SeedModule struct{}

// movementRow is a movement row of one user, resolved against the database.
type movementRow struct {
	syncID      string
	categoryID  int64
	date        string
	cents       money.Cents
	description string
}

// Name returns the module's name for logs and errors.
func (SeedModule) Name() string { return "cashflow" }

// Files lists the CSV files the module imports, with their headers.
func (SeedModule) Files() []csvdata.File {
	return []csvdata.File{
		{Name: groupsFile, Header: groupsHeader},
		{Name: categoriesFile, Header: categoriesHeader},
		{Name: movementsFile, Header: movementsHeader},
		{Name: balancesFile, Header: balancesHeader},
	}
}

// Import mirrors the cashflow CSV files that exist into the database.
func (SeedModule) Import(tx *sql.Tx, files map[string][][]string, log *slog.Logger) error {
	// The groups come first, because categories name them. A group that left
	// the file is removed only after the categories were updated.
	groupRows, hasGroups := files[groupsFile]
	var keepGroups map[string]bool
	if hasGroups {
		var err error
		if keepGroups, err = mirrorGroups(tx, groupRows); err != nil {
			return fmt.Errorf("%s: %w", groupsFile, err)
		}
	}
	categoryRows, hasCategories := files[categoriesFile]
	if hasCategories {
		if err := mirrorCategories(tx, categoryRows, log); err != nil {
			return fmt.Errorf("%s: %w", categoriesFile, err)
		}
	}
	if hasGroups {
		if err := csvdata.DeleteUnlisted(log, tx, "cashflow_budget_groups",
			"SELECT id, name FROM cashflow_budget_groups", keepGroups, nameScan); err != nil {
			return fmt.Errorf("%s: %w", groupsFile, err)
		}
	}
	if hasGroups || hasCategories {
		if err := checkExpenseGroups(tx); err != nil {
			return fmt.Errorf("%s: %w", categoriesFile, err)
		}
	}
	if rows, ok := files[movementsFile]; ok {
		behind, err := csvBehind(tx)
		if err != nil {
			return fmt.Errorf("%s: %w", movementsFile, err)
		}
		if behind {
			// Devices changed the movements after the file was last written, so
			// the file is older than the database. Importing it would undo those
			// changes; the export that follows rewrites it from the database.
			log.Warn("not importing movements: the database has changes the file does not have yet",
				"file", movementsFile)
			return nil
		}
		if err := mirrorMovements(tx, rows); err != nil {
			return fmt.Errorf("%s: %w", movementsFile, err)
		}
	}
	if rows, ok := files[balancesFile]; ok {
		if err := mirrorBalances(tx, rows, log); err != nil {
			return fmt.Errorf("%s: %w", balancesFile, err)
		}
	}
	return nil
}

// AfterExport records that the movements file now holds everything in the
// database. The seed runner calls it once the exported files are written. It
// runs at start-up, before the server accepts any change, so every change so
// far is in the file.
func (SeedModule) AfterExport(db *sql.DB) error {
	if _, err := db.Exec(
		`INSERT INTO cashflow_csv_state (id, revision, csv_revision) VALUES (1, 0, 0)
		 ON CONFLICT(id) DO UPDATE SET csv_revision = revision`,
	); err != nil {
		return fmt.Errorf("record the movements file as up to date: %w", err)
	}
	return nil
}

// Sync does nothing: every figure is derived by views, not maintained.
func (SeedModule) Sync(*sql.DB, time.Time, *slog.Logger) error { return nil }

// Export returns the cashflow tables in write order. Rows come out in one
// deterministic order so unchanged data exports identically.
func (SeedModule) Export(db *sql.DB) ([]csvdata.Table, error) {
	groups, err := exportGroups(db)
	if err != nil {
		return nil, fmt.Errorf("export %s: %w", groupsFile, err)
	}
	categories, err := exportCategories(db)
	if err != nil {
		return nil, fmt.Errorf("export %s: %w", categoriesFile, err)
	}
	movements, err := exportMovements(db)
	if err != nil {
		return nil, fmt.Errorf("export %s: %w", movementsFile, err)
	}
	balances, err := exportBalances(db)
	if err != nil {
		return nil, fmt.Errorf("export %s: %w", balancesFile, err)
	}
	return []csvdata.Table{
		{Name: groupsFile, Rows: groups},
		{Name: categoriesFile, Rows: categories},
		{Name: movementsFile, Rows: movements},
		{Name: balancesFile, Rows: balances},
	}, nil
}

// exportGroups writes the groups in the order of the file they came from.
func exportGroups(db database.Querier) ([][]string, error) {
	rows, err := db.Query("SELECT name, share_pct FROM cashflow_budget_groups ORDER BY position ASC")
	if err != nil {
		return nil, fmt.Errorf("query cashflow budget groups for export: %w", err)
	}
	defer rows.Close()

	result := [][]string{groupsHeader}
	for rows.Next() {
		var name string
		var share int
		if err := rows.Scan(&name, &share); err != nil {
			return nil, fmt.Errorf("scan cashflow budget group for export: %w", err)
		}
		result = append(result, []string{name, strconv.Itoa(share)})
	}
	return result, rows.Err()
}

func exportCategories(db database.Querier) ([][]string, error) {
	rows, err := db.Query(
		`SELECT c.name, c.direction, c.icon, c.color, COALESCE(g.name, ''), COALESCE(c.role, '')
		 FROM cashflow_categories c
		 LEFT JOIN cashflow_budget_groups g ON g.id = c.budget_group_id
		 ORDER BY c.name ASC`)
	if err != nil {
		return nil, fmt.Errorf("query cashflow categories for export: %w", err)
	}
	defer rows.Close()

	result := [][]string{categoriesHeader}
	for rows.Next() {
		var name, direction, icon, color, group, role string
		if err := rows.Scan(&name, &direction, &icon, &color, &group, &role); err != nil {
			return nil, fmt.Errorf("scan cashflow category for export: %w", err)
		}
		result = append(result, []string{name, direction, icon, color, group, role})
	}
	return result, rows.Err()
}

func exportMovements(db database.Querier) ([][]string, error) {
	rows, err := db.Query(
		`SELECT u.username, c.name, m.date, m.amount_cents, COALESCE(m.description, ''), m.sync_id
		 FROM cashflow_movements m
		 JOIN users u ON u.id = m.user_id
		 JOIN cashflow_categories c ON c.id = m.category_id
		 ORDER BY m.date ASC, u.username ASC, c.name ASC, m.id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query cashflow movements for export: %w", err)
	}
	defer rows.Close()

	result := [][]string{movementsHeader}
	for rows.Next() {
		var user, category, date, description, syncID string
		var cents money.Cents
		if err := rows.Scan(&user, &category, &date, &cents, &description, &syncID); err != nil {
			return nil, fmt.Errorf("scan cashflow movement for export: %w", err)
		}
		result = append(result, []string{user, category, date, csvdata.FormatCents(cents), description, syncID})
	}
	return result, rows.Err()
}

func validDirection(direction string) bool {
	return direction == DirectionIncome || direction == DirectionExpense
}

func nameScan(r *sql.Rows) (int64, string, error) {
	var id int64
	var name string
	if err := r.Scan(&id, &name); err != nil {
		return 0, "", err
	}
	return id, name, nil
}

// mirrorGroups makes the budget groups table match the file, without removing
// the groups that left it (the caller does, once the categories no longer name
// them). It returns the names in the file. Each share is a whole percent and
// together they are exactly 100; the order of the file is kept.
func mirrorGroups(q database.Querier, rows [][]string) (map[string]bool, error) {
	keep := make(map[string]bool, len(rows))
	total := 0
	for i, row := range rows {
		if len(row) < 2 {
			return nil, fmt.Errorf("row %d: expected 2 columns", i+2)
		}
		name := strings.TrimSpace(row[0])
		if name == "" {
			return nil, fmt.Errorf("row %d: name is required", i+2)
		}
		if keep[name] {
			return nil, fmt.Errorf("row %d: budget group %q is listed more than once", i+2, name)
		}
		share, err := strconv.Atoi(strings.TrimSpace(row[1]))
		if err != nil || share < 1 || share > 100 {
			return nil, fmt.Errorf("row %d: share_pct %q must be a whole number from 1 to 100", i+2, strings.TrimSpace(row[1]))
		}
		keep[name] = true
		total += share

		// The update only fires when something changed.
		if _, err := q.Exec(
			`INSERT INTO cashflow_budget_groups (name, share_pct, position) VALUES (?, ?, ?)
			 ON CONFLICT(name) DO UPDATE SET
			   share_pct = excluded.share_pct,
			   position = excluded.position,
			   updated_at = CURRENT_TIMESTAMP
			 WHERE cashflow_budget_groups.share_pct IS NOT excluded.share_pct
			    OR cashflow_budget_groups.position IS NOT excluded.position`,
			name, share, i+1,
		); err != nil {
			return nil, fmt.Errorf("upsert cashflow budget group %q: %w", name, err)
		}
	}
	if len(rows) > 0 && total != 100 {
		return nil, fmt.Errorf("the shares add up to %d, want exactly 100", total)
	}
	return keep, nil
}

// checkExpenseGroups reports an error when budget groups exist but an expense
// category names none, because its spending would belong to no budget.
func checkExpenseGroups(q database.Querier) error {
	var groups int
	if err := q.QueryRow("SELECT COUNT(*) FROM cashflow_budget_groups").Scan(&groups); err != nil {
		return fmt.Errorf("count cashflow budget groups: %w", err)
	}
	if groups == 0 {
		return nil
	}
	rows, err := q.Query(
		`SELECT name FROM cashflow_categories
		 WHERE direction = 'expense' AND budget_group_id IS NULL ORDER BY name ASC`)
	if err != nil {
		return fmt.Errorf("list expense categories without a budget group: %w", err)
	}
	defer rows.Close()

	var missing []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("scan expense category: %w", err)
		}
		missing = append(missing, name)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(missing) > 0 {
		return fmt.Errorf("expense categories without a budget_group while budget groups exist: %s", strings.Join(missing, ", "))
	}
	return nil
}

// mirrorCategories makes the categories table match the file. A category that
// is not in the file is removed together with its movements, so the file
// stays the one source of truth.
func mirrorCategories(q database.Querier, rows [][]string, log *slog.Logger) error {
	groupIDs, err := loadIDs(q, "SELECT id, name FROM cashflow_budget_groups")
	if err != nil {
		return fmt.Errorf("list cashflow budget groups: %w", err)
	}
	keep := make(map[string]bool, len(rows))
	for i, row := range rows {
		if len(row) < 5 {
			return fmt.Errorf("row %d: expected 6 columns", i+2)
		}
		name, direction := strings.TrimSpace(row[0]), strings.TrimSpace(row[1])
		icon, color := strings.TrimSpace(row[2]), strings.TrimSpace(row[3])
		var groupID any
		if group := strings.TrimSpace(row[4]); group != "" {
			id, ok := groupIDs[group]
			if !ok {
				return fmt.Errorf("row %d: unknown budget_group %q", i+2, group)
			}
			if direction == DirectionIncome {
				return fmt.Errorf("row %d: income category %q cannot have a budget_group", i+2, name)
			}
			groupID = id
		}
		role, err := parseRole(row[5], direction)
		if err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}
		if name == "" {
			return fmt.Errorf("row %d: name is required", i+2)
		}
		if !validDirection(direction) {
			return fmt.Errorf("row %d: invalid direction %q", i+2, direction)
		}
		if !iconPattern.MatchString(icon) {
			return fmt.Errorf("row %d: invalid icon %q (a lowercase Lucide name such as shopping-basket)", i+2, icon)
		}
		if !colorPattern.MatchString(color) {
			return fmt.Errorf("row %d: invalid color %q (use #RRGGBB)", i+2, color)
		}
		if keep[name] {
			return fmt.Errorf("row %d: category %q is listed more than once", i+2, name)
		}
		keep[name] = true

		// The update only fires when something changed.
		if _, err := q.Exec(
			`INSERT INTO cashflow_categories (name, direction, icon, color, budget_group_id, role) VALUES (?, ?, ?, ?, ?, NULLIF(?, ''))
			 ON CONFLICT(name) DO UPDATE SET
			   direction = excluded.direction,
			   icon = excluded.icon,
			   color = excluded.color,
			   budget_group_id = excluded.budget_group_id,
			   role = excluded.role,
			   updated_at = CURRENT_TIMESTAMP
			 WHERE cashflow_categories.direction IS NOT excluded.direction
			    OR cashflow_categories.icon IS NOT excluded.icon
			    OR cashflow_categories.color IS NOT excluded.color
			    OR cashflow_categories.budget_group_id IS NOT excluded.budget_group_id
			    OR cashflow_categories.role IS NOT excluded.role`,
			name, direction, icon, color, groupID, role,
		); err != nil {
			return fmt.Errorf("upsert cashflow category %q: %w", name, err)
		}
	}
	return csvdata.DeleteUnlisted(log, q, "cashflow_categories",
		"SELECT id, name FROM cashflow_categories", keep, nameScan)
}

// parseRole reads the role column: empty for none, or savings, investments or
// goals. Only an expense category can have a role.
func parseRole(value, direction string) (string, error) {
	role := strings.ToLower(strings.TrimSpace(value))
	switch role {
	case "":
		return "", nil
	case RoleSavings, RoleInvestments, RoleGoals, RoleCredits:
		if direction != DirectionExpense {
			return "", errors.New("only an expense category can have a role")
		}
		return role, nil
	}
	return "", fmt.Errorf("invalid role %q: expected savings, investments, goals, credits or empty", value)
}

// loadIDs returns the IDs of a table keyed by name; query selects (id, name).
func loadIDs(q database.Querier, query string) (map[string]int64, error) {
	rows, err := q.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make(map[string]int64)
	for rows.Next() {
		id, name, err := nameScan(rows)
		if err != nil {
			return nil, err
		}
		ids[name] = id
	}
	return ids, rows.Err()
}

func movementKey(r movementRow) string {
	return fmt.Sprintf("%s|%s|%d|%d|%s", r.syncID, r.date, r.categoryID, r.cents, r.description)
}

// mirrorMovements reconciles the movements of every user with the file. Rows
// may repeat exactly and every copy counts. A user's rows are replaced only
// when they differ from the database, and a user with no rows in the file ends
// up with no movements.
func mirrorMovements(q database.Querier, rows [][]string) error {
	userIDs, err := loadIDs(q, "SELECT id, username FROM users")
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}
	categoryIDs, err := loadIDs(q, "SELECT id, name FROM cashflow_categories")
	if err != nil {
		return fmt.Errorf("list cashflow categories: %w", err)
	}

	byUser := make(map[int64][]movementRow)
	seenSyncIDs := make(map[string]int)
	for i, row := range rows {
		if len(row) < 6 {
			return fmt.Errorf("row %d: expected 6 columns", i+2)
		}
		user, category := strings.TrimSpace(row[0]), strings.TrimSpace(row[1])
		userID, ok := userIDs[user]
		if !ok {
			return fmt.Errorf("row %d: user %q not found", i+2, user)
		}
		categoryID, ok := categoryIDs[category]
		if !ok {
			return fmt.Errorf("row %d: cashflow category %q not found", i+2, category)
		}
		r := movementRow{categoryID: categoryID, date: strings.TrimSpace(row[2]), syncID: strings.TrimSpace(row[5])}
		if !ValidSyncID(r.syncID) {
			return fmt.Errorf("row %d: invalid sync_id %q (a lowercase UUID)", i+2, r.syncID)
		}
		if first, dup := seenSyncIDs[r.syncID]; dup {
			return fmt.Errorf("row %d: sync_id %q is already used by row %d", i+2, r.syncID, first)
		}
		seenSyncIDs[r.syncID] = i + 2
		if err := month.ValidateDay(r.date); err != nil {
			return fmt.Errorf("row %d: invalid date: %w", i+2, err)
		}
		r.cents, err = csvdata.ParseNonNegativeCents(row[3], "amount_cents")
		if err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}
		if r.cents <= 0 {
			return fmt.Errorf("row %d: amount_cents must be positive", i+2)
		}
		r.description = strings.TrimSpace(row[4])
		byUser[userID] = append(byUser[userID], r)
	}

	// Users are reconciled in ID order so the result is deterministic. A user's
	// rows are replaced only when they differ. All the clearing happens before
	// any insert, so a sync_id that the file moves from one person to another is
	// free again by the time it is inserted.
	var changed []int64
	for _, userID := range slices.Sorted(maps.Values(userIDs)) {
		desired := byUser[userID]
		keys := make([]string, len(desired))
		for i, r := range desired {
			keys[i] = movementKey(r)
		}

		replaced, err := csvdata.ClearIfChanged(q, "cashflow_movements", "user_id", userID,
			`SELECT sync_id, date, category_id, amount_cents, COALESCE(description, '')
			 FROM cashflow_movements WHERE user_id = ?`,
			func(r *sql.Rows) (string, error) {
				var row movementRow
				if err := r.Scan(&row.syncID, &row.date, &row.categoryID, &row.cents, &row.description); err != nil {
					return "", err
				}
				return movementKey(row), nil
			}, keys)
		if err != nil {
			return err
		}
		if replaced {
			changed = append(changed, userID)
		}
	}
	for _, userID := range changed {
		for _, r := range byUser[userID] {
			if _, err := q.Exec(
				`INSERT INTO cashflow_movements (sync_id, user_id, category_id, date, amount_cents, description)
				 VALUES (?, ?, ?, ?, ?, NULLIF(?, ''))`,
				r.syncID, userID, r.categoryID, r.date, r.cents, r.description,
			); err != nil {
				return fmt.Errorf("insert cashflow movement for user %d on %s: %w", userID, r.date, err)
			}
		}
	}
	return nil
}

// csvBehind reports whether devices changed the movements after the movements
// file was last written, which makes the file older than the database. A
// database that has never been changed that way is not behind.
func csvBehind(q database.Querier) (bool, error) {
	var behind bool
	err := q.QueryRow(
		`SELECT COALESCE((SELECT revision > csv_revision FROM cashflow_csv_state WHERE id = 1), 0)`,
	).Scan(&behind)
	if err != nil {
		return false, fmt.Errorf("read whether the movements file is behind: %w", err)
	}
	return behind, nil
}

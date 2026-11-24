package seed

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"

	"github.com/tiagorocha94/household-webapp/internal/database"
)

// ignoredColumns are generated, so they say nothing about the data itself.
var ignoredColumns = map[string]bool{"id": true, "created_at": true, "updated_at": true}

// Snapshot is the content of every data table, one line of text per row, for
// comparing the database before and after the seed flow. IDs and timestamps
// are left out, so rows are compared by what they hold.
type Snapshot map[string][]string

// TableChange lists the rows one table gains and loses between two snapshots.
// A row that changed appears in both lists.
type TableChange struct {
	Table   string
	Added   []string
	Removed []string
}

// Diff returns the tables whose rows differ between s and after, by table name.
func (s Snapshot) Diff(after Snapshot) []TableChange {
	names := make([]string, 0, len(after))
	for name := range after {
		names = append(names, name)
	}
	for name := range s {
		if _, ok := after[name]; !ok {
			names = append(names, name)
		}
	}
	slices.Sort(names)

	var changes []TableChange
	for _, name := range names {
		added, removed := diffRows(s[name], after[name])
		if len(added)+len(removed) > 0 {
			changes = append(changes, TableChange{Table: name, Added: added, Removed: removed})
		}
	}
	return changes
}

// Capture reads every data table (not the migration bookkeeping). Password
// hashes are shown as a short fingerprint, enough to see that one changed.
func Capture(q database.Querier) (Snapshot, error) {
	names, err := tableNames(q)
	if err != nil {
		return nil, err
	}
	snapshot := make(Snapshot, len(names))
	for _, name := range names {
		rows, err := captureTable(q, name)
		if err != nil {
			return nil, fmt.Errorf("capture %s: %w", name, err)
		}
		snapshot[name] = rows
	}
	return snapshot, nil
}

func tableNames(q database.Querier) ([]string, error) {
	rows, err := q.Query(
		`SELECT name FROM sqlite_master
		 WHERE type = 'table' AND name NOT LIKE 'sqlite_%' AND name != 'schema_migrations'
		 ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan table name: %w", err)
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

func captureTable(q database.Querier, table string) ([]string, error) {
	columns, err := tableColumns(q, table)
	if err != nil {
		return nil, err
	}
	quoted := make([]string, len(columns))
	for i, c := range columns {
		quoted[i] = quoteIdentifier(c)
	}

	rows, err := q.Query("SELECT " + strings.Join(quoted, ", ") + " FROM " + quoteIdentifier(table))
	if err != nil {
		return nil, fmt.Errorf("select rows: %w", err)
	}
	defer rows.Close()

	var lines []string
	values := make([]any, len(columns))
	targets := make([]any, len(columns))
	for i := range values {
		targets[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(targets...); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		parts := make([]string, len(columns))
		for i, c := range columns {
			parts[i] = renderValue(c, values[i])
		}
		lines = append(lines, strings.Join(parts, " | "))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	slices.Sort(lines)
	return lines, nil
}

// tableColumns returns the columns of table that describe its data.
func tableColumns(q database.Querier, table string) ([]string, error) {
	rows, err := q.Query("PRAGMA table_info(" + quoteIdentifier(table) + ")")
	if err != nil {
		return nil, fmt.Errorf("list columns: %w", err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var (
			cid        int
			name, kind string
			notNull    int
			defaultVal any
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &kind, &notNull, &defaultVal, &primaryKey); err != nil {
			return nil, fmt.Errorf("scan column: %w", err)
		}
		if !ignoredColumns[name] {
			columns = append(columns, name)
		}
	}
	return columns, rows.Err()
}

func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func renderValue(column string, value any) string {
	var text string
	switch v := value.(type) {
	case nil:
		text = ""
	case []byte:
		text = string(v)
	default:
		text = fmt.Sprint(v)
	}
	if column == "password_hash" {
		sum := sha256.Sum256([]byte(text))
		return "#" + hex.EncodeToString(sum[:4])
	}
	return text
}

// diffRows compares two sorted row lists as multisets.
func diffRows(before, after []string) (added, removed []string) {
	left := make(map[string]int, len(before))
	for _, row := range before {
		left[row]++
	}
	for _, row := range after {
		if left[row] > 0 {
			left[row]--
			continue
		}
		added = append(added, row)
	}
	for _, row := range before {
		if left[row] > 0 {
			left[row]--
			removed = append(removed, row)
		}
	}
	return added, removed
}

package csvdata

import (
	"database/sql"
	"fmt"
	"log/slog"
	"maps"

	"github.com/tiagorocha94/household-webapp/internal/database"
)

// QueryInt64s runs query and returns its single integer column.
func QueryInt64s(q database.Querier, query string) ([]int64, error) {
	rows, err := q.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// DeleteUnlisted deletes rows of table whose key is not in keep and reports
// the removal to log. selectQuery must return rows that keyScan turns into
// (id, key).
func DeleteUnlisted(log *slog.Logger, q database.Querier, table, selectQuery string, keep map[string]bool, keyScan func(*sql.Rows) (int64, string, error)) error {
	rows, err := q.Query(selectQuery)
	if err != nil {
		return fmt.Errorf("list %s: %w", table, err)
	}
	defer rows.Close()

	var stale []int64
	for rows.Next() {
		id, key, err := keyScan(rows)
		if err != nil {
			return fmt.Errorf("scan %s: %w", table, err)
		}
		if !keep[key] {
			stale = append(stale, id)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	for _, id := range stale {
		if _, err := q.Exec("DELETE FROM "+table+" WHERE id = ?", id); err != nil {
			return fmt.Errorf("delete %s %d: %w", table, id, err)
		}
	}
	if len(stale) > 0 {
		log.Info("removed rows missing from csv", "table", table, "rows", len(stale))
	}
	return nil
}

// ClearIfChanged compares the rows of table owned by parentID, selected by
// query (which takes the parent ID), with the desired keys, ignoring order. A
// row's key is what scan returns for it. When they differ it deletes the
// parent's rows and returns true, so the caller inserts the desired rows; when
// they match it changes nothing and returns false.
func ClearIfChanged(q database.Querier, table, parentColumn string, parentID int64, query string, scan func(*sql.Rows) (string, error), desired []string) (bool, error) {
	rows, err := q.Query(query, parentID)
	if err != nil {
		return false, fmt.Errorf("read %s of %s %d: %w", table, parentColumn, parentID, err)
	}
	defer rows.Close()

	existing := make(map[string]int)
	for rows.Next() {
		key, err := scan(rows)
		if err != nil {
			return false, fmt.Errorf("scan %s: %w", table, err)
		}
		existing[key]++
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	rows.Close()

	want := make(map[string]int, len(desired))
	for _, k := range desired {
		want[k]++
	}
	if maps.Equal(existing, want) {
		return false, nil
	}
	if _, err := q.Exec("DELETE FROM "+table+" WHERE "+parentColumn+" = ?", parentID); err != nil {
		return false, fmt.Errorf("clear %s of %s %d: %w", table, parentColumn, parentID, err)
	}
	return true, nil
}

// ScopedKey joins a parent ID and a name into one map key, for names that are
// only unique within their parent.
func ScopedKey(parentID int64, name string) string {
	return fmt.Sprintf("%d\x00%s", parentID, name)
}

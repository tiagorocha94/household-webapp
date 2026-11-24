package goals_test

import (
	"database/sql"
	"reflect"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/auth"
	"github.com/tiagorocha94/household-webapp/internal/csvdata"
	"github.com/tiagorocha94/household-webapp/internal/goals"
	"github.com/tiagorocha94/household-webapp/internal/seed"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

// newRunner runs the goals module after auth, which imports the users it refers to.
func newRunner() *seed.Runner {
	return seed.NewRunner(testutil.Logger(), auth.SeedModule{}, goals.SeedModule{})
}

func writeTestCSV(t *testing.T, path string, rows [][]string) {
	t.Helper()
	if err := csvdata.Write(path, rows); err != nil {
		t.Fatalf("write test CSV %s: %v", path, err)
	}
}

func queryString(t *testing.T, db *sql.DB, query string) string {
	t.Helper()
	var s string
	if err := db.QueryRow(query).Scan(&s); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return s
}

func assertCSVData(t *testing.T, path string, header []string, want [][]string) {
	t.Helper()
	got, err := csvdata.Read(path, header)
	if err != nil {
		t.Fatalf("csvdata.Read(%s) error = %v", path, err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("csvdata.Read(%s) = %#v, want %#v", path, got, want)
	}
}

package server

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

func dig(v any, keys ...string) any {
	for _, k := range keys {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[k]
	}
	return v
}

// checkShape compares a decoded JSON value with its schema and fails the test
// for each difference; see shapeProblems.
func checkShape(t *testing.T, spec map[string]any, schema, value any, where string) {
	t.Helper()
	for _, problem := range shapeProblems(spec, schema, value, where) {
		t.Error(problem)
	}
}

// shapeProblems compares the JSON value with the schema, recursively, and lists
// what differs: property names, null where the schema is not nullable, the JSON
// type of each value, and enum values. A schema that is not an object, or with
// no properties, is not checked further.
func shapeProblems(spec map[string]any, schema, value any, where string) []string {
	s, ok := schema.(map[string]any)
	if !ok {
		return nil
	}
	if ref, ok := s["$ref"].(string); ok {
		name := strings.TrimPrefix(ref, "#/components/schemas/")
		return shapeProblems(spec, dig(spec, "components", "schemas", name), value, where)
	}
	if value == nil {
		if nullable, _ := s["nullable"].(bool); !nullable {
			return []string{fmt.Sprintf("%s: is null but the schema is not nullable", where)}
		}
		return nil
	}

	switch s["type"] {
	case "array":
		items, ok := value.([]any)
		if !ok {
			return []string{fmt.Sprintf("%s: want array, got %T", where, value)}
		}
		if len(items) == 0 {
			return []string{fmt.Sprintf("%s: empty array, the fixture needs an item to check", where)}
		}
		return shapeProblems(spec, s["items"], items[0], where+"[]")
	case "object":
		return objectProblems(spec, s, value, where)
	case "string", "integer", "number", "boolean":
		return scalarProblems(s, value, where)
	}
	return nil
}

func objectProblems(spec map[string]any, s map[string]any, value any, where string) []string {
	props, _ := s["properties"].(map[string]any)
	if props == nil {
		return nil
	}
	obj, ok := value.(map[string]any)
	if !ok {
		return []string{fmt.Sprintf("%s: want object, got %T", where, value)}
	}
	var problems []string
	for k, sub := range props {
		v, present := obj[k]
		if !present {
			problems = append(problems, fmt.Sprintf("%s: %q is documented but not returned", where, k))
			continue
		}
		problems = append(problems, shapeProblems(spec, sub, v, where+"."+k)...)
	}
	for k := range obj {
		if _, documented := props[k]; !documented {
			problems = append(problems, fmt.Sprintf("%s: %q is returned but not documented", where, k))
		}
	}
	return problems
}

func scalarProblems(s map[string]any, value any, where string) []string {
	var ok bool
	switch s["type"] {
	case "string":
		_, ok = value.(string)
	case "boolean":
		_, ok = value.(bool)
	case "number":
		ok = isJSONNumber(value)
	case "integer":
		ok = isJSONNumber(value) && isWhole(value)
	}
	if !ok {
		return []string{fmt.Sprintf("%s: want %v, got %v (%T)", where, s["type"], value, value)}
	}
	if enum, has := s["enum"].([]any); has && !slices.Contains(enum, value) {
		return []string{fmt.Sprintf("%s: %v is not one of %v", where, value, enum)}
	}
	return nil
}

func isJSONNumber(v any) bool {
	switch v.(type) {
	case float64, json.Number:
		return true
	}
	return false
}

func isWhole(v any) bool {
	switch n := v.(type) {
	case float64:
		return n == math.Trunc(n)
	case json.Number:
		_, err := n.Int64()
		return err == nil
	}
	return false
}

// TestShapeProblemsFindsTypeNullAndEnumMismatches checks the checker: a loose
// contract test that cannot fail proves nothing.
func TestShapeProblemsFindsTypeNullAndEnumMismatches(t *testing.T) {
	t.Parallel()

	var spec map[string]any
	if err := yaml.Unmarshal([]byte(`
components:
  schemas:
    Row:
      type: object
      properties:
        name: { type: string }
        count: { type: integer }
        kind: { type: string, enum: [a, b] }
        note: { type: string, nullable: true }
        tags: { type: array, items: { type: string } }
`), &spec); err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	schema := map[string]any{"$ref": "#/components/schemas/Row"}
	good := map[string]any{"name": "x", "count": float64(2), "kind": "a", "note": nil, "tags": []any{"t"}}

	if got := shapeProblems(spec, schema, good, "row"); len(got) != 0 {
		t.Fatalf("a matching value has problems: %v", got)
	}
	tests := map[string]struct {
		field string
		value any
		want  string
	}{
		"wrong type":       {"name", float64(3), "want string"},
		"fraction":         {"count", 2.5, "want integer"},
		"unknown enum":     {"kind", "c", "is not one of"},
		"null not allowed": {"name", nil, "not nullable"},
		"null array":       {"tags", nil, "not nullable"},
		"empty array":      {"tags", []any{}, "empty array"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			value := map[string]any{}
			for k, v := range good {
				value[k] = v
			}
			value[tt.field] = tt.value
			got := strings.Join(shapeProblems(spec, schema, value, "row"), "; ")
			if !strings.Contains(got, tt.want) {
				t.Errorf("problems = %q, want one containing %q", got, tt.want)
			}
		})
	}
}

// TestOpenAPIVersionMatchesTheLatestRelease keeps the spec's version in step
// with the newest released version in CHANGELOG.md.
func TestOpenAPIVersionMatchesTheLatestRelease(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../../docs/architecture/openapi.yaml")
	if err != nil {
		t.Fatalf("read openapi.yaml: %v", err)
	}
	var spec map[string]any
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("parse openapi.yaml: %v", err)
	}
	changelog, err := os.ReadFile("../../CHANGELOG.md")
	if err != nil {
		t.Fatalf("read CHANGELOG.md: %v", err)
	}
	m := regexp.MustCompile(`(?m)^## \[v([0-9][^\]]*)\]`).FindSubmatch(changelog)
	if m == nil {
		t.Fatal("CHANGELOG.md has no released version heading")
	}
	if got := dig(spec, "info", "version"); got != string(m[1]) {
		t.Errorf("openapi.yaml version = %v, want %s (latest in CHANGELOG.md)", got, m[1])
	}
}

// TestOpenAPIMatchesResponses keeps docs/architecture/openapi.yaml in sync with the
// handlers. Adding a GET endpoint to the spec requires adding it to paths.
func TestOpenAPIMatchesResponses(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../../docs/architecture/openapi.yaml")
	if err != nil {
		t.Fatalf("read openapi.yaml: %v", err)
	}
	var spec map[string]any
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("parse openapi.yaml: %v", err)
	}

	srv, db := newTestServer(t)
	testerID := insertTestUser(t, db, "tester", "tester-password")
	cookie := loginTestUser(t, srv, "tester", "tester-password")

	accountID := testutil.MustExec(t, db, "INSERT INTO goal_accounts (name, monthly_deposit_cents) VALUES ('Household', 36600)")
	goalID := testutil.MustExec(t, db,
		`INSERT INTO goal_account_goals (account_id, name, target_cents, start_month, end_month, status)
		 VALUES (?, 'Trip', 120000, '2026-01', '2026-12', 'active')`, accountID)
	testutil.MustExec(t, db, `INSERT INTO goal_account_movements (account_id, goal_id, type, amount_cents, month) VALUES (?, NULL, 'deposit', 50000, '2026-01')`, accountID)
	testutil.MustExec(t, db, `INSERT INTO goal_account_movements (account_id, goal_id, type, amount_cents, month) VALUES (?, ?, 'allocation', 10000, '2026-01')`, accountID, goalID)

	contractID := testutil.MustExec(t, db,
		`INSERT INTO credit_contracts (name, category, contract_number, lender, original_principal_cents, start_month, expected_day)
		 VALUES ('Car', 'car', 'C-1', 'Bank', 100000, '2026-01', 5)`)
	testutil.MustExec(t, db,
		`INSERT INTO credit_payments (contract_id, month, type, principal_cents, interest_cents)
		 VALUES (?, '2026-01', 'regular', 25000, 1000)`, contractID)

	entityID := testutil.MustExec(t, db, "INSERT INTO investment_entities (name, kind) VALUES ('Broker', 'broker')")
	instrumentID := testutil.MustExec(t, db,
		`INSERT INTO investment_instruments (entity_id, name, type) VALUES (?, 'World ETF', 'etf')`, entityID)
	testutil.MustExec(t, db,
		`INSERT INTO investment_contributions (instrument_id, month, amount_cents) VALUES (?, '2026-01', 10000)`, instrumentID)
	testutil.MustExec(t, db,
		`INSERT INTO investment_snapshots (instrument_id, month, value_cents) VALUES (?, '2026-01', 10500)`, instrumentID)

	savingsID := testutil.MustExec(t, db, "INSERT INTO savings_accounts (name, user_id, target_cents) VALUES ('Emergency fund', ?, 500000)", testerID)
	testutil.MustExec(t, db, `INSERT INTO savings_movements (account_id, month, type, amount_cents, description) VALUES (?, '2026-01', 'deposit', 300000, 'Opening')`, savingsID)
	testutil.MustExec(t, db, `INSERT INTO savings_movements (account_id, month, type, amount_cents) VALUES (?, '2026-02', 'withdrawal', 50000)`, savingsID)
	testutil.MustExec(t, db, `INSERT INTO savings_snapshots (account_id, month, value_cents) VALUES (?, '2026-03', 255000)`, savingsID)

	groupID := testutil.MustExec(t, db, "INSERT INTO cashflow_budget_groups (name, share_pct, position) VALUES ('Essential', 100, 1)")
	salaryID := testutil.MustExec(t, db, "INSERT INTO cashflow_categories (name, direction, icon, color) VALUES ('salary', 'income', 'briefcase', '#16A34A')")
	foodID := testutil.MustExec(t, db, "INSERT INTO cashflow_categories (name, direction, icon, color, budget_group_id) VALUES ('food', 'expense', 'utensils', '#E76F51', ?)", groupID)
	testutil.MustExec(t, db,
		`INSERT INTO cashflow_movements (sync_id, user_id, category_id, date, amount_cents, description) VALUES ('00000000-0000-4000-8000-000000000001', ?, ?, '2026-01-02', 200000, 'January salary')`, testerID, salaryID)
	testutil.MustExec(t, db,
		`INSERT INTO cashflow_movements (sync_id, user_id, category_id, date, amount_cents) VALUES ('00000000-0000-4000-8000-000000000002', ?, ?, '2026-01-05', 5000)`, testerID, foodID)

	// The overview reads the current month, so give it income and an expense
	// over the budget to report as something that needs attention.
	thisMonth := time.Now().Format("2006-01")
	testutil.MustExec(t, db,
		`INSERT INTO cashflow_movements (sync_id, user_id, category_id, date, amount_cents) VALUES ('00000000-0000-4000-8000-000000000003', ?, ?, ?, 10000)`, testerID, salaryID, thisMonth+"-01")
	testutil.MustExec(t, db,
		`INSERT INTO cashflow_movements (sync_id, user_id, category_id, date, amount_cents) VALUES ('00000000-0000-4000-8000-000000000004', ?, ?, ?, 20000)`, testerID, foodID, thisMonth+"-01")
	testutil.MustExec(t, db,
		`INSERT INTO cashflow_balances (user_id, month, balance_cents) VALUES (?, '2026-01', 150000)`, testerID)

	// Spec path template -> concrete request path.
	paths := map[string]string{
		"/healthz":                        "/api/v1/healthz",
		"/info":                           "/api/v1/info",
		"/auth/me":                        "/api/v1/auth/me",
		"/overview":                       "/api/v1/overview",
		"/overview/net-worth":             "/api/v1/overview/net-worth",
		"/goals/accounts":                 "/api/v1/goals/accounts",
		"/goals/accounts/{id}":            fmt.Sprintf("/api/v1/goals/accounts/%d", accountID),
		"/goals/accounts/{id}/goals":      fmt.Sprintf("/api/v1/goals/accounts/%d/goals", accountID),
		"/goals/accounts/{id}/movements":  fmt.Sprintf("/api/v1/goals/accounts/%d/movements", accountID),
		"/goals/accounts/{id}/projection": fmt.Sprintf("/api/v1/goals/accounts/%d/projection", accountID),
		"/goals/{id}":                     fmt.Sprintf("/api/v1/goals/%d", goalID),
		"/credits":                        "/api/v1/credits",
		"/credits/due":                    "/api/v1/credits/due",
		"/credits/{id}":                   fmt.Sprintf("/api/v1/credits/%d", contractID),
		"/credits/{id}/payments":          fmt.Sprintf("/api/v1/credits/%d/payments", contractID),

		"/savings/accounts":                           "/api/v1/savings/accounts",
		"/savings/accounts/{id}":                      fmt.Sprintf("/api/v1/savings/accounts/%d", savingsID),
		"/savings/accounts/{id}/movements":            fmt.Sprintf("/api/v1/savings/accounts/%d/movements", savingsID),
		"/savings/accounts/{id}/snapshots":            fmt.Sprintf("/api/v1/savings/accounts/%d/snapshots", savingsID),
		"/investments/entities":                       "/api/v1/investments/entities",
		"/investments/entities/{id}":                  fmt.Sprintf("/api/v1/investments/entities/%d", entityID),
		"/investments/entities/{id}/instruments":      fmt.Sprintf("/api/v1/investments/entities/%d/instruments", entityID),
		"/investments/instruments/{id}":               fmt.Sprintf("/api/v1/investments/instruments/%d", instrumentID),
		"/investments/instruments/{id}/contributions": fmt.Sprintf("/api/v1/investments/instruments/%d/contributions", instrumentID),
		"/investments/instruments/{id}/snapshots":     fmt.Sprintf("/api/v1/investments/instruments/%d/snapshots", instrumentID),

		"/cashflow/categories":               "/api/v1/cashflow/categories",
		"/cashflow/budget/{month}":           "/api/v1/cashflow/budget/2026-01",
		"/cashflow/balance/{month}":          "/api/v1/cashflow/balance/2026-01",
		"/cashflow/months":                   "/api/v1/cashflow/months",
		"/cashflow/months/{month}":           "/api/v1/cashflow/months/2026-01",
		"/cashflow/months/{month}/movements": "/api/v1/cashflow/months/2026-01/movements",
	}

	specPaths, _ := spec["paths"].(map[string]any)
	for template, item := range specPaths {
		if dig(item, "get") == nil {
			continue
		}
		if _, ok := paths[template]; !ok {
			t.Errorf("GET %s is in openapi.yaml but not in this test's paths table", template)
		}
	}

	for template, path := range paths {
		t.Run(template, func(t *testing.T) {
			t.Parallel()

			schema := dig(specPaths, template, "get", "responses", "200", "content", "application/json", "schema")
			if schema == nil {
				t.Fatalf("openapi.yaml has no 200 JSON schema for GET %s", template)
			}
			response := serveRequest(t, srv, http.MethodGet, path, cookie, "")
			assertAPIStatus(t, response, http.StatusOK)
			checkShape(t, spec, schema, decodeAPI(t, response), template)
		})
	}
}

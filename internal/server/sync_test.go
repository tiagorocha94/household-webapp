package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	authservice "github.com/tiagorocha94/household-webapp/internal/auth/service"
	authstore "github.com/tiagorocha94/household-webapp/internal/auth/store"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

const syncPath = "/api/v1/sync/cashflow"

func syncID(n int) string { return fmt.Sprintf("00000000-0000-4000-8000-%012d", n) }

// enrolDevice makes a user and an enrolled device for them, and returns the
// device's bearer header.
func enrolDevice(t *testing.T, db *sql.DB, username string) map[string]string {
	t.Helper()

	insertTestUser(t, db, username, username+"-password")
	svc := authservice.New(authstore.New(db))
	code, _, err := svc.AddDevice(username, username+" phone")
	if err != nil {
		t.Fatalf("AddDevice(%s) error = %v", username, err)
	}
	token, _, err := svc.Enrol(code)
	if err != nil {
		t.Fatalf("Enrol(%s) error = %v", username, err)
	}
	return map[string]string{"Authorization": "Bearer " + token}
}

func upsertOp(n int, date string, cents int, note string) string {
	return fmt.Sprintf(`{"op":"upsert","syncId":%q,"category":"food","date":%q,"amountCents":%d,"description":%q}`, syncID(n), date, cents, note)
}

func batch(ops ...string) string { return `{"operations":[` + strings.Join(ops, ",") + `]}` }

func movementOwners(t *testing.T, db *sql.DB) map[string]string {
	t.Helper()
	rows, err := db.Query(`SELECT m.sync_id, u.username FROM cashflow_movements m JOIN users u ON u.id = m.user_id`)
	if err != nil {
		t.Fatalf("query movements: %v", err)
	}
	defer rows.Close()
	owners := map[string]string{}
	for rows.Next() {
		var id, user string
		if err := rows.Scan(&id, &user); err != nil {
			t.Fatalf("scan movement: %v", err)
		}
		owners[id] = user
	}
	return owners
}

func TestSyncEndToEndMatchesTheSpec(t *testing.T) {
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
	testutil.MustExec(t, db, `INSERT INTO cashflow_categories (name, direction, icon, color) VALUES ('food', 'expense', 'utensils', '#E76F51')`)
	alice := enrolDevice(t, db, "alice")

	body := batch(
		// The refused operation comes first: checkShape compares the first
		// result, and only a rejected one has a reason.
		upsertOp(2, "2026-02-30", 100, ""),
		upsertOp(1, "2026-03-05", 4000, "lunch"),
		fmt.Sprintf(`{"op":"delete","syncId":%q}`, syncID(3)),
	)
	response := serveRequestWith(t, srv, http.MethodPost, syncPath, nil, body, alice)
	assertAPIStatus(t, response, http.StatusOK)
	got := decodeAPI(t, response)
	checkShape(t, spec, dig(spec, "paths", "/sync/cashflow", "post", "responses", "200", "content", "application/json", "schema"), got, "POST /sync/cashflow")

	results := got.(map[string]any)["results"].([]any)
	wantStatus := []string{"rejected", "ok", "ok"}
	if len(results) != len(wantStatus) {
		t.Fatalf("results = %v, want %d", results, len(wantStatus))
	}
	for i, r := range results {
		m := r.(map[string]any)
		if m["status"] != wantStatus[i] {
			t.Errorf("result %d = %v, want status %s", i, m, wantStatus[i])
		}
	}
	if owners := movementOwners(t, db); len(owners) != 1 || owners[syncID(1)] != "alice" {
		t.Errorf("movements = %v, want only alice's %s", owners, syncID(1))
	}

	// The movement is visible through the read-only API too.
	cookie := loginTestUser(t, srv, "alice", "alice-password")
	response = serveRequest(t, srv, http.MethodGet, "/api/v1/cashflow/months/2026-03/movements", cookie, "")
	assertAPIStatus(t, response, http.StatusOK)
	if list := decodeAPI(t, response).([]any); len(list) != 1 || list[0].(map[string]any)["description"] != "lunch" {
		t.Errorf("movements of 2026-03 = %v, want the synced one", list)
	}
}

func TestSyncNeedsADeviceToken(t *testing.T) {
	t.Parallel()

	srv, db := newTestServer(t)
	testutil.MustExec(t, db, `INSERT INTO cashflow_categories (name, direction, icon, color) VALUES ('food', 'expense', 'utensils', '#E76F51')`)
	alice := enrolDevice(t, db, "alice")
	cookie := loginTestUser(t, srv, "alice", "alice-password")
	body := batch(upsertOp(1, "2026-03-05", 4000, ""))

	// No credentials, the browser session alone, and a wrong token are refused.
	assertAPIStatus(t, serveRequest(t, srv, http.MethodPost, syncPath, nil, body), http.StatusUnauthorized)
	assertAPIStatus(t, serveRequest(t, srv, http.MethodPost, syncPath, cookie, body), http.StatusUnauthorized)
	assertAPIStatus(t, serveRequestWith(t, srv, http.MethodPost, syncPath, nil, body, map[string]string{"Authorization": "Bearer nope"}), http.StatusUnauthorized)
	if owners := movementOwners(t, db); len(owners) != 0 {
		t.Fatalf("movements = %v after refused requests, want none", owners)
	}

	// A device token does not open the read-only API.
	assertAPIStatus(t, serveRequestWith(t, srv, http.MethodGet, "/api/v1/cashflow/months", nil, "", alice), http.StatusUnauthorized)

	// Revoking stops the device.
	assertAPIStatus(t, serveRequestWith(t, srv, http.MethodPost, syncPath, nil, body, alice), http.StatusOK)
	if err := authservice.New(authstore.New(db)).RevokeDevice("alice phone"); err != nil {
		t.Fatalf("RevokeDevice() error = %v", err)
	}
	assertAPIStatus(t, serveRequestWith(t, srv, http.MethodPost, syncPath, nil, body, alice), http.StatusUnauthorized)

	response := serveRequest(t, srv, http.MethodGet, syncPath, nil, "")
	assertAPIStatus(t, response, http.StatusMethodNotAllowed)
}

func TestSyncWritesOnlyForTheDevicesPerson(t *testing.T) {
	t.Parallel()

	srv, db := newTestServer(t)
	testutil.MustExec(t, db, `INSERT INTO cashflow_categories (name, direction, icon, color) VALUES ('food', 'expense', 'utensils', '#E76F51')`)
	alice := enrolDevice(t, db, "alice")
	bob := enrolDevice(t, db, "bob")

	assertAPIStatus(t, serveRequestWith(t, srv, http.MethodPost, syncPath, nil, batch(upsertOp(1, "2026-03-05", 4000, "")), alice), http.StatusOK)

	// Bob's device names alice in the body, which is ignored, and tries to
	// overwrite and delete her movement, which is refused.
	body := `{"username":"alice","userId":1,"operations":[` +
		upsertOp(1, "2026-03-05", 1, "takeover") + `,` +
		fmt.Sprintf(`{"op":"delete","syncId":%q}`, syncID(1)) + `,` +
		upsertOp(2, "2026-03-06", 500, "bob's") + `]}`
	response := serveRequestWith(t, srv, http.MethodPost, syncPath, nil, body, bob)
	assertAPIStatus(t, response, http.StatusOK)
	results := decodeAPI(t, response).(map[string]any)["results"].([]any)
	for i, want := range []string{"rejected", "rejected", "ok"} {
		if got := results[i].(map[string]any)["status"]; got != want {
			t.Errorf("result %d status = %v, want %s", i, got, want)
		}
	}
	owners := movementOwners(t, db)
	if owners[syncID(1)] != "alice" || owners[syncID(2)] != "bob" || len(owners) != 2 {
		t.Errorf("owners = %v, want movement 1 alice's and movement 2 bob's", owners)
	}
	var cents int
	if err := db.QueryRow("SELECT amount_cents FROM cashflow_movements WHERE sync_id = ?", syncID(1)).Scan(&cents); err != nil || cents != 4000 {
		t.Errorf("alice's movement = %d cents, %v, want it untouched at 4000", cents, err)
	}
}

func TestSyncRejectsMalformedBatches(t *testing.T) {
	t.Parallel()

	srv, db := newTestServer(t)
	testutil.MustExec(t, db, `INSERT INTO cashflow_categories (name, direction, icon, color) VALUES ('food', 'expense', 'utensils', '#E76F51')`)
	alice := enrolDevice(t, db, "alice")

	many := make([]string, 201)
	for i := range many {
		many[i] = upsertOp(i+1, "2026-03-05", 100, "")
	}
	bad := map[string]string{
		"empty body":          ``,
		"not json":            `nope`,
		"no operations key":   `{}`,
		"operations not list": `{"operations":{}}`,
		"amount not a number": `{"operations":[{"op":"upsert","syncId":"` + syncID(1) + `","amountCents":"4000"}]}`,
		"too many operations": batch(many...),
	}
	for name, body := range bad {
		assertAPIStatus(t, serveRequestWith(t, srv, http.MethodPost, syncPath, nil, body, alice), http.StatusBadRequest)
		if owners := movementOwners(t, db); len(owners) != 0 {
			t.Errorf("%s: %d movements were applied from a refused request", name, len(owners))
		}
	}

	// An empty batch is fine and has nothing to report.
	response := serveRequestWith(t, srv, http.MethodPost, syncPath, nil, `{"operations":[]}`, alice)
	assertAPIStatus(t, response, http.StatusOK)
	var out struct {
		Results []any `json:"results"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil || out.Results == nil || len(out.Results) != 0 {
		t.Errorf("empty batch answer = %s, want {\"results\":[]}", response.Body.String())
	}
}

func TestInfoReportsTheSyncAPIVersion(t *testing.T) {
	t.Parallel()

	srv, _ := newTestServer(t)
	response := serveRequest(t, srv, http.MethodGet, "/api/v1/info", nil, "")
	assertAPIStatus(t, response, http.StatusOK)
	if got := decodeAPI(t, response).(map[string]any)["syncApi"]; got != float64(syncAPIVersion) {
		t.Errorf("syncApi = %v, want %d", got, syncAPIVersion)
	}
}

// TestDeviceTokenReadsGoalsButNotOtherData checks the one read-only area that
// phones may read (ADR 012): goals, with the session cookie or a device token.
// TestDeviceTokenReadsTheBudgetTotalsOnly checks that a device reads the
// month's budget by group (ADR 015) but not the movements behind it.
func TestDeviceTokenReadsTheBudgetTotalsOnly(t *testing.T) {
	t.Parallel()

	srv, db := newTestServer(t)
	alice := enrolDevice(t, db, "alice")
	testutil.MustExec(t, db, "INSERT INTO cashflow_budget_groups (name, share_pct, position) VALUES ('Essential', 100, 1)")

	response := serveRequestWith(t, srv, http.MethodGet, "/api/v1/cashflow/budget/2026-03", nil, "", alice)
	assertAPIStatus(t, response, http.StatusOK)
	budget, _ := decodeAPI(t, response).(map[string]any)
	if groups, _ := budget["byGroup"].([]any); len(groups) != 1 || budget["month"] != "2026-03" {
		t.Errorf("budget read by a device = %v, want March with one group", budget)
	}
	for _, key := range []string{"byUser", "byCategory"} {
		if _, ok := budget[key]; ok {
			t.Errorf("budget has %q, want totals by group only", key)
		}
	}
	assertAPIStatus(t, serveRequestWith(t, srv, http.MethodGet, "/api/v1/cashflow/budget/2026-13", nil, "", alice), http.StatusBadRequest)
	assertAPIStatus(t, serveRequest(t, srv, http.MethodGet, "/api/v1/cashflow/budget/2026-03", nil, ""), http.StatusUnauthorized)
	assertAPIStatus(t, serveRequestWith(t, srv, http.MethodGet, "/api/v1/cashflow/months/2026-03/movements", nil, "", alice), http.StatusUnauthorized)
}

// TestDeviceTokenReadsOnlyItsOwnersCashBalance checks that a device reads the
// cash of the person it belongs to (ADR 019), worked out from the typed balance
// and the cashflow after it, and nobody else's.
func TestDeviceTokenReadsOnlyItsOwnersCashBalance(t *testing.T) {
	t.Parallel()

	srv, db := newTestServer(t)
	alice := enrolDevice(t, db, "alice")
	bob := enrolDevice(t, db, "bob")
	testutil.MustExec(t, db, "INSERT INTO cashflow_balances (user_id, month, balance_cents) SELECT id, '2026-02', 100000 FROM users WHERE username = 'alice'")
	testutil.MustExec(t, db, "INSERT INTO cashflow_categories (name, direction, icon, color) VALUES ('food', 'expense', 'utensils', '#E76F51')")
	testutil.MustExec(t, db, `INSERT INTO cashflow_movements (sync_id, user_id, category_id, date, amount_cents)
		SELECT '00000000-0000-4000-8000-000000000009', u.id, c.id, '2026-03-10', 2500 FROM users u, cashflow_categories c WHERE u.username = 'alice' AND c.name = 'food'`)

	cash := func(headers map[string]string, month string) map[string]any {
		response := serveRequestWith(t, srv, http.MethodGet, "/api/v1/cashflow/balance/"+month, nil, "", headers)
		assertAPIStatus(t, response, http.StatusOK)
		body, _ := decodeAPI(t, response).(map[string]any)
		return body
	}
	if got := cash(alice, "2026-03"); got["balanceCents"] != float64(97500) || got["month"] != "2026-03" {
		t.Errorf("alice's cash in March = %v, want 97500", got)
	}
	if got := cash(alice, "2026-02"); got["balanceCents"] != float64(100000) {
		t.Errorf("alice's cash in February = %v, want the typed 100000", got)
	}
	if got := cash(bob, "2026-03"); got["balanceCents"] != nil {
		t.Errorf("bob's cash = %v, want null: no balance was typed for him", got)
	}
	assertAPIStatus(t, serveRequestWith(t, srv, http.MethodGet, "/api/v1/cashflow/balance/2026-13", nil, "", alice), http.StatusBadRequest)
	assertAPIStatus(t, serveRequest(t, srv, http.MethodGet, "/api/v1/cashflow/balance/2026-03", nil, ""), http.StatusUnauthorized)
}

func TestDeviceTokenReadsGoalsButNotOtherData(t *testing.T) {
	t.Parallel()

	srv, db := newTestServer(t)
	alice := enrolDevice(t, db, "alice")
	testutil.MustExec(t, db, "INSERT INTO goal_accounts (name, monthly_deposit_cents) VALUES ('Household', 36600)")

	response := serveRequestWith(t, srv, http.MethodGet, "/api/v1/goals/accounts", nil, "", alice)
	assertAPIStatus(t, response, http.StatusOK)
	if accounts, _ := decodeAPI(t, response).([]any); len(accounts) != 1 {
		t.Errorf("accounts read by a device = %v, want 1", accounts)
	}
	assertAPIStatus(t, serveRequestWith(t, srv, http.MethodGet, "/api/v1/goals/accounts", nil, "", map[string]string{"Authorization": "Bearer nope"}), http.StatusUnauthorized)
	assertAPIStatus(t, serveRequest(t, srv, http.MethodGet, "/api/v1/goals/accounts", nil, ""), http.StatusUnauthorized)

	// Everything else that is read-only stays for the web session. Savings,
	// investments and credits are open to devices too (ADR 022).
	for _, path := range []string{"/api/v1/cashflow/months", "/api/v1/overview"} {
		assertAPIStatus(t, serveRequestWith(t, srv, http.MethodGet, path, nil, "", alice), http.StatusUnauthorized)
	}
}

// TestDeviceTokenReadsSavingsInvestmentsAndCredits checks that a device reads
// the accounts the phone lists and the credits due (ADR 022), with the owner of
// each account in the answer, and still nothing of the cashflow movements.
func TestDeviceTokenReadsSavingsInvestmentsAndCredits(t *testing.T) {
	t.Parallel()

	srv, db := newTestServer(t)
	alice := enrolDevice(t, db, "alice")
	var userID int64
	if err := db.QueryRow("SELECT id FROM users WHERE username = 'alice'").Scan(&userID); err != nil {
		t.Fatalf("find alice: %v", err)
	}
	testutil.MustExec(t, db, "INSERT INTO savings_accounts (name, user_id) VALUES ('Rainy day', ?)", userID)
	entityID := testutil.MustExec(t, db, "INSERT INTO investment_entities (name, kind, user_id) VALUES ('Broker', 'broker', ?)", userID)
	testutil.MustExec(t, db, "INSERT INTO investment_instruments (entity_id, name, type) VALUES (?, 'World ETF', 'etf')", entityID)
	creditID := testutil.MustExec(t, db,
		`INSERT INTO credit_contracts (name, category, contract_number, lender, original_principal_cents, start_month, expected_day)
		 VALUES ('Car', 'car', 'C-1', 'Bank', 100000, '2020-01', 5)`)
	testutil.MustExec(t, db,
		`INSERT INTO credit_payments (contract_id, month, type, principal_cents, interest_cents)
		 VALUES (?, '2020-01', 'regular', 25000, 1000)`, creditID)

	reads := []struct {
		path string
		want string
	}{
		{"/api/v1/savings/accounts", "owner"},
		{"/api/v1/investments/entities", "user"},
		{fmt.Sprintf("/api/v1/investments/entities/%d/instruments", entityID), "name"},
		{"/api/v1/credits", "expectedDay"},
		{fmt.Sprintf("/api/v1/credits/%d/payments", creditID), "totalCents"},
		{"/api/v1/credits/due", "expectedDay"},
	}
	for _, r := range reads {
		response := serveRequestWith(t, srv, http.MethodGet, r.path, nil, "", alice)
		assertAPIStatus(t, response, http.StatusOK)
		list, _ := decodeAPI(t, response).([]any)
		if len(list) != 1 {
			t.Fatalf("GET %s = %v, want one item", r.path, list)
		}
		if _, ok := list[0].(map[string]any)[r.want]; !ok {
			t.Errorf("GET %s item = %v, want a %q field", r.path, list[0], r.want)
		}
		assertAPIStatus(t, serveRequest(t, srv, http.MethodGet, r.path, nil, ""), http.StatusUnauthorized)
	}

	due := decodeAPI(t, serveRequestWith(t, srv, http.MethodGet, "/api/v1/credits/due", nil, "", alice)).([]any)[0].(map[string]any)
	if due["name"] != "Car" || due["totalCents"] != 26000.0 || due["expectedDay"] != 5.0 {
		t.Errorf("credit due = %v, want Car with the last instalment of 26000 and day 5", due)
	}
	assertAPIStatus(t, serveRequestWith(t, srv, http.MethodGet, "/api/v1/cashflow/months/2026-03/movements", nil, "", alice), http.StatusUnauthorized)
}

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

func decodeAPI(t *testing.T, response *httptest.ResponseRecorder) any {
	t.Helper()
	var body any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not json: %v; body = %s", err, response.Body.String())
	}
	return body
}

func assertAPIStatus(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	if response.Code != want {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, want, response.Body.String())
	}
	if ct := response.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content type = %q, want json", ct)
	}
}

func TestAPIGoals(t *testing.T) {
	t.Parallel()

	srv, db := newTestServer(t)
	insertTestUser(t, db, "tester", "tester-password")
	cookie := loginTestUser(t, srv, "tester", "tester-password")

	accountID := testutil.MustExec(t, db, "INSERT INTO goal_accounts (name, monthly_deposit_cents) VALUES ('Household', 36600)")
	goalID := testutil.MustExec(t, db,
		`INSERT INTO goal_account_goals (account_id, name, target_cents, start_month, end_month, status)
		 VALUES (?, 'Trip', 120050, '2026-01', '2026-12', 'active')`, accountID)
	testutil.MustExec(t, db, `INSERT INTO goal_account_movements (account_id, goal_id, type, amount_cents, month) VALUES (?, NULL, 'deposit', 50000, '2026-01')`, accountID)
	testutil.MustExec(t, db, `INSERT INTO goal_account_movements (account_id, goal_id, type, amount_cents, month) VALUES (?, ?, 'allocation', 10000, '2026-01')`, accountID, goalID)

	base := fmt.Sprintf("/api/v1/goals/accounts/%d", accountID)

	assertAPIStatus(t, serveRequest(t, srv, http.MethodGet, "/api/v1/goals/accounts", nil, ""), http.StatusUnauthorized)

	list := serveRequest(t, srv, http.MethodGet, "/api/v1/goals/accounts", cookie, "")
	assertAPIStatus(t, list, http.StatusOK)
	accounts := decodeAPI(t, list).([]any)
	if len(accounts) != 1 || accounts[0].(map[string]any)["monthlyDepositCents"] != 36600.0 {
		t.Fatalf("accounts = %v, want one account with monthlyDepositCents 36600", accounts)
	}

	detail := serveRequest(t, srv, http.MethodGet, base, cookie, "")
	assertAPIStatus(t, detail, http.StatusOK)
	balance := decodeAPI(t, detail).(map[string]any)["balance"].(map[string]any)
	if balance["depositsCents"] != 50000.0 || balance["unallocatedCents"] != 40000.0 {
		t.Fatalf("balance = %v, want depositsCents 50000 and unallocatedCents 40000", balance)
	}

	goals := serveRequest(t, srv, http.MethodGet, base+"/goals", cookie, "")
	assertAPIStatus(t, goals, http.StatusOK)
	first := decodeAPI(t, goals).([]any)[0].(map[string]any)
	if first["targetCents"] != 120050.0 || first["allocatedCents"] != 10000.0 {
		t.Fatalf("goal = %v, want targetCents 120050 and allocatedCents 10000", first)
	}

	assertAPIStatus(t, serveRequest(t, srv, http.MethodGet, base+"/movements", cookie, ""), http.StatusOK)

	projection := serveRequest(t, srv, http.MethodGet, base+"/projection", cookie, "")
	assertAPIStatus(t, projection, http.StatusOK)
	body := decodeAPI(t, projection).(map[string]any)
	if body["monthlyDepositCents"] != 36600.0 || len(body["months"].([]any)) == 0 {
		t.Fatalf("projection = %v", body)
	}

	goalPath := fmt.Sprintf("/api/v1/goals/%d", goalID)
	gd := serveRequest(t, srv, http.MethodGet, goalPath, cookie, "")
	assertAPIStatus(t, gd, http.StatusOK)
	movements := decodeAPI(t, gd).(map[string]any)["movements"].([]any)
	if len(movements) != 1 || !movements[0].(map[string]any)["goalInflow"].(bool) {
		t.Fatalf("goal movements = %v", movements)
	}

	assertAPIStatus(t, serveRequest(t, srv, http.MethodGet, "/api/v1/goals/accounts/9999", cookie, ""), http.StatusNotFound)
	assertAPIStatus(t, serveRequest(t, srv, http.MethodGet, "/api/v1/goals/accounts/9999/projection", cookie, ""), http.StatusNotFound)
	assertAPIStatus(t, serveRequest(t, srv, http.MethodGet, "/api/v1/goals/9999", cookie, ""), http.StatusNotFound)
	assertAPIStatus(t, serveRequest(t, srv, http.MethodGet, "/api/v1/goals/accounts/abc", cookie, ""), http.StatusBadRequest)
}

func TestAPICredits(t *testing.T) {
	t.Parallel()

	srv, db := newTestServer(t)
	insertTestUser(t, db, "tester", "tester-password")
	cookie := loginTestUser(t, srv, "tester", "tester-password")

	contractID := testutil.MustExec(t, db,
		`INSERT INTO credit_contracts (name, category, contract_number, lender, original_principal_cents, start_month)
		 VALUES ('Car', 'car', 'C-1', 'Bank', 100000, '2026-01')`)
	testutil.MustExec(t, db,
		`INSERT INTO credit_payments (contract_id, month, type, principal_cents, interest_cents)
		 VALUES (?, '2026-01', 'regular', 25000, 1000)`, contractID)

	path := fmt.Sprintf("/api/v1/credits/%d", contractID)

	assertAPIStatus(t, serveRequest(t, srv, http.MethodGet, "/api/v1/credits", nil, ""), http.StatusUnauthorized)

	list := serveRequest(t, srv, http.MethodGet, "/api/v1/credits", cookie, "")
	assertAPIStatus(t, list, http.StatusOK)
	item := decodeAPI(t, list).([]any)[0].(map[string]any)
	if item["outstandingPrincipalCents"] != 75000.0 || item["interestPaidCents"] != 1000.0 {
		t.Fatalf("contract = %v", item)
	}
	if item["expectedEndMonth"] != nil || item["lastPaymentMonth"] != "2026-01" {
		t.Fatalf("nullable fields = %v / %v", item["expectedEndMonth"], item["lastPaymentMonth"])
	}

	detail := serveRequest(t, srv, http.MethodGet, path, cookie, "")
	assertAPIStatus(t, detail, http.StatusOK)
	if got := decodeAPI(t, detail).(map[string]any); got["name"] != "Car" || got["status"] != "active" {
		t.Fatalf("contract = %v, want the Car contract itself, not a wrapper", got)
	}

	payments := serveRequest(t, srv, http.MethodGet, path+"/payments", cookie, "")
	assertAPIStatus(t, payments, http.StatusOK)
	p := decodeAPI(t, payments).([]any)[0].(map[string]any)
	if p["principalCents"] != 25000.0 || p["totalCents"] != 26000.0 {
		t.Fatalf("payment = %v", p)
	}

	assertAPIStatus(t, serveRequest(t, srv, http.MethodGet, "/api/v1/credits/9999", cookie, ""), http.StatusNotFound)
	assertAPIStatus(t, serveRequest(t, srv, http.MethodGet, "/api/v1/credits/abc", cookie, ""), http.StatusBadRequest)
}

func TestAPISavings(t *testing.T) {
	t.Parallel()

	srv, db := newTestServer(t)
	userID := insertTestUser(t, db, "tester", "tester-password")
	cookie := loginTestUser(t, srv, "tester", "tester-password")

	accountID := testutil.MustExec(t, db, "INSERT INTO savings_accounts (name, user_id, target_cents) VALUES ('Emergency fund', ?, 500000)", userID)
	testutil.MustExec(t, db, `INSERT INTO savings_movements (account_id, month, type, amount_cents) VALUES (?, '2026-01', 'deposit', 300000)`, accountID)
	testutil.MustExec(t, db, `INSERT INTO savings_movements (account_id, month, type, amount_cents) VALUES (?, '2026-02', 'withdrawal', 50000)`, accountID)
	testutil.MustExec(t, db, `INSERT INTO savings_snapshots (account_id, month, value_cents) VALUES (?, '2026-03', 255000)`, accountID)
	testutil.MustExec(t, db, "INSERT INTO savings_accounts (name) VALUES ('Holiday pot')")
	path := fmt.Sprintf("/api/v1/savings/accounts/%d", accountID)

	assertAPIStatus(t, serveRequest(t, srv, http.MethodGet, "/api/v1/savings/accounts", nil, ""), http.StatusUnauthorized)

	list := serveRequest(t, srv, http.MethodGet, "/api/v1/savings/accounts", cookie, "")
	assertAPIStatus(t, list, http.StatusOK)
	items := decodeAPI(t, list).([]any)
	if len(items) != 2 || items[0].(map[string]any)["name"] != "Emergency fund" {
		t.Fatalf("accounts = %v, want two ordered by name", items)
	}
	got := items[0].(map[string]any)
	if got["balanceCents"] != 255000.0 || got["interestCents"] != 5000.0 ||
		got["progressPct"] != 51.0 || got["owner"] != "tester" || got["targetCents"] != 500000.0 {
		t.Fatalf("account = %v", got)
	}
	if bare := items[1].(map[string]any); bare["owner"] != nil || bare["targetCents"] != nil || bare["lastSnapshotMonth"] != nil {
		t.Fatalf("account without owner, target or snapshot = %v, want nulls", bare)
	}

	detail := serveRequest(t, srv, http.MethodGet, path, cookie, "")
	assertAPIStatus(t, detail, http.StatusOK)
	if d := decodeAPI(t, detail).(map[string]any); d["name"] != "Emergency fund" {
		t.Fatalf("account = %v", d)
	}
	moves := decodeAPI(t, serveRequest(t, srv, http.MethodGet, path+"/movements", cookie, "")).([]any)
	if len(moves) != 2 || moves[0].(map[string]any)["type"] != "withdrawal" {
		t.Fatalf("movements = %v, want the withdrawal first", moves)
	}
	snaps := decodeAPI(t, serveRequest(t, srv, http.MethodGet, path+"/snapshots", cookie, "")).([]any)
	if len(snaps) != 1 || snaps[0].(map[string]any)["valueCents"] != 255000.0 {
		t.Fatalf("snapshots = %v", snaps)
	}

	assertAPIStatus(t, serveRequest(t, srv, http.MethodGet, "/api/v1/savings/accounts/9999", cookie, ""), http.StatusNotFound)
	assertAPIStatus(t, serveRequest(t, srv, http.MethodGet, "/api/v1/savings/accounts/abc", cookie, ""), http.StatusBadRequest)
}

func TestAPIOverview(t *testing.T) {
	t.Parallel()

	srv, db := newTestServer(t)
	insertTestUser(t, db, "tester", "tester-password")
	cookie := loginTestUser(t, srv, "tester", "tester-password")

	assertAPIStatus(t, serveRequest(t, srv, http.MethodGet, "/api/v1/overview", nil, ""), http.StatusUnauthorized)

	// Nothing recorded yet: every list is empty, never null, and no month is an error.
	response := serveRequest(t, srv, http.MethodGet, "/api/v1/overview", cookie, "")
	assertAPIStatus(t, response, http.StatusOK)
	got := decodeAPI(t, response).(map[string]any)
	for _, key := range []string{"attention", "activeGoals", "contracts", "holdings"} {
		if list, ok := got[key].([]any); !ok || len(list) != 0 {
			t.Errorf("%s = %v, want an empty list", key, got[key])
		}
	}
	if cashflow := got["cashflow"].(map[string]any); cashflow["incomeCents"] != 0.0 || cashflow["groups"] == nil {
		t.Errorf("cashflow = %v, want zero figures and an empty group list", cashflow)
	}
}

func TestUnmatchedRoutesReturnJSONNotFound(t *testing.T) {
	t.Parallel()

	srv, db := newTestServer(t)
	insertTestUser(t, db, "owner", "owner-password")
	cookie := loginTestUser(t, srv, "owner", "owner-password")

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/"},
		{http.MethodPost, "/login"},
		{http.MethodGet, "/api/v1/anything"},
	} {
		for _, c := range []*http.Cookie{nil, cookie} {
			response := serveRequest(t, srv, tc.method, tc.path, c, "")
			assertAPIStatus(t, response, http.StatusNotFound)
		}
	}
}

func TestWrongMethodOnKnownPathReturnsJSON405WithAllow(t *testing.T) {
	t.Parallel()

	srv, db := newTestServer(t)
	insertTestUser(t, db, "owner", "owner-password")
	cookie := loginTestUser(t, srv, "owner", "owner-password")

	for _, tc := range []struct{ method, path, allow string }{
		{http.MethodPost, "/api/v1/goals/accounts", "GET"}, // the API is read-only
		{http.MethodDelete, "/api/v1/credits/1", "GET"},
		{http.MethodGet, "/api/v1/auth/login", "POST"},
		{http.MethodPut, "/api/v1/healthz", "GET"},
	} {
		for _, c := range []*http.Cookie{nil, cookie} {
			response := serveRequest(t, srv, tc.method, tc.path, c, "")
			assertAPIStatus(t, response, http.StatusMethodNotAllowed)
			if got := response.Header().Get("Allow"); got != tc.allow {
				t.Errorf("%s %s: Allow = %q, want %q", tc.method, tc.path, got, tc.allow)
			}
			if got := decodeAPI(t, response).(map[string]any)["error"]; got != "Method Not Allowed" {
				t.Errorf("%s %s: error = %v, want Method Not Allowed", tc.method, tc.path, got)
			}
		}
	}
}

func TestHealthzNeedsNoAuthentication(t *testing.T) {
	t.Parallel()

	srv, _ := newTestServer(t)
	response := serveRequest(t, srv, http.MethodGet, "/api/v1/healthz", nil, "")
	assertAPIStatus(t, response, http.StatusOK)
	if got := decodeAPI(t, response).(map[string]any)["status"]; got != "ok" {
		t.Errorf("status = %v, want ok", got)
	}
}

func TestInfoReportsTheVersionWithoutAuthentication(t *testing.T) {
	t.Parallel()

	srv, _ := newTestServer(t)
	response := serveRequest(t, srv, http.MethodGet, "/api/v1/info", nil, "")
	assertAPIStatus(t, response, http.StatusOK)
	if got := decodeAPI(t, response).(map[string]any)["version"]; got != testVersion {
		t.Errorf("version = %v, want %s", got, testVersion)
	}
}

func TestResponsesCarryRequestIDAndSecurityHeaders(t *testing.T) {
	t.Parallel()

	srv, _ := newTestServer(t)
	response := serveRequestWith(t, srv, http.MethodGet, "/api/v1/auth/me", nil, "",
		map[string]string{"X-Request-Id": "abc-123"})

	assertAPIStatus(t, response, http.StatusUnauthorized)
	for header, want := range map[string]string{
		"X-Request-Id":           "abc-123",
		"X-Content-Type-Options": "nosniff",
		"Cache-Control":          "no-store",
	} {
		if got := response.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

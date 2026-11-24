package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	authapi "github.com/tiagorocha94/household-webapp/internal/auth/api"
	authservice "github.com/tiagorocha94/household-webapp/internal/auth/service"
	authstore "github.com/tiagorocha94/household-webapp/internal/auth/store"
	cashflowapi "github.com/tiagorocha94/household-webapp/internal/cashflow/api"
	cashflowservice "github.com/tiagorocha94/household-webapp/internal/cashflow/service"
	cashflowstore "github.com/tiagorocha94/household-webapp/internal/cashflow/store"
	creditsapi "github.com/tiagorocha94/household-webapp/internal/credits/api"
	creditsservice "github.com/tiagorocha94/household-webapp/internal/credits/service"
	creditsstore "github.com/tiagorocha94/household-webapp/internal/credits/store"
	goalsapi "github.com/tiagorocha94/household-webapp/internal/goals/api"
	goalsservice "github.com/tiagorocha94/household-webapp/internal/goals/service"
	goalsstore "github.com/tiagorocha94/household-webapp/internal/goals/store"
	investmentsapi "github.com/tiagorocha94/household-webapp/internal/investments/api"
	investmentsservice "github.com/tiagorocha94/household-webapp/internal/investments/service"
	investmentsstore "github.com/tiagorocha94/household-webapp/internal/investments/store"
	"github.com/tiagorocha94/household-webapp/internal/overview"
	overviewapi "github.com/tiagorocha94/household-webapp/internal/overview/api"
	savingsapi "github.com/tiagorocha94/household-webapp/internal/savings/api"
	savingsservice "github.com/tiagorocha94/household-webapp/internal/savings/service"
	savingsstore "github.com/tiagorocha94/household-webapp/internal/savings/store"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

const (
	// testVersion is the version the test server reports.
	testVersion = "v0.0.0-test"
)

func newTestServer(t *testing.T) (*Server, *sql.DB) {
	t.Helper()
	return newTestServerWith(t, authapi.Options{})
}

func newTestServerWith(t *testing.T, authOptions authapi.Options) (*Server, *sql.DB) {
	t.Helper()
	return newTestServerOptions(t, authOptions, Options{})
}

func newTestServerOptions(t *testing.T, authOptions authapi.Options, options Options) (*Server, *sql.DB) {
	t.Helper()

	db := testutil.NewDB(t)
	authService := authservice.New(authstore.New(db))
	goalsService := goalsservice.New(goalsstore.New(db))
	creditsService := creditsservice.New(creditsstore.New(db))
	investmentsService := investmentsservice.New(investmentsstore.New(db))
	savingsService := savingsservice.New(savingsstore.New(db))
	cashflowService := cashflowservice.New(cashflowstore.New(db))
	srv := New(
		testutil.Logger(),
		testVersion,
		"127.0.0.1:0",
		authService,
		authapi.New(authService, testutil.Logger(), authOptions),
		goalsapi.New(goalsService),
		creditsapi.New(creditsService),
		investmentsapi.New(investmentsService),
		savingsapi.New(savingsService),
		cashflowapi.New(cashflowService),
		overviewapi.New(
			overview.New(goalsService, creditsService, investmentsService, savingsService, cashflowService),
			overview.NewHistory(goalsService, creditsService, investmentsService, savingsService, cashflowService),
		),
		options,
	)
	return srv, db
}

func insertTestUser(t *testing.T, db *sql.DB, username, password string) int64 {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password for %q: %v", username, err)
	}
	result, err := db.Exec(
		"INSERT INTO users (username, password_hash) VALUES (?, ?)",
		username, string(hash),
	)
	if err != nil {
		t.Fatalf("insert test user %q: %v", username, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("get test user %q ID: %v", username, err)
	}
	return id
}

func loginTestUser(t *testing.T, srv *Server, username, password string) *http.Cookie {
	t.Helper()

	body, err := json.Marshal(map[string]string{"username": username, "password": password})
	if err != nil {
		t.Fatalf("marshal login body: %v", err)
	}
	response := serveRequest(t, srv, http.MethodPost, "/api/v1/auth/login", nil, string(body))
	if response.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	result := response.Result()
	t.Cleanup(func() { result.Body.Close() })
	cookies := result.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("login cookies = %d, want 1", len(cookies))
	}
	return cookies[0]
}

// serveRequest sends a request through the full middleware chain. A non-empty body is
// sent as JSON.
func serveRequest(t *testing.T, srv *Server, method, path string, cookie *http.Cookie, body string) *httptest.ResponseRecorder {
	t.Helper()
	return serveRequestWith(t, srv, method, path, cookie, body, nil)
}

// serveRequestWith is serveRequest with extra request headers.
func serveRequestWith(t *testing.T, srv *Server, method, path string, cookie *http.Cookie, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range headers {
		request.Header.Set(k, v)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	srv.handler.ServeHTTP(response, request)
	return response
}

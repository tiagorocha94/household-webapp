package server

import (
	"net/http"
	"strconv"
	"testing"

	authapi "github.com/tiagorocha94/household-webapp/internal/auth/api"
)

const authAPIPrefix = "/api/v1/auth"

func TestLoginAndMe(t *testing.T) {
	t.Parallel()

	srv, db := newTestServer(t)
	userID := insertTestUser(t, db, "owner", "owner-password")

	// Username is trimmed and case-insensitive.
	response := serveRequest(t, srv, http.MethodPost, authAPIPrefix+"/login", nil,
		`{"username":"  OWNER ","password":"owner-password"}`)
	assertAPIStatus(t, response, http.StatusOK)
	body := decodeAPI(t, response).(map[string]any)
	if body["id"] != float64(userID) || body["username"] != "owner" {
		t.Fatalf("login body = %v, want id %d and username owner", body, userID)
	}

	result := response.Result()
	defer result.Body.Close()
	var cookie *http.Cookie
	for _, c := range result.Cookies() {
		if c.Name == "session_token" {
			cookie = c
		}
	}
	if cookie == nil || cookie.Value == "" || !cookie.HttpOnly || cookie.MaxAge <= 0 {
		t.Fatalf("session cookie = %+v, want a non-empty HttpOnly cookie with lifetime", cookie)
	}

	me := serveRequest(t, srv, http.MethodGet, authAPIPrefix+"/me", cookie, "")
	assertAPIStatus(t, me, http.StatusOK)
	meBody := decodeAPI(t, me).(map[string]any)
	if meBody["id"] != float64(userID) || meBody["username"] != "owner" {
		t.Fatalf("me body = %v", meBody)
	}
}

func TestLoginRejectsBadRequests(t *testing.T) {
	t.Parallel()

	srv, db := newTestServer(t)
	insertTestUser(t, db, "owner", "owner-password")

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{"wrong password", `{"username":"owner","password":"nope"}`, http.StatusUnauthorized, "Invalid credentials"},
		{"unknown user", `{"username":"ghost","password":"x"}`, http.StatusUnauthorized, "Invalid credentials"},
		{"malformed json", `{"username":`, http.StatusBadRequest, "Bad Request"},
		{"empty body", ``, http.StatusBadRequest, "Bad Request"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			response := serveRequest(t, srv, http.MethodPost, authAPIPrefix+"/login", nil, tt.body)
			assertAPIStatus(t, response, tt.wantStatus)
			if got := decodeAPI(t, response).(map[string]any)["error"]; got != tt.wantError {
				t.Fatalf("error = %v, want %q", got, tt.wantError)
			}
			result := response.Result()
			defer result.Body.Close()
			if len(result.Cookies()) != 0 {
				t.Fatalf("failed login set cookies: %v", result.Cookies())
			}
		})
	}
}

func TestCurrentUserRequiresAuthentication(t *testing.T) {
	t.Parallel()

	srv, _ := newTestServer(t)

	response := serveRequest(t, srv, http.MethodGet, authAPIPrefix+"/me", nil, "")
	assertAPIStatus(t, response, http.StatusUnauthorized)
}

func TestLogout(t *testing.T) {
	t.Parallel()

	srv, db := newTestServer(t)
	insertTestUser(t, db, "owner", "owner-password")
	cookie := loginTestUser(t, srv, "owner", "owner-password")

	response := serveRequest(t, srv, http.MethodPost, authAPIPrefix+"/logout", cookie, "")
	if response.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if response.Body.Len() != 0 {
		t.Fatalf("logout body = %q, want empty", response.Body.String())
	}
	result := response.Result()
	defer result.Body.Close()
	cookies := result.Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge >= 0 {
		t.Fatalf("logout cookies = %v, want one expired cookie", cookies)
	}

	// The old token is no longer accepted.
	me := serveRequest(t, srv, http.MethodGet, authAPIPrefix+"/me", cookie, "")
	assertAPIStatus(t, me, http.StatusUnauthorized)

	// Logging out without a session is not an error.
	again := serveRequest(t, srv, http.MethodPost, authAPIPrefix+"/logout", nil, "")
	if again.Code != http.StatusNoContent {
		t.Fatalf("logout without session status = %d, want %d", again.Code, http.StatusNoContent)
	}
}

func TestLogoutRejectsCrossSiteRequests(t *testing.T) {
	t.Parallel()

	srv, db := newTestServer(t)
	insertTestUser(t, db, "owner", "owner-password")
	cookie := loginTestUser(t, srv, "owner", "owner-password")

	response := serveRequestWith(t, srv, http.MethodPost, authAPIPrefix+"/logout", cookie, "",
		map[string]string{"Sec-Fetch-Site": "cross-site"})
	assertAPIStatus(t, response, http.StatusForbidden)

	// The foreign request did not end the session.
	me := serveRequest(t, srv, http.MethodGet, authAPIPrefix+"/me", cookie, "")
	assertAPIStatus(t, me, http.StatusOK)

	same := serveRequestWith(t, srv, http.MethodPost, authAPIPrefix+"/logout", cookie, "",
		map[string]string{"Sec-Fetch-Site": "same-origin"})
	if same.Code != http.StatusNoContent {
		t.Fatalf("same-origin logout status = %d, want %d", same.Code, http.StatusNoContent)
	}
}

func TestLoginIsRateLimitedAfterRepeatedFailures(t *testing.T) {
	t.Parallel()

	srv, db := newTestServer(t)
	insertTestUser(t, db, "owner", "owner-password")
	insertTestUser(t, db, "other", "other-password")

	const attempts = 10
	for i := range attempts {
		response := serveRequest(t, srv, http.MethodPost, authAPIPrefix+"/login", nil,
			`{"username":"owner","password":"wrong"}`)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("failed attempt %d status = %d, want %d", i+1, response.Code, http.StatusUnauthorized)
		}
	}

	// Blocked now, even with the right password.
	response := serveRequest(t, srv, http.MethodPost, authAPIPrefix+"/login", nil,
		`{"username":"owner","password":"owner-password"}`)
	assertAPIStatus(t, response, http.StatusTooManyRequests)
	if seconds, err := strconv.Atoi(response.Header().Get("Retry-After")); err != nil || seconds <= 0 {
		t.Errorf("Retry-After = %q, want a positive number of seconds", response.Header().Get("Retry-After"))
	}
}

func TestSessionCookieSecureFlagFollowsTheOption(t *testing.T) {
	t.Parallel()

	for _, secure := range []bool{false, true} {
		srv, db := newTestServerWith(t, authapi.Options{SecureCookies: secure})
		insertTestUser(t, db, "owner", "owner-password")

		cookie := loginTestUser(t, srv, "owner", "owner-password")
		if cookie.Secure != secure {
			t.Errorf("SecureCookies=%v: login cookie Secure = %v", secure, cookie.Secure)
		}
		out := serveRequest(t, srv, http.MethodPost, authAPIPrefix+"/logout", cookie, "")
		result := out.Result()
		cleared := result.Cookies()
		result.Body.Close()
		if len(cleared) != 1 || cleared[0].Secure != secure {
			t.Errorf("SecureCookies=%v: logout cookies = %v, want one with Secure = %v", secure, cleared, secure)
		}
	}
}

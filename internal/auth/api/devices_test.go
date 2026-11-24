package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/tiagorocha94/household-webapp/internal/auth"
	"github.com/tiagorocha94/household-webapp/internal/auth/service"
	"github.com/tiagorocha94/household-webapp/internal/auth/store"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

// newDeviceHandler returns a Handler on a fresh database with the user alice.
func newDeviceHandler(t *testing.T) (*Handler, *service.Service) {
	t.Helper()

	db := testutil.NewDB(t)
	testutil.InsertUser(t, db, "alice")
	svc := service.New(store.New(db))
	return New(svc, testutil.Logger(), Options{}), svc
}

func enrolRequest(h *Handler, remoteAddr, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/devices/enrol", strings.NewReader(body))
	r.RemoteAddr = remoteAddr
	w := httptest.NewRecorder()
	h.Enrol(w, r)
	return w
}

func enrolToken(t *testing.T, h *Handler, svc *service.Service, name string) string {
	t.Helper()

	code, _, err := svc.AddDevice("alice", name)
	if err != nil {
		t.Fatalf("AddDevice() error = %v", err)
	}
	w := enrolRequest(h, "198.51.100.1:1000", `{"code":"`+code+`"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("Enrol status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	var got apiEnrolResponse
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode enrol response: %v", err)
	}
	if got.Token == "" || got.Username != "alice" || got.DeviceName != name {
		t.Fatalf("Enrol response = %+v, want a token for alice's %s", got, name)
	}
	return got.Token
}

func TestEnrolAnswersWithTheTokenOnce(t *testing.T) {
	t.Parallel()

	h, svc := newDeviceHandler(t)
	code, _, err := svc.AddDevice("alice", "phone")
	if err != nil {
		t.Fatalf("AddDevice() error = %v", err)
	}
	body := `{"code":"` + code + `"}`

	if w := enrolRequest(h, "198.51.100.1:1", body); w.Code != http.StatusOK {
		t.Fatalf("first Enrol status = %d, want 200; body = %s", w.Code, w.Body.String())
	} else if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store on a response holding a token", got)
	}
	if w := enrolRequest(h, "198.51.100.1:1", body); w.Code != http.StatusUnauthorized {
		t.Errorf("second Enrol status = %d, want 401", w.Code)
	}
}

func TestEnrolRejectsBadRequests(t *testing.T) {
	t.Parallel()

	h, _ := newDeviceHandler(t)
	for _, body := range []string{``, `not json`, `{}`, `{"code":"  "}`, `{"code":"` + strings.Repeat("A", 2000) + `"}`} {
		if w := enrolRequest(h, "198.51.100.1:1", body); w.Code != http.StatusBadRequest {
			t.Errorf("Enrol(%.20q) status = %d, want 400", body, w.Code)
		}
	}
}

func TestEnrolIsRateLimitedPerAddress(t *testing.T) {
	t.Parallel()

	h, svc := newDeviceHandler(t)
	code, _, err := svc.AddDevice("alice", "phone")
	if err != nil {
		t.Fatalf("AddDevice() error = %v", err)
	}

	for i := range maxLoginFailures {
		if w := enrolRequest(h, "203.0.113.9:1", `{"code":"WRONG-CODE1"}`); w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d status = %d, want 401", i+1, w.Code)
		}
	}
	w := enrolRequest(h, "203.0.113.9:2", `{"code":"`+code+`"}`)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status after %d failures = %d, want 429 even for the right code", maxLoginFailures, w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("429 has no Retry-After")
	}

	// Another address is not affected.
	if w := enrolRequest(h, "203.0.113.10:1", `{"code":"`+code+`"}`); w.Code != http.StatusOK {
		t.Errorf("other address status = %d, want 200", w.Code)
	}
}

// whoAmI is a handler behind RequireDevice that reports what it was given.
func whoAmI(t *testing.T, h *Handler) http.Handler {
	t.Helper()
	return h.RequireDevice(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		device, user := auth.DeviceFromContext(r.Context()), auth.UserFromContext(r.Context())
		if device == nil || user == nil || device.UserID != user.ID {
			t.Errorf("context holds device %v and user %v, want both, of the same person", device, user)
			return
		}
		_, _ = w.Write([]byte(user.Username + "/" + device.Name))
	}))
}

func deviceRequest(next http.Handler, remoteAddr string, headers map[string]string, cookie *http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/sync/cashflow", nil)
	r.RemoteAddr = remoteAddr
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	next.ServeHTTP(w, r)
	return w
}

func TestRequireDeviceAcceptsOnlyAnEnrolledDevicesBearerToken(t *testing.T) {
	t.Parallel()

	h, svc := newDeviceHandler(t)
	token := enrolToken(t, h, svc, "phone")
	next := whoAmI(t, h)

	w := deviceRequest(next, "198.51.100.1:1", map[string]string{"Authorization": "Bearer " + token}, nil)
	if w.Code != http.StatusOK || w.Body.String() != "alice/phone" {
		t.Errorf("valid token: status = %d, body = %q, want 200 alice/phone", w.Code, w.Body.String())
	}
	w = deviceRequest(next, "198.51.100.1:1", map[string]string{"Authorization": "bearer " + token}, nil)
	if w.Code != http.StatusOK {
		t.Errorf("lower-case scheme status = %d, want 200", w.Code)
	}

	rejected := map[string]map[string]string{
		"no header":      nil,
		"wrong token":    {"Authorization": "Bearer " + strings.Repeat("0", 64)},
		"no scheme":      {"Authorization": token},
		"basic scheme":   {"Authorization": "Basic " + token},
		"empty token":    {"Authorization": "Bearer "},
		"token as query": {"X-Token": token},
	}
	for name, headers := range rejected {
		w := deviceRequest(next, "198.51.100.2:1", headers, nil)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, want 401", name, w.Code)
		}
		if w.Header().Get("WWW-Authenticate") != "Bearer" {
			t.Errorf("%s: WWW-Authenticate = %q, want Bearer", name, w.Header().Get("WWW-Authenticate"))
		}
	}
}

func TestRequireDeviceIgnoresTheSessionCookieAndRequireAuthIgnoresTheToken(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	testutil.MustExec(t, db, "INSERT INTO users (username, password_hash) VALUES ('alice', ?)", string(hash))
	svc := service.New(store.New(db))
	h := New(svc, testutil.Logger(), Options{})

	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"alice","password":"secret"}`))
	login.RemoteAddr = "198.51.100.1:1"
	lw := httptest.NewRecorder()
	h.Login(lw, login)
	cookies := lw.Result().Cookies()
	if lw.Code != http.StatusOK || len(cookies) != 1 {
		t.Fatalf("login status = %d with %d cookies, want 200 with 1", lw.Code, len(cookies))
	}

	// A logged-in browser session is not a device.
	if w := deviceRequest(whoAmI(t, h), "198.51.100.1:2", nil, cookies[0]); w.Code != http.StatusUnauthorized {
		t.Errorf("RequireDevice with only a session cookie: status = %d, want 401", w.Code)
	}

	// And a device token does not open the read-only browser API.
	code, _, err := svc.AddDevice("alice", "phone")
	if err != nil {
		t.Fatalf("AddDevice() error = %v", err)
	}
	token, _, err := svc.Enrol(code)
	if err != nil {
		t.Fatalf("Enrol() error = %v", err)
	}
	reachedHandler := false
	guarded := RequireAuth(svc, testutil.Logger(), http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reachedHandler = true }))
	if w := deviceRequest(guarded, "198.51.100.1:3", map[string]string{"Authorization": "Bearer " + token}, nil); w.Code != http.StatusUnauthorized || reachedHandler {
		t.Errorf("RequireAuth with a device token: status = %d, reached = %v, want 401 and not reached", w.Code, reachedHandler)
	}
}

func TestRequireDeviceRejectsARevokedDevice(t *testing.T) {
	t.Parallel()

	h, svc := newDeviceHandler(t)
	token := enrolToken(t, h, svc, "phone")
	next := whoAmI(t, h)
	headers := map[string]string{"Authorization": "Bearer " + token}

	if w := deviceRequest(next, "198.51.100.1:1", headers, nil); w.Code != http.StatusOK {
		t.Fatalf("before revoking: status = %d, want 200", w.Code)
	}
	if err := svc.RevokeDevice("phone"); err != nil {
		t.Fatalf("RevokeDevice() error = %v", err)
	}
	if w := deviceRequest(next, "198.51.100.1:1", headers, nil); w.Code != http.StatusUnauthorized {
		t.Errorf("after revoking: status = %d, want 401", w.Code)
	}
}

func TestRequireDeviceIsRateLimitedPerAddress(t *testing.T) {
	t.Parallel()

	h, svc := newDeviceHandler(t)
	token := enrolToken(t, h, svc, "phone")
	next := whoAmI(t, h)

	for i := range maxLoginFailures {
		if w := deviceRequest(next, "203.0.113.9:1", map[string]string{"Authorization": "Bearer wrong"}, nil); w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d status = %d, want 401", i+1, w.Code)
		}
	}
	good := map[string]string{"Authorization": "Bearer " + token}
	w := deviceRequest(next, "203.0.113.9:2", good, nil)
	if w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") == "" {
		t.Errorf("after %d failures: status = %d, Retry-After = %q, want 429 with Retry-After", maxLoginFailures, w.Code, w.Header().Get("Retry-After"))
	}
	if w := deviceRequest(next, "203.0.113.10:1", good, nil); w.Code != http.StatusOK {
		t.Errorf("other address status = %d, want 200", w.Code)
	}
}

func TestRequireSessionOrDeviceAcceptsASessionOrADeviceTokenAndNothingElse(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	testutil.MustExec(t, db, "INSERT INTO users (username, password_hash) VALUES ('alice', ?)", string(hash))
	svc := service.New(store.New(db))
	h := New(svc, testutil.Logger(), Options{})

	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"alice","password":"secret"}`))
	login.RemoteAddr = "198.51.100.1:1"
	lw := httptest.NewRecorder()
	h.Login(lw, login)
	cookies := lw.Result().Cookies()
	if lw.Code != http.StatusOK || len(cookies) != 1 {
		t.Fatalf("login status = %d with %d cookies, want 200 with 1", lw.Code, len(cookies))
	}
	code, _, err := svc.AddDevice("alice", "phone")
	if err != nil {
		t.Fatalf("AddDevice() error = %v", err)
	}
	token, _, err := svc.Enrol(code)
	if err != nil {
		t.Fatalf("Enrol() error = %v", err)
	}

	reached := 0
	guarded := h.RequireSessionOrDevice(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached++
		if auth.UserFromContext(r.Context()) == nil {
			t.Error("the handler was reached without a user in the context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	if w := deviceRequest(guarded, "198.51.100.1:2", nil, cookies[0]); w.Code != http.StatusOK {
		t.Errorf("session cookie: status = %d, want 200", w.Code)
	}
	if w := deviceRequest(guarded, "198.51.100.1:3", map[string]string{"Authorization": "Bearer " + token}, nil); w.Code != http.StatusOK {
		t.Errorf("device token: status = %d, want 200", w.Code)
	}
	if reached != 2 {
		t.Fatalf("handler reached %d times, want 2", reached)
	}

	// Neither, or a wrong token (even next to a valid session), is refused.
	if w := deviceRequest(guarded, "198.51.100.2:1", nil, nil); w.Code != http.StatusUnauthorized {
		t.Errorf("no credentials: status = %d, want 401", w.Code)
	}
	wrong := map[string]string{"Authorization": "Bearer " + strings.Repeat("0", 64)}
	w := deviceRequest(guarded, "198.51.100.2:2", wrong, cookies[0])
	if w.Code != http.StatusUnauthorized || w.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Errorf("wrong token with a valid session: status = %d, WWW-Authenticate = %q, want 401 Bearer", w.Code, w.Header().Get("WWW-Authenticate"))
	}
	if reached != 2 {
		t.Errorf("handler reached %d times after refused requests, want still 2", reached)
	}
}

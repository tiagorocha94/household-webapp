package server

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRecoverPanics(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	h := recoverPanics(logger, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	response := httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/x", nil))

	assertAPIStatus(t, response, http.StatusInternalServerError)
	if got := decodeAPI(t, response).(map[string]any)["error"]; got != "Internal Server Error" {
		t.Errorf("response error = %v, want a generic message", got)
	}
	if strings.Contains(response.Body.String(), "boom") {
		t.Errorf("response leaked the panic value: %s", response.Body.String())
	}
	if out := buf.String(); !strings.Contains(out, "handler panicked") || !strings.Contains(out, "boom") || !strings.Contains(out, "path=/x") {
		t.Errorf("log output = %q", out)
	}
}

func TestRecoverPanicsKeepsAbortHandler(t *testing.T) {
	t.Parallel()

	h := recoverPanics(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) }))

	defer func() {
		if err, _ := recover().(error); !errors.Is(err, http.ErrAbortHandler) {
			t.Fatalf("recovered %v, want http.ErrAbortHandler to pass through", err)
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
}

func TestRequestIDKeepsAValidCallerIDAndReplacesOthers(t *testing.T) {
	t.Parallel()

	var seen string
	h := requestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = requestIDFrom(r.Context())
	}))

	tests := []struct {
		name, sent string
		keep       bool
	}{
		{"valid", "req-1.A_b", true},
		{"empty", "", false},
		{"spaces", "a b", false},
		{"newline", "a\nb", false},
		{"too long", strings.Repeat("a", 65), false},
	}
	for _, tt := range tests {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		if tt.sent != "" {
			r.Header["X-Request-Id"] = []string{tt.sent}
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		got := w.Header().Get("X-Request-Id")
		if tt.keep && got != tt.sent || !tt.keep && (got == tt.sent || got == "") {
			t.Errorf("%s: response ID = %q for sent %q", tt.name, got, tt.sent)
		}
		if seen != got {
			t.Errorf("%s: context ID = %q, response ID = %q, want the same", tt.name, seen, got)
		}
	}
}

func TestAccessLogRecordsStatusAndRequestID(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	h := requestID(accessLog(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})))

	r := httptest.NewRequest(http.MethodGet, "/brew", nil)
	r.Header.Set("X-Request-Id", "id-7")
	h.ServeHTTP(httptest.NewRecorder(), r)

	out := buf.String()
	for _, want := range []string{"msg=request", "method=GET", "path=/brew", "status=418", "request_id=id-7", "duration="} {
		if !strings.Contains(out, want) {
			t.Errorf("log %q is missing %q", out, want)
		}
	}
	if !strings.Contains(out, "level=INFO") {
		t.Errorf("log %q: a 418 should log at info level", out)
	}
}

func TestAccessLogDefaultsToOKAndLogsServerErrorsAtErrorLevel(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	accessLog(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hi"))
	})).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !strings.Contains(buf.String(), "status=200") {
		t.Errorf("log %q: want status=200 when the handler only writes a body", buf.String())
	}

	buf.Reset()
	accessLog(logger, recoverPanics(logger, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if out := buf.String(); !strings.Contains(out, "status=500") || !strings.Contains(out, "level=ERROR msg=request") {
		t.Errorf("log %q: want the recovered panic logged as a 500 at error level", out)
	}
}

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	securityHeaders(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).
		ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
}

func TestLimitBodyCapsRequestBodies(t *testing.T) {
	t.Parallel()

	var readErr error
	h := limitBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/",
		strings.NewReader(strings.Repeat("x", maxRequestBodyBytes+1))))
	if readErr == nil {
		t.Error("reading a body over the cap succeeded, want an error")
	}

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/",
		strings.NewReader(strings.Repeat("x", maxRequestBodyBytes))))
	if readErr != nil {
		t.Errorf("reading a body at the cap failed: %v", readErr)
	}
}

func TestWithTimeoutAnswersJSON503WhenTheHandlerIsSlow(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	defer close(release)
	h := withTimeout(20*time.Millisecond, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	assertAPIStatus(t, w, http.StatusServiceUnavailable)
	if got := decodeAPI(t, w).(map[string]any)["error"]; got != "Service Unavailable" {
		t.Errorf("response error = %v, want Service Unavailable", got)
	}

	fast := httptest.NewRecorder()
	withTimeout(time.Second, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(fast, httptest.NewRequest(http.MethodGet, "/", nil))
	if fast.Code != http.StatusNoContent {
		t.Errorf("fast handler status = %d, want %d", fast.Code, http.StatusNoContent)
	}
}

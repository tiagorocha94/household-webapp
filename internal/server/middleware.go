package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/api"
)

const (
	// requestIDHeader carries the request ID on the request and the response.
	requestIDHeader = "X-Request-Id"
	// maxRequestBodyBytes is the default cap on a request body. The API has no
	// large uploads, and login sets a smaller cap of its own.
	maxRequestBodyBytes = 1 << 20
)

type requestIDKey struct{}

// statusRecorder remembers the status code a handler wrote.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// recoverPanics turns a panic in next into a JSON 500 and logs it with the
// request and stack. http.ErrAbortHandler is passed through, because it is the
// documented way to abort a response on purpose.
func recoverPanics(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(rec)
			}
			logger.Error("handler panicked",
				"method", r.Method, "path", r.URL.Path, "request_id", requestIDFrom(r.Context()),
				"panic", rec, "stack", string(debug.Stack()))
			api.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		}()
		next.ServeHTTP(w, r)
	})
}

// requestID gives every request an ID. It keeps a sane ID sent by the caller
// (up to 64 letters, digits, dots, dashes and underscores) and generates one
// otherwise, then adds it to the response and to the request context.
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if !validRequestID(id) {
			id = newRequestID()
		}
		w.Header().Set(requestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

// requestIDFrom returns the request ID stored by requestID, or "".
func requestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

func validRequestID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, c := range id {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '.', c == '-', c == '_':
		default:
			return false
		}
	}
	return true
}

func newRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}

// accessLog logs one line per request: method, path, status, duration and
// request ID. Server errors log at error level, the rest at info. Put it
// inside requestID, so the ID is available.
func accessLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)

		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}
		level := slog.LevelInfo
		if status >= http.StatusInternalServerError {
			level = slog.LevelError
		}
		logger.Log(r.Context(), level, "request",
			"method", r.Method, "path", r.URL.Path, "status", status,
			"duration", time.Since(start), "request_id", requestIDFrom(r.Context()))
	})
}

// securityHeaders stops browsers from guessing content types and from caching
// responses, which can hold private data.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// limitBody caps every request body at maxRequestBodyBytes.
func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		next.ServeHTTP(w, r)
	})
}

// withTimeout answers 503 with a JSON error when next takes longer than d.
func withTimeout(d time.Duration, next http.Handler) http.Handler {
	timeout := http.TimeoutHandler(next, d, `{"error":"Service Unavailable"}`)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set before the timeout handler runs, because it writes its own
		// response without the inner handler's headers when it times out.
		w.Header().Set("Content-Type", "application/json")
		timeout.ServeHTTP(w, r)
	})
}

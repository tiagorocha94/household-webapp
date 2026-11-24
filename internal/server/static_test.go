package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	testIndexHTML = "<!doctype html><title>Household</title>"
	testAppJS     = "console.log('app')"
)

// writeFrontend writes a small built frontend and returns its folder. A file
// next to the folder, outside it, must never be served.
func writeFrontend(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "dist")
	for name, content := range map[string]string{
		"dist/index.html":        testIndexHTML,
		"dist/assets/app.abc.js": testAppJS,
		"dist/favicon.svg":       "<svg/>",
		"outside.txt":            "secret",
	} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("create folder for %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

// serveStatic sends a request through the static handler wrapped like the
// server wraps it, with the JSON content type already set by the timeout
// middleware and a fallback that answers like the API does.
func serveStatic(t *testing.T, dir, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusNotFound
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			status = http.StatusMethodNotAllowed
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"error":"fallback"}`))
	})
	handler := withTimeout(time.Second, staticHandler(os.DirFS(dir), fallback))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(method, target, nil))
	return response
}

func TestStaticServesTheIndexAtTheRoot(t *testing.T) {
	t.Parallel()
	response := serveStatic(t, writeFrontend(t), http.MethodGet, "/")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if body := response.Body.String(); body != testIndexHTML {
		t.Fatalf("body = %q, want the index page", body)
	}
	if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", got)
	}
	if got := response.Header().Get("Content-Security-Policy"); got != contentSecurityPolicy {
		t.Fatalf("Content-Security-Policy = %q", got)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", got)
	}
}

func TestStaticCachesHashedAssetsForGood(t *testing.T) {
	t.Parallel()
	response := serveStatic(t, writeFrontend(t), http.MethodGet, "/assets/app.abc.js")

	if response.Code != http.StatusOK || response.Body.String() != testAppJS {
		t.Fatalf("status = %d, body = %q", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); !strings.Contains(got, "javascript") {
		t.Fatalf("Content-Type = %q, want javascript", got)
	}
	if got := response.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q", got)
	}
}

func TestStaticServesOtherFilesWithoutLongCaching(t *testing.T) {
	t.Parallel()
	response := serveStatic(t, writeFrontend(t), http.MethodGet, "/favicon.svg")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, "image/svg+xml") {
		t.Fatalf("Content-Type = %q, want image/svg+xml", got)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", got)
	}
}

func TestStaticOpensTheAppAtAnyAddressWithoutAnExtension(t *testing.T) {
	t.Parallel()
	dir := writeFrontend(t)

	for _, target := range []string{"/goals", "/credits/3", "/assets"} {
		response := serveStatic(t, dir, http.MethodGet, target)
		if response.Code != http.StatusOK || response.Body.String() != testIndexHTML {
			t.Errorf("GET %s: status = %d, body = %q, want the index page", target, response.Code, response.Body.String())
		}
	}
}

func TestStaticLeavesWhatItCannotServeToTheFallback(t *testing.T) {
	t.Parallel()
	dir := writeFrontend(t)

	cases := []struct {
		method, target string
		want           int
	}{
		{http.MethodGet, "/assets/missing.js", http.StatusNotFound},
		{http.MethodGet, "/api/v1/unknown", http.StatusNotFound},
		{http.MethodGet, "/api", http.StatusNotFound},
		{http.MethodPost, "/", http.StatusMethodNotAllowed},
		{http.MethodDelete, "/assets/app.abc.js", http.StatusMethodNotAllowed},
	}
	for _, c := range cases {
		response := serveStatic(t, dir, c.method, c.target)
		if response.Code != c.want || !strings.Contains(response.Body.String(), "fallback") {
			t.Errorf("%s %s: status = %d, body = %q, want %d from the fallback", c.method, c.target, response.Code, response.Body.String(), c.want)
		}
	}
}

func TestStaticStaysInsideItsFolder(t *testing.T) {
	t.Parallel()
	dir := writeFrontend(t)

	for _, target := range []string{"/../outside.txt", "/assets/../../outside.txt", "/%2e%2e/outside.txt"} {
		response := serveStatic(t, dir, http.MethodGet, target)
		if strings.Contains(response.Body.String(), "secret") {
			t.Errorf("GET %s served a file outside the folder", target)
		}
	}
}

func TestStaticAnswersHeadWithoutABody(t *testing.T) {
	t.Parallel()
	response := serveStatic(t, writeFrontend(t), http.MethodHead, "/")

	if response.Code != http.StatusOK || response.Body.Len() != 0 {
		t.Fatalf("status = %d, body length = %d, want 200 and no body", response.Code, response.Body.Len())
	}
}

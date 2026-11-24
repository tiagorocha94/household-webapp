package server

import (
	"net/http"
	"os"
	"strings"
	"testing"

	authapi "github.com/tiagorocha94/household-webapp/internal/auth/api"
)

func TestServerServesTheFrontendNextToTheAPI(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServerOptions(t, authapi.Options{}, Options{Frontend: os.DirFS(writeFrontend(t))})

	page := serveRequest(t, srv, http.MethodGet, "/", nil, "")
	if page.Code != http.StatusOK || page.Body.String() != testIndexHTML {
		t.Fatalf("GET /: status = %d, body = %q, want the index page", page.Code, page.Body.String())
	}

	health := serveRequest(t, srv, http.MethodGet, "/api/v1/healthz", nil, "")
	if health.Code != http.StatusOK || !strings.HasPrefix(health.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("GET /api/v1/healthz: status = %d, Content-Type = %q", health.Code, health.Header().Get("Content-Type"))
	}

	unknown := serveRequest(t, srv, http.MethodGet, "/api/v1/nothing", nil, "")
	if unknown.Code != http.StatusNotFound || !strings.HasPrefix(unknown.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("GET /api/v1/nothing: status = %d, Content-Type = %q, want a JSON 404", unknown.Code, unknown.Header().Get("Content-Type"))
	}

	// The API still needs a session.
	protected := serveRequest(t, srv, http.MethodGet, "/api/v1/credits", nil, "")
	if protected.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/v1/credits without a session: status = %d, want 401", protected.Code)
	}
}

func TestServerWithoutAFrontendKeepsAJSON404AtTheRoot(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)

	response := serveRequest(t, srv, http.MethodGet, "/", nil, "")
	if response.Code != http.StatusNotFound || !strings.HasPrefix(response.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("status = %d, Content-Type = %q, want a JSON 404", response.Code, response.Header().Get("Content-Type"))
	}
}

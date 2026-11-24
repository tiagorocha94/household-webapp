package server

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	authservice "github.com/tiagorocha94/household-webapp/internal/auth/service"
	authstore "github.com/tiagorocha94/household-webapp/internal/auth/store"
)

// TestEnrolRouteMatchesTheSpec enrols through the whole server and compares the
// answer with openapi.yaml, then checks the enrolment errors.
func TestEnrolRouteMatchesTheSpec(t *testing.T) {
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
	insertTestUser(t, db, "alice", "alice-password")
	code, _, err := authservice.New(authstore.New(db)).AddDevice("alice", "phone")
	if err != nil {
		t.Fatalf("AddDevice() error = %v", err)
	}

	body, _ := json.Marshal(map[string]string{"code": code})
	response := serveRequest(t, srv, http.MethodPost, "/api/v1/devices/enrol", nil, string(body))
	assertAPIStatus(t, response, http.StatusOK)
	got := decodeAPI(t, response)
	checkShape(t, spec, dig(spec, "paths", "/devices/enrol", "post", "responses", "200", "content", "application/json", "schema"), got, "POST /devices/enrol")
	if m, _ := got.(map[string]any); m["username"] != "alice" || m["deviceName"] != "phone" || m["token"] == "" {
		t.Errorf("enrol answer = %v, want a token for alice's phone", got)
	}

	// The code is used up, and a session is not needed to try one.
	assertAPIStatus(t, serveRequest(t, srv, http.MethodPost, "/api/v1/devices/enrol", nil, string(body)), http.StatusUnauthorized)
	assertAPIStatus(t, serveRequest(t, srv, http.MethodPost, "/api/v1/devices/enrol", nil, `{"code":""}`), http.StatusBadRequest)

	// Only POST is allowed.
	response = serveRequest(t, srv, http.MethodGet, "/api/v1/devices/enrol", nil, "")
	assertAPIStatus(t, response, http.StatusMethodNotAllowed)
	if allow := response.Header().Get("Allow"); !strings.Contains(allow, "POST") {
		t.Errorf("Allow = %q, want it to list POST", allow)
	}
}

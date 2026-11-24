package server

import (
	"net/http"
	"slices"
	"testing"

	authapi "github.com/tiagorocha94/household-webapp/internal/auth/api"
	"github.com/tiagorocha94/household-webapp/internal/modules"
)

func TestInfoListsEveryModuleByDefault(t *testing.T) {
	t.Parallel()

	srv, _ := newTestServer(t)
	response := serveRequest(t, srv, http.MethodGet, "/api/v1/info", nil, "")
	assertAPIStatus(t, response, http.StatusOK)

	got := decodeAPI(t, response).(map[string]any)["modules"]
	list, ok := got.([]any)
	if !ok {
		t.Fatalf("modules = %#v, want a list", got)
	}
	names := make([]string, 0, len(list))
	for _, item := range list {
		names = append(names, item.(string))
	}
	if !slices.Equal(names, modules.All()) {
		t.Errorf("modules = %v, want %v", names, modules.All())
	}
}

func TestInfoListsOnlyTheEnabledModules(t *testing.T) {
	t.Parallel()

	set, err := modules.NewSet([]string{modules.Cashflow})
	if err != nil {
		t.Fatalf("NewSet() error = %v", err)
	}
	srv, _ := newTestServerOptions(t, authapi.Options{}, Options{Modules: set})
	response := serveRequest(t, srv, http.MethodGet, "/api/v1/info", nil, "")
	assertAPIStatus(t, response, http.StatusOK)

	got := decodeAPI(t, response).(map[string]any)["modules"].([]any)
	if len(got) != 1 || got[0] != modules.Cashflow {
		t.Errorf("modules = %v, want [cashflow]", got)
	}
}

// A module that is not enabled has no routes at all, so its paths are not
// found even for someone who is not signed in; an enabled one asks for a session.
func TestDisabledModulesHaveNoRoutes(t *testing.T) {
	t.Parallel()

	set, err := modules.NewSet([]string{modules.Cashflow})
	if err != nil {
		t.Fatalf("NewSet() error = %v", err)
	}
	srv, _ := newTestServerOptions(t, authapi.Options{}, Options{Modules: set})

	tests := []struct {
		path string
		want int
	}{
		{"/api/v1/cashflow/months", http.StatusUnauthorized},
		{"/api/v1/goals/accounts", http.StatusNotFound},
		{"/api/v1/savings/accounts", http.StatusNotFound},
		{"/api/v1/credits", http.StatusNotFound},
		{"/api/v1/investments/entities", http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()

			assertAPIStatus(t, serveRequest(t, srv, http.MethodGet, tt.path, nil, ""), tt.want)
		})
	}
}

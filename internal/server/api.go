package server

import (
	"net/http"
	"strings"

	"github.com/tiagorocha94/household-webapp/internal/api"
	authapi "github.com/tiagorocha94/household-webapp/internal/auth/api"
)

// syncAPIVersion is the version of the sync protocol (POST /sync/cashflow). It
// goes up when a change would break a phone that was built for the old one.
const syncAPIVersion = 1

// handle wires a read-only JSON endpoint behind authentication.
func (s *Server) handle(action string, fn func(*http.Request) (any, error)) http.Handler {
	return authapi.RequireAuth(s.authService, s.logger, s.adaptJSON(action, fn))
}

// handleReadable is handle for the read-only routes that an enrolled phone may
// read as well as the web interface: it takes the session cookie or the device
// token (ADR 012).
func (s *Server) handleReadable(action string, fn func(*http.Request) (any, error)) http.Handler {
	return s.authHandler.RequireSessionOrDevice(s.adaptJSON(action, fn))
}

// handleHealth answers 200 while the process is serving requests.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	_ = api.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleInfo reports which build is running and which sync protocol it speaks.
func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	_ = api.WriteJSON(w, http.StatusOK, map[string]any{"version": s.version, "syncApi": syncAPIVersion})
}

// handleNotFound answers every unmatched request with a JSON error: 405 with
// an Allow header when the path exists for other methods, 404 otherwise. The
// mux cannot do this itself, because the catch-all route matches every method.
func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	if allowed := s.allowedMethods(r); len(allowed) > 0 {
		w.Header().Set("Allow", strings.Join(allowed, ", "))
		api.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}
	api.WriteError(w, http.StatusNotFound, "Not Found")
}

// allowedMethods lists the methods that have a real route for r's path.
func (s *Server) allowedMethods(r *http.Request) []string {
	var allowed []string
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		probe := r.Clone(r.Context())
		probe.Method = method
		if _, pattern := s.router.Handler(probe); pattern != "" && pattern != "/" {
			allowed = append(allowed, method)
		}
	}
	return allowed
}

// adaptJSON turns a handler that returns (body, error) into an
// http.HandlerFunc: on success it writes body as JSON with status 200, on
// error it maps the error with api.ErrorStatus and writes
// {"error": "..."}.
func (s *Server) adaptJSON(action string, fn func(*http.Request) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := fn(r)
		if err != nil {
			status, message := api.ErrorStatus(s.logger, action, err)
			api.WriteError(w, status, message)
			return
		}
		if err := api.WriteJSON(w, http.StatusOK, body); err != nil {
			s.logger.Error("encode json response", "action", action, "error", err)
		}
	}
}

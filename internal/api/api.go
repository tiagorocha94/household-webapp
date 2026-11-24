// Package api provides what the HTTP handlers share: JSON response writers,
// the error-to-status mapping and path parameter parsing.
package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
)

// ErrorStatus maps err to an HTTP status and a message that is safe to show
// the user. Unrecognized errors are logged to log (with action for context)
// and mapped to 500 without leaking details.
func ErrorStatus(log *slog.Logger, action string, err error) (int, string) {
	switch {
	case errors.Is(err, apperr.ErrBadRequest):
		return http.StatusBadRequest, "Bad Request"
	case errors.Is(err, apperr.ErrNotFound):
		return http.StatusNotFound, "Not Found"
	default:
		log.Error("request failed", "action", action, "error", err)
		return http.StatusInternalServerError, "Internal Server Error"
	}
}

// PathID parses the {id} path value as a positive integer. A malformed value
// returns apperr.ErrBadRequest; field names the ID in the error message.
func PathID(r *http.Request, field string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid %s: %w", field, apperr.ErrBadRequest)
	}
	return id, nil
}

// WriteJSON encodes body into a buffer first, so an encoding failure never
// produces a half-written response: the client gets a 500 instead, and the
// encoding error is returned for the caller to log. Callers that write fixed,
// always-encodable bodies may ignore it.
func WriteJSON(w http.ResponseWriter, status int, body any) error {
	var buf bytes.Buffer
	encodeErr := json.NewEncoder(&buf).Encode(body)
	if encodeErr != nil {
		status = http.StatusInternalServerError
		buf.Reset()
		buf.WriteString("{\"error\":\"Internal Server Error\"}\n")
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	// A failed write means the client went away; there is nothing to do.
	_, _ = buf.WriteTo(w)
	return encodeErr
}

// WriteError writes {"error": message} with the given status.
func WriteError(w http.ResponseWriter, status int, message string) {
	_ = WriteJSON(w, status, map[string]string{"error": message})
}

// Package apperr defines sentinel errors shared across every module's
// service layer, so the HTTP layer can map failures to consistent status
// codes without each module inventing its own error vocabulary.
package apperr

import "errors"

// ErrBadRequest indicates the request itself was malformed (for example an
// unparsable ID).
var ErrBadRequest = errors.New("bad request")

// ErrNotFound indicates the requested resource does not exist.
var ErrNotFound = errors.New("not found")

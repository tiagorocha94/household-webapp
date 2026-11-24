# 008 Standard library router and plain middleware

Status: accepted

Context: the API is small, with a single write endpoint, and the project prefers the standard
library ([conventions](../development/code-style.md#standard-library-first)).

Decision: route with the Go 1.22 `ServeMux`, and write cross-cutting concerns
(request ID, access log, security headers, body limit, timeout, panic
recovery) as plain `func(http.Handler) http.Handler` in `internal/server`.
Handlers are plain `http.Handler`.

Consequences: no router dependency, and the 405 handling is done by hand.
Revisit a router such as chi if write endpoints multiply, or once there are
many middlewares or per-route middleware starts to repeat. Switching touches
`routes()`, the router field in `Server`, and `api.PathID` if path values do
not populate `r.PathValue`.

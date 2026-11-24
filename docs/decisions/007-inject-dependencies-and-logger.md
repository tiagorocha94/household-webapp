# 007 Dependencies, including the logger, are injected

Status: accepted

Context: package-level state such as the default `slog` logger hides what a
component needs and makes its output hard to control in tests.

Decision: `cmd/server` builds the logger (and the rest of the wiring) and
passes it explicitly: to migrations, the seed runner and modules, the server
and handlers. Nothing outside `cmd/server` calls the package-level `slog`
functions; the linter (`forbidigo`) checks this. Tests pass `testutil.Logger()`.

Consequences: signatures carry a `*slog.Logger` where something is logged.
Logging style rules are in
[conventions.md](../development/code-style.md#logging).

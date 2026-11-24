# Architecture

household-app is a monolithic Go server that exposes a read-only JSON API under
`/api/v1` (the one exception is the [cashflow sync](../modules/cashflow/sync.md) endpoint for enrolled phones). It renders no HTML; it serves the frontend, which is built into the binary (`web/dist`). Data is maintained in CSV files; the server loads
them into SQLite on startup and exports them back after sync. The frontend is a separate program in `web/` ([frontend.md](frontend.md)).

| Layer    | Technology                    |
|----------|-------------------------------|
| Language | Go 1.27 (stdlib first)        |
| Database | SQLite via modernc.org/sqlite |
| API      | `net/http` with JSON bodies   |
| Auth     | bcrypt (golang.org/x/crypto)  |

## Startup

`cmd/server` runs migrate, import, sync and export, then serves. Any failure
aborts startup, so the server never accepts requests on half-prepared data.
Details and CSV rules: [csv-seed.md](csv-seed.md).

## Modules

Each feature lives in its own package under `internal/` ([ADR 006](../decisions/006-layered-feature-modules.md)).
The module root keeps the domain types and pure logic; the layers sit in sub-packages:

- `model.go` (root): domain types and constants (in `goals` split over `accounts.go`, `goals.go`, `movements.go`)
- `store/`: database access (queries, views)
- `service/`: business logic and authorization
- `api/`: JSON handlers and wire types
- `seed.go` (root): the module's CSV import, export and sync (`SeedModule`)

Sub-packages import the root for the types, never the other way round, so
there are no import cycles. Wiring sites alias them by module
(`creditsstore`, `creditsservice`, `creditsapi`, and so on).

All six modules (`auth`, `goals`, `credits`, `investments`, `savings`, `cashflow`) are split this way. The `overview` package is the exception: it owns no tables, so it has only a service that reads the other modules' services and an `api/` handler ([Home overview](../modules/home/overview.md)). Add
`<entity>_store.go`-style splitting inside a sub-package once a file passes
roughly 250-300 lines. Module-specific notes:

- `auth`: the root also holds the request-context helpers (`WithUser`,
  `UserFromContext`) and the user lookups the CSV imports use; `RequireAuth`
  lives in `auth/api`.
- `goals`: the root also holds the allocation `Schedule` and the types the
  store and service share (`GoalPlan`, `MonthActual`, `ProjectionMonth`).

Dependencies flow handler, service, store, and `cmd/server/main.go` wires
them.

Shared packages:

| Package | Provides |
|---------|----------|
| `apperr` | sentinel errors |
| `api` | JSON writers, error-to-status mapping, path parsing |
| `config` | `config.yaml` loading and defaults |
| `csvdata` | CSV reading and writing, cent parsing, mirror helpers |
| `database` | connection, migrations, the `Querier` interface |
| `money` | the `Cents` type and cent-exact splitting |
| `month` | `YYYY-MM` validation, counting and arithmetic |
| `seed` | the startup orchestrator and the `Module` interface ([ADR 005](../decisions/005-modules-own-their-seed-logic.md)) |
| `testutil` | temporary migrated databases and a discard logger for tests |

`cmd/server` creates the logger and passes it to whatever logs
([ADR 007](../decisions/007-inject-dependencies-and-logger.md)). The router and
middleware are plain standard library ([ADR 008](../decisions/008-standard-library-router.md)).

## Request flow

1. Middleware wraps the router, outermost first: `requestID` (accepts or
   generates `X-Request-Id`), `accessLog` (one line per request),
   `securityHeaders` (`nosniff`, `Cache-Control: no-store`), `limitBody`
   (1 MiB), `withTimeout` (10 s, JSON 503) and `recoverPanics` (a panic
   becomes a JSON 500). The router then matches the route: an unmatched path
   gets a JSON 404, and a known path with the wrong method gets a JSON 405
   with an `Allow` header. When the frontend is built in, paths outside `/api`
   are served from it (unknown extensionless paths get `index.html`,
   with a Content-Security-Policy), and the 404 and 405 stay JSON for the rest.
2. `RequireAuth` checks the `session_token` cookie and injects the user,
   or answers 401.
3. The handler parses path values, calls the service and returns
   `(any, error)`. It never writes the response or decides status codes.
   Malformed IDs return `apperr.ErrBadRequest`.
4. The service holds the business logic and returns `apperr.ErrNotFound` for a
   missing resource. Any logged-in user may read everything
   ([ADR 010](../decisions/010-household-data-is-shared.md)).
5. The store queries tables and views. It returns `nil, nil` for a missing row
   and real errors for real failures.
6. `Server.adaptJSON` writes the body with status 200, or maps the error with
   `api.ErrorStatus` to `{"error": "..."}`. Unrecognized errors are
   logged and answered as 500 without details.

| Situation | Status |
|-----------|--------|
| Malformed or non-positive ID | 400 |
| No valid session | 401 |
| Unknown resource or path | 404 |
| Wrong method on a known path | 405 |
| Too many failed logins | 429 |
| Anything else | 500 |
| Handler took over 10 s | 503 |

Login and logout write directly because they manage the session cookie (see
[authentication.md](authentication.md)).

Wire types (DTOs with explicit `json` tags) live next to each module's
handlers and are documented in [openapi.yaml](openapi.yaml) ([ADR 004](../decisions/004-json-api-only-interface.md)). A server test
compares every documented `GET` response with the spec.

## Database

Conventions are in [database.md](database.md). In short:
`foreign_keys`, `busy_timeout` and WAL are set through the DSN so they apply to
every pooled connection, migrations run on startup and are tracked in
`schema_migrations`, and SQL views compute goal balances and credit summaries.

## Testing

`go test ./...`. Tests use isolated temporary SQLite databases with the
embedded migrations applied (`internal/testutil`), so they exercise the 
same constraints and views as the application.

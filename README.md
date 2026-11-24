# household-webapp

A personal application for managing household finances. It serves a read-only
JSON API for shared goal allocation, credit contract/payment tracking, an investment portfolio and a cash-flow ledger. Data
is maintained in CSV files; the frontend lives in `web/`.

## Quick start

Try it on an invented household first (it needs Go and Node):

```bash
make demo    # build the UI and server, then serve a demo household on :8081
```

Log in as `alex` or `sam` with the password `demo`.

To use your own data, keep it in a separate private repository (a `seed/` folder of CSV
files plus a `config.yaml` whose `seed.dir` points at it), never in this one
([ADR 017](docs/decisions/017-seed-data-in-a-private-repo.md)). Passwords in
`users.csv` must be [bcrypt hashes](docs/architecture/csv-seed.md#userscsv). Build the
server with `make build`, then start it from your data repository:

```bash
cd ../household-data && ./run.sh ../household-webapp/household-app
```

`make run` starts the server on this repository's `config.yaml` (its `seed.dir` is `seed`,
which does not exist here, so use it only after pointing `seed.dir` at your data):

```bash
make run     # build the UI and server, prepare the data, then serve on :8080
make reset   # delete the database first, then run
```

Startup migrates the database, mirrors the CSV files into it, runs sync and
exports the files again. See [CSV seed and export](docs/architecture/csv-seed.md);
use `household-app -dry-run` to preview the changes without writing anything.

The UI is embedded in the server binary. `make run` installs its Node packages
when needed. Without Node or Make, `go run ./cmd/server/` serves only the API.

Built binaries report the latest tag plus any later commit count and hash, such
as `v0.3.0-4-g1a2b3c4`. Read it with `household-app -version` or
`GET /api/v1/info`. Releases are described in
[releasing.md](docs/development/releasing.md).

Enrol phones with:

```bash
household-app device add <user> <name>
household-app device list
household-app device revoke <name>
```

See [device authentication](docs/architecture/authentication.md#devices).

Log in with a user from your `users.csv` (the example is made up):

```bash
curl -c cookies.txt -X POST localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"alice-example"}'
curl -b cookies.txt localhost:8080/api/v1/goals/accounts
```

## Configuration

`config.yaml` (override with `make run CONFIG=my-config.yaml`):

```yaml
server:
  host: 127.0.0.1
  port: 8080
  secure_cookies: false  # true when served over HTTPS
database:
  path: household.db
seed:
  dir: seed
log:
  level: info
# modules: [cashflow, goals]  # which features run; leave it out to run all of them
```

`modules` chooses the features the server runs: `cashflow`, `goals`, `savings`, `credits`, `investments`. A module that is left out has no API routes (they answer 404), its CSV files are neither read nor written, and the web app hides its pages. `GET /api/v1/info` lists the modules that run. An unknown name stops the server at start.

The server listens on localhost only. To reach it from another machine set `server.host` to `0.0.0.0`, but note there is no TLS. If you put it behind a TLS proxy, set `server.secure_cookies: true`.

## Make commands

| Command      | Description                                   |
|--------------|-----------------------------------------------|
| `make run`   | Build the web interface and the server, then start it (it prepares the data first) |
| `make reset` | Delete the database, then run                 |
| `make clean` | Delete the database                           |
| `make build` | Build only the `household-app` binary with its version set (embeds the last web build) |
| `make test`  | Run the tests in random order                 |
| `make check` | Run the local format, build, vet and test checks |
| `make lint`  | Run golangci-lint and govulncheck locally (needs network) |
| `make lint-code` | Run golangci-lint, as relevant pushes and version tags do (needs network) |
| `make vulncheck` | Run govulncheck, as the weekly, manual and version-tagged security workflow does (needs network) |
| `make web-build` / `web-check` / `web-types` | The frontend in `web/`, see [docs/architecture/frontend.md](docs/architecture/frontend.md) (needs Node) |

## Tech stack

Go (stdlib first), SQLite via modernc.org/sqlite (pure Go, no CGO), bcrypt from
golang.org/x/crypto.

## Documentation and development

Start at [docs/README.md](docs/README.md). Verify changes with
`make check`.

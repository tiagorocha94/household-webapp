# AGENTS.md

Guidance for AI agents working on household-app: a Go server exposing a
read-only JSON API under `/api/v1` (plus the cashflow sync endpoint for enrolled phones), backed by SQLite. Data lives in CSV files
that are imported, synced and exported on every start. The frontend is a Svelte
app in `web/` ([ADR 009](docs/decisions/009-frontend-svelte-spa.md)).

## Read first

- [README.md](README.md): quick start and `make` commands
- [docs/architecture/overview.md](docs/architecture/overview.md): module layout, layers, request flow, error mapping
- [docs/architecture/csv-seed.md](docs/architecture/csv-seed.md): startup lifecycle, CSV rules and how modules plug into `seed`
- [docs/decisions/](docs/decisions/): why the project is built this way
- [docs/architecture/frontend.md](docs/architecture/frontend.md) and [web/AGENTS.md](web/AGENTS.md): UI work only
- [docs/architecture/database.md](docs/architecture/database.md): naming, money, months, constraints
- `docs/modules/<module>/overview.md`: module behavior (source of truth)
- [docs/architecture/openapi.yaml](docs/architecture/openapi.yaml): wire format
- [docs/development/backlog.md](docs/development/backlog.md): known limitations and ideas
- [docs/development/](docs/development/workflow.md): workflow, code style, changelog and release rules

## Rules

- **Code**: Uber Go Style Guide, stdlib first, doc comment on every exported
  member, errors wrapped and never ignored, an injected `*slog.Logger` for
  logging. Details in [docs/development/code-style.md](docs/development/code-style.md).
- **Data**: migrations change schema only. Data changes go through the CSV
  files. Never edit a shipped migration: add a new one that
  moves the existing values ([ADR 003](docs/decisions/003-fix-forward-migrations.md)).
  Money is integer cents everywhere.
- **Commits**: conventional format, atomic, each compiles on its own, work on
  `main`. Update `docs/` when behavior changes and `CHANGELOG.md` only for
  user-facing changes (see [docs/development/changelog-guidelines.md](docs/development/changelog-guidelines.md)).

## Frontend

UI work stays in `web/`; Go work stays out of it. An API change also updates
`docs/architecture/openapi.yaml` and the generated types in `web/` in the same commit.

## Do not

- Add frameworks, libraries or architectural patterns without discussion.
- Add write endpoints (data changes go through the CSV files). The one exception is the cashflow sync endpoint for enrolled devices, described in [cashflow sync](docs/modules/cashflow/sync.md) and decided in [ADR 011](docs/decisions/011-devices-write-cashflow-movements.md).
- Add user registration or password management.
- Create separate branches.

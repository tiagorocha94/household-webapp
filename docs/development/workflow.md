# Workflow

How to work on the project. Code rules are in [code-style.md](code-style.md),
the design in [architecture](../architecture/overview.md).

## Before changing anything

1. Read [AGENTS.md](../../AGENTS.md) and the `docs/modules/<module>/` pages of
   the feature you are changing.
2. Look at the code in related packages and follow its patterns.
3. Check that the project is healthy: `make check`.

## Making a change

1. Follow the module layout in [the overview](../architecture/overview.md#modules).
2. Keep each commit to one logical change; no forward references to code that
   does not exist.
3. If an API response changed (route, field, type), update
   [openapi.yaml](../architecture/openapi.yaml) and regenerate the frontend
   types (`make web-types`) in the same commit. `TestOpenAPIMatchesResponses`
   fails when property names drift or a documented GET is missing from its
   `paths` table, or when a value's type, an enum value or a null differs from
   the spec.
4. Update the relevant `docs/` page when behavior changes, and `CHANGELOG.md`
   when a user can see the difference ([guidelines](changelog-guidelines.md)).
5. Verify: `make check` (and `make lint` before pushing; `make web-check` for
   frontend changes). CI also runs the browser tests in `web/e2e` against the
   built server on the demo household, once for each set of modules; locally,
   after `make web-build build`, run `npm run e2e` in `web/` (the first time
   `npx playwright install chromium`).

## Commits

- Conventional format `type: description` (feat, fix, chore, docs, refactor,
  test, style, perf, ci, build).
- Each commit is atomic and compiles on its own.

## Pull requests

`main` only changes through a pull request, which must pass the CI checks and is
squash merged, so the title of the pull request becomes the commit on `main` and
follows the conventional format above. Branches are named `type/short-topic`
(for example `feat/optional-modules`) and are deleted after the merge. Keep a
pull request to one unit of work with its tests and docs. Release tags
(`vX.Y.Z`) are created by the maintainer on `main` after the release steps in
[releasing.md](releasing.md).

## Adding a feature module

1. Create a package under `internal/` with the module name, laid out as in
   [the overview](../architecture/overview.md#modules): `model.go` and
   `seed.go` in the root, and `store/`, `service/` and `api/` sub-packages.
2. Add a migration file in `internal/database/migrations/sql/` for new tables.
3. Add CSV import, export and sync as a `SeedModule` in
   `internal/<module>/seed.go` (it satisfies `seed.Module`) and list it in
   `newSeedRunner` in `cmd/server/main.go`.
4. Add the module's CSV files to `demo/` (invented data, so the demo and the tests cover the module). Real data never goes in this repository.
5. Wire the dependencies in `cmd/server/main.go` and register the read-only
   routes in `internal/server/server.go`.
6. Create `docs/modules/<module>/` with `overview.md`, `schema.md` and `csv.md`,
   and list them in [docs/README.md](../README.md).
7. Update [openapi.yaml](../architecture/openapi.yaml), add the new `GET`
   routes to the `paths` table in `internal/server/openapi_test.go`, and update
   `CHANGELOG.md`.

## Adding a migration

Create `NNNN_description.sql` with the next number in
`internal/database/migrations/sql/`. Migrations change the schema only, never
data except moving existing values to a new shape, and run on the next start.
Never edit a migration that has shipped; add a new one
([ADR 003](../decisions/003-fix-forward-migrations.md)).

## Working with data

Edit the CSV files under `seed/` and restart
([csv-seed.md](../architecture/csv-seed.md)).

## Running locally

1. `make reset` starts from a clean database and the CSV data (only when you
   want to start over; a schema change never needs it).
2. `make run`, then log in and call the API as shown in the
   [README](../../README.md#quick-start).

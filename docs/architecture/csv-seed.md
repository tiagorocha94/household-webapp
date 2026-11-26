# CSV Import, Sync and Export

## Overview

Migrations handle schema. CSV files handle data. CSV is the durable store
between runs: on every server start the data is migrated, imported, synced and
exported back. There are no separate commands.

```text
migrate -> import (mirror CSV) -> sync -> export -> serve
```

The directory is `seed.dir` in `config.yaml` (default `seed`). It holds your real
data, which lives in its own private repository, never in this one
([ADR 017](../decisions/017-seed-data-in-a-private-repo.md)); `/seed/` is
ignored by git here so it cannot be committed by mistake. For smoke and
end-to-end tests use the invented household in `demo/` (see below). The server
rewrites the files it exports, so tests and `make demo` work on a copy. Any failing step
aborts startup. Export only runs after a successful import, so a malformed CSV
can never overwrite good files. The server is read-only, so nothing is exported
on shutdown. `make reset` deletes the database and starts fresh from the CSVs.
Why: [ADR 001](../decisions/001-csv-is-the-durable-store.md).

## Who Does What

`internal/seed` is only the orchestrator. It reads each module's CSV files and
checks their headers, hands the rows to the module inside a transaction,
runs each module's sync, and writes each module's export back. What the files
mean lives in the modules, which satisfy the small `seed.Module` interface
(`Name`, `Files`, `Import`, `Sync`, `Export`, and optionally `AfterExport`, called
once the module's files are written) from their own `seed.go`: `auth`
for users, `goals` and `credits`. The runner passes each module the logger it
was given ([ADR 007](../decisions/007-inject-dependencies-and-logger.md)). `cmd/server/main.go` lists them, and the
shared CSV and mirror helpers are in `internal/csvdata`. A new
module does not touch `seed` (see
[ADR 005](../decisions/005-modules-own-their-seed-logic.md)).

## File Processing Order

The modules run one after another in the order `cmd/server/main.go` lists
them. Apart from `users.csv`, which must come first because investments,
savings and cashflow name users, they do not depend on each other's data, so the order does
not matter for the result. They are not imported in parallel: SQLite lets one writer in at a
time, so parallel imports would only wait for each other and could fail with
"database is locked".

1. `users.csv`, `devices.csv`
2. `goal_accounts.csv`, `goal_account_goals.csv`,
   `goal_account_movements.csv` ([goals](../modules/goals/csv.md))
3. `credit_contracts.csv`, `credit_payments.csv` ([credits](../modules/credits/csv.md))
4. `investment_entities.csv`, `investment_instruments.csv`,
   `investment_contributions.csv`, `investment_snapshots.csv`
   ([investments](../modules/investments/csv.md))
5. `savings_accounts.csv`, `savings_movements.csv`, `savings_snapshots.csv`
   ([savings](../modules/savings/csv.md))
6. `cashflow_budget_groups.csv`, `cashflow_categories.csv`, `cashflow_movements.csv`
   ([cashflow](../modules/cashflow/csv.md))

Column formats live with each module.

### users.csv

| Column   | Description                |
|----------|----------------------------|
| username | Unique username            |
| password | A bcrypt hash (`$2a$`, `$2b$` or `$2y$`). A plain password is refused |

Make a hash with `htpasswd -nbBC 10 "" 'the-password' | tr -d ':\n'`, or any
bcrypt tool, and paste the result (it starts with `$2y$`) into the column. A
value that is not a valid hash, a plain password included, stops the import and
names the row, so the repository never holds a password. A changed hash applies
on the next start. The file is never rewritten. A hash never needs quoting in
the CSV.

### devices.csv

Written by the server, not by hand. One row per enrolled phone.

| Column     | Description                                        |
|------------|----------------------------------------------------|
| username   | The user the phone belongs to                      |
| name       | The device name                                    |
| token_hash | SHA-256 hash of the device token, never the token  |
| enrolled_at | Unix seconds when it was enrolled                 |

## Import Behavior

The import is a **mirror** of the CSV files:

- A file that is **absent** is skipped; its tables are not touched.
- A file that **exists** is authoritative. Its rows are inserted or updated, and
  rows in the database that are not in the file are removed. A file with only a
  header clears its table.
- **Users are never removed**: removing a user from `users.csv` does nothing,
  and does not end their sessions. To drop a user, run `make reset`, which
  recreates the database from the CSV files.
- **Devices are only restored**: `devices.csv` adds the enrolled phone of a user
  who has no device row, which is what a rebuilt database looks like. It never
  removes or changes a device.
- Importing the same files twice is a no-op: no row is written (the checks
  run in the database, so unchanged rows cost one statement each). Parent rows (accounts, goals,
  contracts) keep their IDs and are only updated when a column changed. Child
  rows (movements, payments) are only replaced for a parent whose rows differ
  from the database. A user's password hash is kept when the password in the
  CSV still matches it.
- Each module (users, goals, credits) is imported in **one transaction**, and
  every file is read and header-checked before anything is written. A failure
  rolls the module back.
- Malformed rows are errors, not skipped, because skipping a row would delete
  its data. References must resolve: a movement naming an unknown goal fails
  the import.
- Removing an account, goal or contract also removes its children through the
  foreign keys.

Money columns are raw non-negative integer cents under `*_cents` headers (see
[database.md](database.md#money-values)). Decimal points,
commas, signs, overflow and stale headers are rejected.

## Demo data

`demo/` is an invented household (two users, `alex` and `sam`, both with the password
`demo`) with every module filled in: cashflow for four months, a goals account with three
goals, a car loan, two investments and two savings accounts. It is the data the tests
and `make demo` start from, never anyone's real data. `make demo` copies it to
`.demo/` (ignored by git) and serves it on port 8081 with its own database, because the
server rewrites the files it exports. `TestDemoDataStartsAndIsRepeatable` checks that it
imports, syncs and exports, and that starting again changes nothing, so a change to
the CSV format that the demo data does not follow fails the tests. The movements are
dated 2026-08 to 2026-11, so a later month shows as empty.

## Comparing two runs

`scripts/snapshot-api.sh BASE_URL OUT_DIR` logs in (with `HOUSEHOLD_USER` and
`HOUSEHOLD_PASSWORD`) and saves what every list and detail endpoint answers as
sorted JSON, one file each. Run it against two servers (or two builds on the same
data) and compare with `diff -r`: no difference means the same answers. It stops
at the first answer that is not 200, so it also serves as a smoke test. The overview
depends on the day, so compare runs made on the same day. To compare the CSV files
too, start each server on its own copy of the data directory and `diff -r` the
copies.

## Dry run

`household-app -dry-run` (or `go run ./cmd/server -dry-run`) reports what
starting the server would change, then exits without serving. It runs the
migrations, import, sync and export on a **copy** of the database in a
temporary directory, so neither the database nor the CSV files are written. The
report lists:

- the migrations that would run;
- per table, the rows that the import and sync would add or remove (a changed
  row shows as one removed and one added), up to ten of each, without IDs and
  timestamps, and with password hashes shown as a short fingerprint;
- the CSV files the export would create or rewrite, with their row counts.

Errors that would stop startup, such as a bad header, stop the dry run the same
way. With nothing to change it prints `Nothing would change.`

## Export Behavior

- Overwrites existing files in the target directory.
- Exports every module's files. `users.csv` is left untouched so an export
  cannot replace working passwords with placeholders; `devices.csv` is written
  from the database.
- Row order is deterministic, so unchanged data exports identically.

Each file is written to a temporary file and renamed, so a crash cannot 
truncate it. `seed/` is under version control, so every start that changes data
shows up as a diff and a push is the backup.

## Sync Behavior

`sync` runs after every import:

- Creates missing goal allocations in cent-exact monthly amounts. A running
  server does the same once a day ([background jobs](overview.md#background-jobs)).
- Settles the goals that were completed by hand: records the closing movement
  (a release of the leftover or a cover of the overspend) and the closing
  balance, in one transaction per goal. No goal completes on its own.

Each module's sync receives the current time from the orchestrator. Only the
goals module has sync work today. Credit status is derived by a database view,
not by sync, and credit payments are actual transactions that are never
generated. Sync is idempotent.

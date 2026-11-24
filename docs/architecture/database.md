# Database Conventions

## Table Naming

All tables are prefixed with their module name to avoid naming collisions across modules.

### Pattern

```
<module>_<entity>
```

For nested entities within a module:

```
<module>_<parent>_<entity>
```

### Examples

| Table                      | Module | Description                     |
|----------------------------|--------|---------------------------------|
| `users`                    | core   | User accounts (no module prefix)|
| `user_sessions`            | core   | Login sessions (hashed tokens)  |
| `devices`                  | core   | Write-enabled phones (hashed secrets) |
| `schema_migrations`        | core   | Migration tracking              |
| `goal_accounts`            | goals  | Goal accounts                   |
| `goal_account_goals`       | goals  | Goals within an account         |
| `goal_account_movements`   | goals  | Ledger movements                |
| `credit_contracts`         | credits | Credit contracts               |
| `credit_payments`          | credits | Credit payment history         |

### Rules

- The `users`, `user_sessions`, `devices` and `schema_migrations` tables are core tables without a module prefix.
- All other tables must be prefixed with the module name.
- Use singular nouns for entity names (`goal`, not `goals`) in compound names, except when the table itself represents a collection (e.g. `goal_accounts`).
- Junction tables use both parent names.
- Use underscores for word separation.

## Column Naming

- Use `snake_case` for all column names.
- Foreign keys are named `<referenced_table_singular>_id` (e.g. `user_id`, `account_id`, `goal_id`).
- Timestamps use `created_at` and `updated_at`.
- Status columns use `status` with text values.
- Month columns storing YYYY-MM use `month` (or `start_month`, `end_month` for ranges).
- Date columns storing a full day (YYYY-MM-DD) use `date`; see [Date Values](#date-values).

### Month Values

Calendar months are normalized across every module as follows:

- Store them as non-null `TEXT` in canonical `YYYY-MM` form. Use nullable
  month columns only when the domain value itself is optional.
- Use zero-padded month numbers (`01` through `12`). Do not store complete
  dates, display-formatted values, or unpadded values in a month column.
- Validate months where data enters (the CSV import) with `month.Validate`
  so a bad row fails with a clear message. Go code parses, counts and
  calculates months only through `internal/month`, never with its own
  `time.Parse("2006-01", ...)` or `AddDate`.
- Enforce the same rule in SQLite so CSV imports and other direct writes cannot
  bypass it. New schemas must use the complete constraint below; checking the
  `GLOB` alone is insufficient because it accepts values such as `2026-00` and
  `2026-19`.

```sql
CHECK (
    month GLOB '2[0-1][0-9][0-9]-[0-1][0-9]'
    AND CAST(SUBSTR(month, 6, 2) AS INTEGER) BETWEEN 1 AND 12
)
```

For a nullable month, wrap both checks in `month IS NULL OR (...)`. Month
ranges additionally require `CHECK (end_month >= start_month)`. Canonical
`YYYY-MM` values sort chronologically as text, so comparisons and ordering do
not require conversion.

### Date Values

A few tables need a full day rather than a month (the cashflow movements).
Such a column is named `date`, never `month`, so the rule above still holds:

- Store it as non-null `TEXT` in canonical `YYYY-MM-DD` form, zero-padded.
- Validate it where data enters (the CSV import) with `month.ValidateDay`,
  which also rejects impossible days such as `2026-02-30`. Go parses and
  compares days only through `internal/month`.
- Enforce a coarse check in SQLite so direct writes cannot store garbage. It
  does not know month lengths, so the import is the strict gate.
- The month is derived with `SUBSTR(date, 1, 7)` in views and queries; it is
  never stored a second time.

```sql
CHECK (
    date GLOB '2[0-1][0-9][0-9]-[0-1][0-9]-[0-3][0-9]'
    AND CAST(SUBSTR(date, 6, 2) AS INTEGER) BETWEEN 1 AND 12
    AND CAST(SUBSTR(date, 9, 2) AS INTEGER) BETWEEN 1 AND 31
)
```

Canonical `YYYY-MM-DD` values sort chronologically as text. To filter a month
use a range (`date >= '2026-03-01' AND date <= '2026-03-31'`) so the index on
`date` is used.

### Money Values

Monetary amounts are stored as `INTEGER` cents in columns suffixed `_cents`
(`amount_cents`, `target_cents`, `principal_cents`). Never store euros in
`REAL`. Go code uses `money.Cents`. CSV fields suffixed `_cents` and JSON API
properties suffixed `Cents` carry raw integer cents (`1234` means EUR 12.34).
Views expose `*_cents` columns too ([ADR 002](../decisions/002-money-as-integer-cents.md)). Splitting a total over several periods uses
`money.Split`, which spreads leftover cents one each over the first periods so
the parts sum exactly to the total.

## Constraints

- Primary keys: `id INTEGER PRIMARY KEY AUTOINCREMENT` for auto-generated IDs, composite `PRIMARY KEY (col1, col2)` for junction tables.
- Foreign keys: Always declared with `FOREIGN KEY (col) REFERENCES table(id)`.
  - Use `ON DELETE CASCADE` for child data that should be removed with its parent (goals, movements, payments).
  - Use `ON DELETE RESTRICT` for references that must not be orphaned.
- Unique constraints: Declared inline or as separate `UNIQUE(col1, col2)`.
- Partial unique indexes: Used for conditional uniqueness (e.g. one allocation per goal/month or one regular payment per contract/month).
- CHECK constraints: Used to enforce valid values at the database level:
  - Enum-like columns: `CHECK (status IN ('active', 'completed', 'cancelled'))`.
  - Positive amounts: `CHECK (amount > 0)`.
  - Month validation: use the canonical format and numeric range check from
    [Month Values](#month-values).
  - Cross-column rules: `CHECK (end_month >= start_month)`.
  - Conditional nullability: `CHECK ((type = 'deposit' AND goal_id IS NULL) OR (type != 'deposit' AND goal_id IS NOT NULL))`.

## Data Types

| Type     | Usage                                    |
|----------|------------------------------------------|
| INTEGER  | IDs, foreign keys, counts, and monetary amounts in cents (`*_cents`) |
| TEXT     | Names, usernames, status, months, dates  |
| DATETIME | Timestamps (DEFAULT CURRENT_TIMESTAMP)   |

## Indexes

- Unique indexes are created for business rules (e.g. one allocation per goal per month).
- Partial indexes use `WHERE` clauses for conditional uniqueness.
- Index naming: `idx_<table>_<description>` (e.g. `idx_goal_account_movements_allocation_unique`).
- Don't add an index whose columns are a prefix of an existing index or 
UNIQUE constraint; SQLite already uses that one.

## CSV File Naming

A CSV seed file is named after its table (`goal_accounts.csv` for
`goal_accounts`). The files of each module are listed in its `csv.md`.

## Migrations

- Migrations handle schema only. No seed data in migrations.
- Migration files: `NNNN_description.sql` in `internal/database/migrations/sql/`.
- A shipped migration is never edited: a change is a new file that also moves the existing values ([ADR 003](../decisions/003-fix-forward-migrations.md)).
- There are no down migrations.

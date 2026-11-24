# Investments Module Schema

Defined in `internal/database/migrations/sql/0004_create_investment_module.sql`;
the column lists live there. Shared conventions:
[database.md](../../architecture/database.md).

## Rules enforced by the database

- Entity names are unique; instrument names are unique within an entity.
- Entity kinds and instrument types are limited to the values in
  [overview.md](overview.md).
- An entity's `user_id` is optional (NULL means the household) and becomes
  NULL if that user is ever deleted. The instruments of an entity take its
  owner; they have no owner of their own.
- Contribution amounts are greater than zero; snapshot values are zero or more.
- Months are canonical `YYYY-MM`.
- There is deliberately no uniqueness on (instrument, month) for
  contributions or snapshots.

## Summary views

`investment_instrument_summary_view` returns one row per instrument with the
contribution and snapshot counts, allocated amount, current value (latest
snapshot), last snapshot month, gain and `gain_pct` (scaled by 100). Only the
contributions are joined; every snapshot figure is a correlated subquery so the
two child tables cannot multiply each other's rows.

`investment_entity_summary_view` aggregates the instrument view per entity.
Nothing is stored or synced.

## Relationships

```text
investment_entities
  └─ investment_instruments (ON DELETE CASCADE)
       ├─ investment_contributions (ON DELETE CASCADE)
       └─ investment_snapshots (ON DELETE CASCADE)
```

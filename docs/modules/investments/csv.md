# Investments CSV Files

Generic seed, export and sync behavior is in
[csv-seed.md](../../architecture/csv-seed.md). Child files identify their
parents by name. Money fields contain raw integer cents.

## investment_entities.csv

| Column | Description                                                          |
|--------|----------------------------------------------------------------------|
| name   | Entity name (unique)                                                 |
| kind   | `bank`, `broker`, `investment_firm`, `state` or `other`              |
| user   | Optional username from `users.csv`; empty means the household owns it |

## investment_instruments.csv

| Column | Description                                                                    |
|--------|--------------------------------------------------------------------------------|
| entity | Entity name                                                                    |
| name   | Instrument name (unique per entity)                                            |
| type   | `etf`, `fund`, `ppr`, `certificados_aforro` or `other`                         |
| notes  | Optional notes                                                                 |

## investment_contributions.csv

| Column       | Description                                |
|--------------|--------------------------------------------|
| entity       | Entity name                                |
| instrument   | Instrument name within that entity         |
| month        | Contribution month (YYYY-MM)               |
| amount_cents | Amount added, positive, in raw integer cents |
| description  | Optional description                       |
| movement     | Optional: the `sync_id` of the cashflow movement this row was added with ([ADR 021](../../decisions/021-details-of-money-moved.md)); empty for rows typed by hand |

## investment_snapshots.csv

| Column      | Description                                  |
|-------------|----------------------------------------------|
| entity      | Entity name                                  |
| instrument  | Instrument name within that entity           |
| month       | Month of the observed value (YYYY-MM)        |
| value_cents | Observed value, zero or more, in raw integer cents |
| description | Optional description                         |

Both child files carry the entity as well as the instrument, because
instrument names are unique only within an entity.

## Import behavior

The import mirrors each file (see [csv-seed.md](../../architecture/csv-seed.md)).
All four files are applied in one transaction, in the order above.

- **Entities** and **instruments**: upserted by name (instruments by entity and
  name), updated only when a column changed; rows not in the file are removed,
  and removing an entity removes its instruments and their rows.
- **Contributions** and **snapshots**: for each instrument, replaced only when
  its rows differ from the database. An instrument with no rows in the file
  ends up with none. Duplicate rows are kept, as several in a month are valid.
- An unknown entity, instrument or user, an unknown kind or type, a bad month or
  non-integer cents rejects the import.

A file containing only a header clears its table. Sync does nothing: figures
are derived by views.

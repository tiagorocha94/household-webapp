# Cashflow CSV Files

Four files, imported in this order: the budget groups, the categories, the
movements and the balances. Generic seed, export and sync behavior is in
[csv-seed.md](../../architecture/csv-seed.md). Movements identify their user and
category by name. Money fields contain raw integer cents. This module imports
after `users.csv`, because every movement names a user.

## cashflow_budget_groups.csv

Optional groups for budgeting ([budgets.md](budgets.md)). Without any rows (a
header-only or missing file) the feature is off.

| Column    | Description                                                        |
|-----------|--------------------------------------------------------------------|
| name      | Group name (unique, case sensitive), such as `Essential`           |
| share_pct | Share of the month's income, a whole percent from 1 to 100         |

The shares must add up to exactly 100. The order of the rows is kept and
exported back the same way; it is the order the groups are shown in.

## cashflow_categories.csv

| Column    | Description                                                        |
|-----------|--------------------------------------------------------------------|
| name      | Category name (unique)                                             |
| direction | `income` or `expense`                                              |
| icon      | [Lucide](https://lucide.dev) icon name in kebab-case, such as `shopping-basket` |
| color     | Colour as `#RRGGBB`, such as `#65A30D`                             |
| budget_group | Name of the group in `cashflow_budget_groups.csv`, or empty    |
| role      | Empty, or where an expense category's money goes: `savings`, `investments`, `goals` or `credits` |

All six columns are required (the file must have exactly this header; an older
five-column file is rejected: add a `role` column). `budget_group` is empty
for income categories. Only an expense category can have a role. A movement in a role category stays an expense, so budgets, totals and the savings rate do not
change; for `savings` the month detail adds them up as `savedCents` ("of which saved")
([ADR 014](../../decisions/014-savings-module.md)).
While at least one group exists, every expense category must name one. The interface draws each category with its icon
on its colour. The mobile cashflow app keeps the same list, with the same
names, directions, icons and colours; keep the two identical.

## cashflow_movements.csv

| Column       | Description                                          |
|--------------|------------------------------------------------------|
| user         | Username from `users.csv`                            |
| category     | Category name from `cashflow_categories.csv`         |
| date         | Day of the movement (YYYY-MM-DD, a real calendar day) |
| amount_cents | Amount, positive, in raw integer cents               |
| description  | Optional description                                 |
| sync_id      | Lowercase UUID naming the movement, unique in the file |

All six columns are required (the file must have exactly this header). The
`sync_id` is made by the device that recorded the movement and never changes, so
a retried or edited send finds the same movement
([ADR 011](../../decisions/011-devices-write-cashflow-movements.md)). A file
without it, such as an older five-column file, is rejected.

Rows may repeat exactly (apart from the `sync_id`); each copy is a separate movement. A refund is an
`income` row in a suitable category, not a negative amount.

## Import behavior

The import mirrors each file (see [csv-seed.md](../../architecture/csv-seed.md)).
Both files are applied in one transaction, in the order above.

- **Budget groups**: upserted by name, updated only when the share or the
  position changed; a group that left the file is removed after the categories
  are updated, so its categories end up ungrouped (and the import is rejected
  if that leaves an expense category without a group while groups remain).
- **Categories**: upserted by name, updated only when the direction, icon,
  colour or group changed; categories not in the file are removed, and removing a
  category removes its movements.
- **Movements**: for each user, replaced only when that user's rows differ from
  the database, `sync_id` included. A user with no rows in the file ends up with
  no movements. A `sync_id` may move from one person to another in the file. All
  rows that change are cleared before any is inserted.
- **When the file is behind**: once devices can change movements, the database
  can hold changes the file does not have yet. The database counts every change
  (`revision`) and records the count the file last held (`csv_revision`) in
  `cashflow_csv_state`. While `revision` is higher, start-up does not import the
  movements file, because that would undo the changes; the export that follows
  rewrites the file from the database and brings the two counts level. A write
  that fails halfway therefore never loses data. Editing the file by hand while
  it is behind has no effect, so stop the server, let it start once, then edit.
- A group share that is not a whole number from 1 to 100, shares that do not
  add up to 100, a repeated group name, an unknown `budget_group`, a group on an
  income category, or an expense category without a group while groups exist
  rejects the import with the row number or the category names.
- A missing, malformed or repeated `sync_id`, an unknown user or category, an unknown direction, an icon that is not a
  lowercase kebab-case name, a colour that is not `#RRGGBB`, a duplicate
  category, a
  date that is not a real `YYYY-MM-DD` day (such as `2026-02-30` or
  `2026-1-5`), or an amount that is zero, negative or not an integer rejects
  the import with the row number.

A file containing only a header clears its table. Export writes names instead
of ids, groups in their file order, categories ordered by name and movements ordered by date, user,
category and creation, so unchanged data exports identically. Sync does
nothing: figures are derived by views. While the server runs, the
`CSVMirror` rewrites the movements file after devices change something
([sync](sync.md)); it is the one piece to remove when the file is retired.

# Cashflow Module Schema

Defined in `internal/database/migrations/sql/0005_create_cashflow_module.sql`,
`0006_cashflow_budget_groups.sql` and `0013_cashflow_category_role.sql` (which
replaced the savings and investments flags added by `0009` and `0010`) and
`0014_cashflow_role_credits.sql`; the
column lists live there. Shared conventions:
[database.md](../../architecture/database.md).

## Rules enforced by the database

- Category names are unique; directions are `income` or `expense`. Every category has an `icon` (lowercase kebab-case, letters, digits and hyphens) and a `color` (`#RRGGBB`).
- Budget group names are unique, a share is a whole percent from 1 to 100 and
  `position` keeps the order of the file ([budgets.md](budgets.md)). A category
  may name one group (`budget_group_id`); removing a group leaves its
  categories ungrouped. That the shares add up to 100, and that every expense
  category has a group while groups exist, is checked by the CSV import, not
  by the database.
- Movement amounts are greater than zero.
- A movement's `date` is a canonical `YYYY-MM-DD` day, checked coarsely (see
  [Date Values](../../architecture/database.md#date-values)). The database does
  not know month lengths; the CSV import rejects impossible days such as
  `2026-02-30`.
- Every movement has a `sync_id`, a 36-character UUID that is unique. Nothing
  else about a movement is unique: identical rows may repeat, each with its own
  `sync_id`.
- `cashflow_csv_state` is one row of bookkeeping (`revision`, `csv_revision`)
  that tells whether `cashflow_movements.csv` is behind the database. It is
  created the first time it is needed and is not exported.
- The month is never stored. Views and queries derive it with
  `SUBSTR(date, 1, 7)`, and month filters use a date range so the index on
  `date` is used.

## Summary views

- `cashflow_month_summary_view`: one row per month that has movements, with
  income, expenses, net, `savings_rate_pct` (scaled by 100, 0 without income)
  and the movement count.
- `cashflow_month_category_view`: the total and movement count per category in
  a month.
- `cashflow_month_user_view`: income, expenses and net per person in a month.

Nothing is stored or synced.

`cashflow_balances` holds the typed bank balances (`user_id`, `month`,
`balance_cents`, unique per person and month). It is the only table here that
holds a figure someone typed besides the movements. A person's cash in any month
is derived from it on request and never stored. Without one it starts from zero
at the person's first movement.

## Relationships

```text
users
  ├─ cashflow_movements (ON DELETE CASCADE)
  └─ cashflow_balances (ON DELETE CASCADE)
cashflow_budget_groups
  └─ cashflow_categories (ON DELETE SET NULL)
cashflow_categories
  └─ cashflow_movements (ON DELETE CASCADE)
```

`role` is empty or `savings`, `investments`, `goals` or `credits`: where an expense category's money goes (ADR 014). Home compares each role with what was typed in that module.

Removing a category from `cashflow_categories.csv` removes its movements.

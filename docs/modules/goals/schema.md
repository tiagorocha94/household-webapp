# Goals Module Schema

Defined in `internal/database/migrations/sql/0002_create_goal_module.sql`;
the column lists live there. Shared conventions:
[database.md](../../architecture/database.md).

## Rules enforced by the database

- Account names are unique, goal names unique per account.
- An account's `monthly_deposit_cents` cannot be negative.
- A goal's `end_month` cannot precede its `start_month`.
- Deposits have no goal; allocations, expenses, releases and covers require one.
- One allocation per goal per month (partial unique index).
- Deleting an account removes its goals and movements.

## Views

- `goal_account_balance_view`: one row per account. Formulas are in the
  [overview](overview.md#balance).
- `goal_account_goal_summary_view`: one row per goal with the month count and
  totals. `monthly_allocation_cents` is `ceil(target / months)`, a display
  value; actual allocations come from `money.Split`. Elapsed months are not in
  the view because they depend on the current month; the goals service
  computes them in Go in the local time zone, like sync and the projection.

## Relationships

```text
goal_accounts
  ├─ goal_account_goals (ON DELETE CASCADE)
  │    └─ goal_account_movements (goal_id, ON DELETE CASCADE)
  └─ goal_account_movements (account_id, ON DELETE CASCADE)
```

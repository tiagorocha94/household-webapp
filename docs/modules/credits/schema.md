# Credits Module Schema

Defined in `internal/database/migrations/sql/0003_create_credit_module.sql` and
`0011_credit_expected_day.sql`; the column lists live there. Shared conventions:
[database.md](../../architecture/database.md).

## Rules enforced by the database

- Contract names are unique, as is (lender, contract number).
- `expected_end_month` cannot precede `start_month`.
- `expected_day` is empty or 1 to 28.
- Each payment has principal plus interest greater than zero.
- One `regular` payment per contract and month (partial unique index);
  `extraordinary` payments are unlimited.

The CSV import adds a rule the database cannot express: cumulative principal
cannot exceed the original principal.

## Summary view

`credit_contract_summary_view` returns one row per contract with payment
count, last payment month, principal, interest and total paid, outstanding
principal, progress (capped at 100%) and `status` (`paid_off` once no
principal is outstanding, otherwise `active`). Nothing is stored or synced.

## Relationships

```text
credit_contracts
  └─ credit_payments (ON DELETE CASCADE)
```

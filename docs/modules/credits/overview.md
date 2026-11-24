# Credits

## Overview

The credits module records the complete history of household credit contracts,
including mortgages, car credit, personal credit, and other loans. Each payment
is split into principal and interest so the outstanding principal and total
cost remain visible. The module is read-only through the [JSON API](../../architecture/openapi.yaml);
data is maintained in the CSV files ([csv.md](csv.md)).

## Contracts

A credit contract contains:

- A user-defined name and one of four categories: `mortgage`, `car`,
  `personal`, or `other`.
- Lender and contract number.
- Original principal and start month.
- An optional expected end month and notes.
- A derived status (see Balances and Status)

Contract names are unique. A lender and contract-number combination can only
be used once. The database stores stable category
values; icons are a client concern.

## Payments

Payments use month-level history and contain separate principal and interest
amounts. Their total is derived rather than stored.

| Type | Description |
|------|-------------|
| `regular` | The normal monthly instalment. Only one is allowed per contract and month. |
| `extraordinary` | An additional principal payment. Multiple entries may exist in one month. |

Payment rules, enforced by the CSV import and the database:

- A payment cannot precede the contract start month.
- Principal and interest are non-negative and their combined total is positive.
- Cumulative principal cannot exceed the original principal.
- Only one regular payment per contract and month.

## Balances and Status

All totals are derived from payment history:

```text
outstanding principal = original principal - principal paid
total paid            = principal paid + interest paid
progress              = principal paid / original principal
```

The status is derived, never stored: `paid_off` when outstanding principal is 
zero, otherwise `active`. Correcting the payment history changes it
immediately. Payments are actual transactions and are never generated.

## Access

Web sessions and device tokens can read every contract and payment
([ADRs 010](../../decisions/010-household-data-is-shared.md) and
[022](../../decisions/022-devices-read-savings-investments-credits.md)).
Enrolled phones can create, replace or remove a payment only as the detail of
one of their cashflow movements; other changes use the CSV files
([ADR 021](../../decisions/021-details-of-money-moved.md)).

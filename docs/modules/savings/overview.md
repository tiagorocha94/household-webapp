# Savings

Savings are money kept accessible, such as an emergency fund. They form a
separate module and never count as investments
([ADR 014](../../decisions/014-savings-module.md)).

## What it records

- **Accounts:** name, optional owner, optional target and notes. An empty owner
  means the household.
- **Movements:** a monthly `deposit` or `withdrawal`.
- **Snapshots:** the observed balance in a month.

The CSV formats are in [csv.md](csv.md).

## Figures

| Figure | Rule |
|--------|------|
| Deposited, withdrawn | Sum of movements of each type |
| Balance | Latest snapshot by month and id; deposits less withdrawals before the first snapshot |
| Interest earned | Balance less deposits plus withdrawals; zero before the first snapshot |
| Progress | Balance as a whole percent of the target; zero without a target and allowed above 100 |

A target is only an amount to reach: savings have no end date or monthly plan.

## Access and connections

The read API accepts a web session or a device token
([ADR 022](../../decisions/022-devices-read-savings-investments-credits.md));
the exact routes are in [OpenAPI](../../architecture/openapi.yaml).

Data can be typed in the CSV files or added as the detail of a cashflow
movement sent by an enrolled phone
([ADR 021](../../decisions/021-details-of-money-moved.md)). A cashflow movement
in the `savings` role remains an expense and contributes to “of which saved.”
Home warns when that cashflow total exceeds the recorded deposits.

The current savings balance contributes to household net worth.
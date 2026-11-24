# Credits CSV Files

Generic seed, export and sync behavior is in
[csv-seed.md](../../architecture/csv-seed.md). The payments file identifies a
contract by its credit name. Money fields contain raw integer cents.

## credit_contracts.csv

| Column             | Description                                      |
|--------------------|--------------------------------------------------|
| name               | Credit name (unique)                             |
| category           | `mortgage`, `car`, `personal`, or `other`       |
| contract_number    | Lender's contract number                         |
| lender             | Bank or finance provider                         |
| original_principal_cents | Original borrowed principal in raw integer cents |
| start_month        | First contract month (YYYY-MM)                   |
| expected_end_month | Optional expected final month (YYYY-MM)          |
| notes              | Optional notes                                   |
| expected_day       | Optional day of the month (1 to 28) by which the regular payment should be recorded; Home warns when it passes without one |

## credit_payments.csv

| Column           | Description                           |
|------------------|---------------------------------------|
| credit           | Credit name                           |
| month            | Payment month (YYYY-MM)               |
| type             | `regular` or `extraordinary`          |
| principal_cents | Raw integer cents reducing outstanding principal |
| interest_cents  | Interest paid in raw integer cents                |
| description      | Optional description                  |
| movement         | Optional: the `sync_id` of the cashflow movement this row was added with ([ADR 021](../../decisions/021-details-of-money-moved.md)); empty for rows typed by hand |

## Import behavior

The import mirrors each file (see [csv-seed.md](../../architecture/csv-seed.md)).
Both files are applied in one transaction.

- **Contracts**: upserted by name, updated only when a column changed;
  contracts not in the file are removed. Status is not in the CSV.
- **Payments**: for each contract, replaced only when its rows differ from the
  database. A contract with no rows in the file ends up with no payments.
  Invalid rows, or total principal above the original principal, reject the
  import.

A file containing only a header clears its table.

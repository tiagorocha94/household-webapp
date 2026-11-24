# Savings CSV files

All files are optional and header-checked like the other modules. Amounts are
raw integer cents. A file missing from the seed folder leaves its table as it
is; a file present is mirrored exactly (rows not listed are removed).

## savings_accounts.csv

| Column       | Description                                                     |
|--------------|-----------------------------------------------------------------|
| name         | Unique account name                                             |
| owner        | Username from `users.csv`, or empty for the household           |
| target_cents | Positive amount to reach, or empty for no target                |
| notes        | Free text                                                       |

## savings_movements.csv

| Column       | Description                                                     |
|--------------|-----------------------------------------------------------------|
| account      | Account name                                                    |
| month        | `YYYY-MM`                                                       |
| type         | `deposit` or `withdrawal`                                       |
| amount_cents | Positive                                                        |
| description  | Free text                                                       |
| movement     | Optional: the `sync_id` of the cashflow movement this row was added with ([ADR 021](../../decisions/021-details-of-money-moved.md)); empty for rows typed by hand |

Several movements may share a month. An account with no rows ends up with no
movements.

## savings_snapshots.csv

| Column      | Description                  |
|-------------|------------------------------|
| account     | Account name                 |
| month       | `YYYY-MM`                    |
| value_cents | Balance seen, zero or more   |
| description | Free text                    |

Moving an instrument such as Certificados de Aforro from the investments files
is a manual edit: add the account here with its movements and snapshots, then
remove the instrument from the investments files.

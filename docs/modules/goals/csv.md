# Goals CSV Files

Generic seed, export and sync behavior is in
[csv-seed.md](../../architecture/csv-seed.md). Money fields contain raw integer
cents. Months are `YYYY-MM`.

## goal_accounts.csv

| Column                | Description                                              |
|-----------------------|----------------------------------------------------------|
| name                  | Account name (unique)                                    |
| monthly_deposit_cents | Deposit the projection assumes per future month, in raw integer cents; 0 means none |

## goal_account_goals.csv

| Column        | Description                              |
|---------------|------------------------------------------|
| account       | Account name                             |
| name          | Goal name (unique per account)           |
| target_cents  | Target amount in raw integer cents       |
| start_month   | Start month                              |
| end_month     | End month                                |
| status        | `active`, `completed`, or `cancelled`    |

## goal_account_movements.csv

| Column      | Description                                       |
|-------------|---------------------------------------------------|
| account     | Account name                                      |
| type        | `deposit`, `allocation`, `expense`, `release` or `cover` (`refund` is read as `release`) |
| goal        | Goal name (empty for deposits)                    |
| month       | Month                                             |
| amount_cents | Amount in raw integer cents (always positive)   |
| description | Optional description                              |

## Import behavior

The import mirrors each file (see [csv-seed.md](../../architecture/csv-seed.md)).
All three files are applied in one transaction.

- **Accounts**: upserted by name, updated only when the deposit changed;
  accounts not in the file are removed.
- **Goals**: upserted by (account, name), updated only when a column changed;
  goals not in the file are removed.
- **Movements**: for each account, replaced only when its rows differ from the
  database. An account with no rows in the file ends up with no movements.

Rows are validated before they are written: months must be `YYYY-MM`, a goal's
end cannot precede its start, deposits take no goal and every other movement
requires one, and a monthly deposit cannot be negative. Account names must be
unique, because the other files refer to accounts by name.

## Export behavior

Rows are ordered by account, month, type, goal name and id, so repeated
exports of unchanged data are identical.

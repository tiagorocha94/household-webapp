# Cashflow

## Overview

The cashflow module is a household ledger of money coming in and going out, per
person and per month. There is no bank or app integration: the CSV files
([csv.md](csv.md)) are produced elsewhere, from the household's tracking app
and bank exports, and this repo only defines their contract. It answers three
questions: how much came in, went out and was left each month (for everyone
and per person), where the money went by category in a month, and which
individual movements are behind those totals. The module is read-only through
the [JSON API](../../architecture/openapi.yaml) for browsers; enrolled phones
change movements through the [sync endpoint](sync.md). Currency is EUR only.

Standing notes:

- **Data is prepared outside this repo.** A separate tool converts bank and app
  exports and writes the CSV files in this module's format. The CSV format is
  the contract, so it follows what this app wants, not the shape of any export,
  and the server never imports anything else.
- **The CSV is a mirror kept for consistency with the other files**, not a way
  to add data in normal use. Movements arrive from the phones.

## Structure

```text
users (auth module)
  └─ movement   one row per movement, owned by a user
category        the fixed set of categories, each income or expense
  └─ movement
```

- **Category**: a name, a direction (`income` or `expense`), a Lucide icon
  name and a colour. The set is fixed and kept in `cashflow_categories.csv`; it
  is a flat list; an optional `budget_group` names the group a category belongs
  to for budgeting ([budgets.md](budgets.md)), and a `role` (`savings`, `investments`, `goals` or `credits`) says where money put aside goes
  (shown as "of which saved", [ADR 014](../../decisions/014-savings-module.md)). Whoever produces the CSV maps their own
  categories to this set. The icon and colour only say how the interface draws
  the category; they never affect the figures.
- **Movement**: a user, a category, a full date (`YYYY-MM-DD`), a positive
  amount, an optional description and a `sync_id` (a UUID the recording
  device makes, so a movement is found again when it is re-sent). The user is a grouping label only: every
  authenticated user sees every movement. Identical rows may repeat (several
  movements in a day are valid, each with its own `sync_id`) and every copy counts.

Amounts are always positive. A refund is an `income` movement in a suitable
category (such as `Others`), never a negative expense.

## Figures

All figures are derived from the movements by views and never stored:

```text
income       = sum of movements whose category is income
expenses     = sum of movements whose category is expense
net          = income - expenses            (may be negative)
savings rate = net * 10000 / income, as an integer; 0 when income is 0
```

`savingsRatePct` is an integer scaled by 100: `2500` means 25.00% and `-8000`
means -80.00%. Clients divide by 100 to display it. The name is the common
term for the ratio; it is not a savings balance. A month's figures count the
movements dated in that calendar month. Month-over-month change is computed by
the frontend from the monthly list.

## Boundaries

- Money moved into savings, investments, goals or credits remains an expense
  here. Its optional destination detail is recorded in the relevant module.
- There is no bank integration or multi-account ledger. One typed balance per
  person anchors the derived cash figure.
- Recurring rules live on the phone; the server receives the ordinary movements
  they create.
- Net worth belongs to [Home](../home/overview.md), and budget rules are in
  [budgets.md](budgets.md).

## API shape

- `/cashflow/categories`: the category set.
- `/cashflow/budget/{month}`: the month's totals, budget groups and the months
  that have movements, for the phone ([ADR 015](../../decisions/015-devices-read-budget.md)).
  It accepts a device token, carries no people or movements, and answers an empty
  month with zeros instead of a 404.
- `/cashflow/balance/{month}`: the caller's own cash at the end of the month, for
  the phone ([ADR 019](../../decisions/019-devices-read-their-own-balance.md)).
  It accepts a device token and never returns anyone else's balance;
  `balanceCents` is null only when the person has neither a typed balance nor a movement.
- `/cashflow/months`: one summary per month that has movements, newest first.
- `/cashflow/months/{month}`: one month's summary plus `byCategory`,
  `byUser` (each person's income, expenses, `spentCents`, `putAsideCents` and
  `cashCents`, counted from zero at the first movement when no bank balance was typed, null only without any movement), `peopleHistory` (income and
  money put aside per person for the month and the five before it) and `byGroup`
  (the budget by group, see [budgets.md](budgets.md); empty without groups).
  Put aside is the expenses of the savings, investments and goals categories;
  credit payments count as spent, since the asset they pay for is not recorded. A month with no movements is a 404; a malformed month is a 400.
- `/cashflow/months/{month}/movements`: the month's movements, newest date
  first. A malformed month is a 400; a month with no movements is an empty
  list, not a 404.

The month is a filter rather than a stored parent, which is why the detail
endpoint and the list endpoint differ for an empty month.

Any authenticated user can read everything
([ADR 010](../../decisions/010-household-data-is-shared.md)). Browsers cannot
change anything; enrolled phones change movements through the
[sync endpoint](sync.md).

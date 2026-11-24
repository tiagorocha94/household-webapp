# Home overview

## Overview

The Home page summarises every module. The server composes it in one read-only
call ([openapi.yaml](../../architecture/openapi.yaml), `GET /api/v1/overview`)
so the page makes a single request and the figures are computed in one place.
The `overview` package owns no tables and no CSV files: on every request it
reads the goals, credits, investments, savings and cashflow services, so nothing can
drift from the modules it summarises. It needs a web session; device tokens
cannot read it ([ADR 012](../../decisions/012-devices-read-goals.md) covers
only the goals routes).

## When a module does not run

The overview is the one place that reads every module, so it is built to work
with any subset (see `modules` in the [README](../../../README.md#configuration)).
A module that does not run is not read: its section of the response
(`goals`, `credit`, `investments`, `savings`, `cashflow`) is left out, its rows
and warnings are missing, and net worth adds up what is left. The checks that
compare cashflow with another module (the `unassigned_` warnings) need both
modules to run. The net worth history takes its months from cashflow, so
without it the list is empty. The web Home and Net worth pages show only what
the response holds.

Net worth month by month is a second read, `GET /api/v1/overview/net-worth`. For
every month that has cashflow it returns the savings, investments, goals, cash
and credit owed at the end of the month, worked out from the modules' typed data
on each request and stored nowhere: savings from the movements and snapshots up
to the month, investments from each instrument's latest snapshot up to it, goals
from the active goals' movements up to it, cash as described in
[ADR 018](../../decisions/018-cash-balance-from-typed-balances.md), and the credit
owed as the original principal less the principal paid up to the month, for the
contracts that had started.

## What it contains

| Section | Content |
|---------|---------|
| `attention` | Things to look at, each with a kind (below). Empty when there are none. The web app shows them on the Notifications page and counts them on Home and on the bell. |
| `goals` | Saved and target totals of the active goals, and `activeGoals`, every active goal with its progress, the soonest end first. |
| `credit` | Outstanding principal and the monthly instalments of the active contracts, and `contracts`, each active contract. |
| `investments` | The current value, and `holdings`, the value at each entity. |
| `savings` | The total balance of the savings accounts, the targets of those that have one added up, the number of accounts, the interest earned and what was withdrawn for emergencies, and `items`, each account with its owner, balance, target, progress and interest. The Home tile and section show only when there is an account. |
| `netWorth` | What the household has minus what it owes: `assetsCents` (the savings balance, the investments' current value, the money saved in the active goals, which is counted once, and the cash), `cashCents` and `cashByUser` (each person's cash this month, empty when nobody has a typed bank balance or a movement, see [cash balances](../cashflow/csv.md#cashflow_balancescsv)), `creditCents` (the outstanding principal of the active contracts) and `netCents`, the difference, which can be negative. Shown as the first tile on Home. |
| `cashflow` | The current month's income, expenses (and `savedCents`, the part in savings categories), net and savings rate, the previous month's expenses at the same point, and the budget groups. |

## Rules

- **Goals shortfall**: for each goal account, the first future month in its
  [projection](../goals/overview.md) whose unallocated amount is negative. The
  entry names the account, the month and the shortfall.
- **Budget over**: a budget group of the current month that spent more than
  its budget. The entry names the group, the amount over and the whole percent
  of the budget spent. Groups without a budget (a month without income) never
  count.
- **Unassigned money**: for the current month and the two before, the expenses
  of the categories with the role `savings`, `investments` or `goals` in
  `cashflow_categories.csv` minus what was typed for that month: the deposits in
  Savings, the contributions in Investments, or the deposits in Goals. A
  positive difference is an entry (`unassigned_savings`,
  `unassigned_investments` or `unassigned_goals`) with the month and the amount.
  More typed than spent is not flagged. Without a category of a role, that role
  is not checked.
- **Unassigned credits**: for the same months, expenses in the categories with
  the role `credits` and no credit payment of any contract recorded for the
  month is an `unassigned_credits` entry with the month and the cashflow
  amount. Only whether a payment exists is checked, not the amount: a cashflow
  instalment often includes insurance and fees that a credit payment does not.
- **Credit unpaid**: an active credit with an `expected_day` whose day of the
  month has passed without a regular payment recorded for the current month.
  The entry names the credit and the month. A credit without an expected day
  is never checked.
- **Active goals** are the goals with status active, including ones that have not started yet.
- **Behind plan**: a goal is behind when less was allocated than an even split
  of its target over its months would have by the current month. This is the
  rule the phone uses.
- **Next instalment**: payments are recorded by month, not by day. The next
  instalment of a contract is the month after its last regular payment, and the
  amount is that payment's total. It is an estimate: a contract without a
  regular payment has none, and an extraordinary payment is not an instalment.
- **Previous month at the same point**: the previous month's expenses up to the
  same day of the month as today, so a month part way through is compared fairly.
- A month without movements gives zero cashflow figures, not an error.
